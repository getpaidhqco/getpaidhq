package config

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/thecodearcher/limen"
	gormadapter "github.com/thecodearcher/limen/adapters/gorm"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"
	organization "github.com/thecodearcher/limen/plugins/organization"
	"gorm.io/gorm"

	"getpaidhq/internal/adapter/limenauth"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// buildLimen constructs the limen auth instance mounted at /api/auth.
//
// The organization plugin is configured per the delegated-authz design:
// Cedar (via limenauth.NewOrganizationAuthorizer) decides every privileged
// action, invitation delivery is a callback (logged until the app has an
// email transport), and lifecycle events are emitted app-side through limen
// after-hooks onto NATS rather than baked into the plugin.
func buildLimen(env lib.Env, baseURL string, logger port.Logger, operationalDB any, authz port.Authz, publisher port.PubSub) (*limen.Limen, error) {
	adapter, err := limenDatabaseAdapter(operationalDB)
	if err != nil {
		return nil, err
	}

	orgPlugin := organization.New(
		organization.WithAuthorize(limenauth.NewOrganizationAuthorizer(authz, logger)),
		organization.WithSendInvitationEmail(func(msg organization.InvitationMessage) {
			// No email transport exists in the app yet (dunning models channels
			// but ships no sender). Log so the flow is observable end-to-end;
			// swap for the real sender when one lands.
			logger.Infof("limen invitation: org=%s email=%s expires=%s token=%s",
				msg.Organization.Slug, msg.Invitation.Email, msg.Invitation.ExpiresAt, msg.Token)
		}),
	)

	return limen.New(&limen.Config{
		BaseURL:  baseURL,
		Database: adapter,
		Secret:   []byte(env.LimenSecret),
		// Serialize discovered schemas to .limen/schemas.json on boot so the
		// limen CLI can generate SQL migrations for limen's tables (users,
		// sessions, organizations, ...), which live outside the goose-managed
		// app schema: go run github.com/thecodearcher/limen/cmd/limen generate migrations
		CLI: &limen.CLIConfig{Enabled: true},
		Plugins: []limen.Plugin{
			credentialpassword.New(),
			orgPlugin,
		},
		HTTP: limen.NewDefaultHTTPConfig(
			limen.WithHTTPBasePath("/api/auth"),
			limen.WithHTTPHooks(limenauth.EventHooks(publisher, logger)),
		),
	})
}

// limenDatabaseAdapter wraps the operational DB handle (selected by DB_DRIVER)
// in the matching limen adapter: gorm directly, pgx through its database/sql
// compatibility layer.
func limenDatabaseAdapter(operationalDB any) (limen.DatabaseAdapter, error) {
	switch db := operationalDB.(type) {
	case *gorm.DB:
		return gormadapter.New(db), nil
	case *pgxpool.Pool:
		return sqladapter.NewPostgreSQL(stdlib.OpenDBFromPool(db)), nil
	default:
		return nil, fmt.Errorf("limen: unsupported operational DB handle %T", operationalDB)
	}
}
