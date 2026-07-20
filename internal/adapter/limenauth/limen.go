package limenauth

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/thecodearcher/limen"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"
	organization "github.com/thecodearcher/limen/plugins/organization"

	"getpaidhq/internal/core/port"
)

// Build constructs the limen auth instance mounted at /api/auth.
//
// The organization plugin is configured per the delegated-authz design:
// Cedar (via NewOrganizationAuthorizer) decides every privileged action,
// invitation delivery is a callback (logged until the app has an email
// transport), and lifecycle events are emitted app-side through limen
// after-hooks onto NATS rather than baked into the plugin.
func Build(secret, baseURL string, logger port.Logger, operationalDB any, authz port.Authz, publisher port.PubSub) (*limen.Limen, error) {
	adapter, err := databaseAdapter(operationalDB)
	if err != nil {
		return nil, err
	}

	orgPlugin := organization.New(
		organization.WithAuthorize(NewOrganizationAuthorizer(authz, logger)),
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
		Secret:   []byte(secret),
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
			limen.WithHTTPHooks(EventHooks(publisher, logger)),
		),
	})
}

// databaseAdapter wraps the pgx pool in limen's sql adapter through pgx's
// database/sql compatibility layer. Limen intentionally supports only the
// pgx driver here: booting with LIMEN_SECRET set requires DB_DRIVER=pgx.
func databaseAdapter(operationalDB any) (limen.DatabaseAdapter, error) {
	pool, ok := operationalDB.(*pgxpool.Pool)
	if !ok {
		return nil, fmt.Errorf("limen requires DB_DRIVER=pgx, got operational DB handle %T", operationalDB)
	}
	return sqladapter.NewPostgreSQL(stdlib.OpenDBFromPool(pool)), nil
}
