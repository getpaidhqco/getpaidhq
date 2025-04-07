package postgres

import (
	"context"
	cdc "github.com/Trendyol/go-pq-cdc"
	"github.com/Trendyol/go-pq-cdc/config"
	"github.com/Trendyol/go-pq-cdc/pq/message/format"
	"github.com/Trendyol/go-pq-cdc/pq/publication"
	"github.com/Trendyol/go-pq-cdc/pq/replication"
	"github.com/Trendyol/go-pq-cdc/pq/slot"
	"log/slog"
	"net/url"
	"os"
	"payloop/internal/application/lib/logger"
)

type CdcStream struct {
	connector *cdc.Connector
	config    config.Config
	logger    logger.Logger
}

func NewCdcStream(databaseURL string, logger logger.Logger) CdcStream {
	host, username, password, database, err := parseDatabaseURL(databaseURL)
	if err != nil {
		slog.Error("parse database URL", "error", err)
		os.Exit(1)
	}
	cfg := config.Config{
		Host:      host,
		Username:  username,
		Password:  password,
		Database:  database,
		DebugMode: false,
		Publication: publication.Config{
			CreateIfNotExists: true,
			Name:              "cdc_pub",
			Operations: publication.Operations{
				publication.OperationInsert,
				publication.OperationDelete,
				publication.OperationTruncate,
				publication.OperationUpdate,
			},
			Tables: publication.Tables{
				publication.Table{
					Name:            "subscriptions",
					ReplicaIdentity: publication.ReplicaIdentityFull,
				}, publication.Table{
					Name:            "payments",
					ReplicaIdentity: publication.ReplicaIdentityFull,
				}, publication.Table{
					Name:            "orders",
					ReplicaIdentity: publication.ReplicaIdentityFull,
				}, publication.Table{
					Name:            "customers",
					ReplicaIdentity: publication.ReplicaIdentityFull,
				}, publication.Table{
					Name:            "products",
					ReplicaIdentity: publication.ReplicaIdentityFull,
				},
			},
		},
		Slot: slot.Config{
			Name:                        "cdc_slot2",
			CreateIfNotExists:           true,
			SlotActivityCheckerInterval: 3000,
		},
		Metric: config.MetricConfig{
			Port: 8082,
		},
		Logger: config.LoggerConfig{
			LogLevel: slog.LevelInfo,
		},
	}

	return CdcStream{
		logger: logger,
		config: cfg,
	}
}

func (c *CdcStream) Start(ctx context.Context, handler func(string, string, interface{}, interface{})) *CdcStream {
	connector, err := cdc.NewConnector(ctx, c.config, func(ctx *replication.ListenerContext) {
		switch msg := ctx.Message.(type) {
		case *format.Insert:
			handler("INSERT", msg.TableName, msg.Decoded, nil)
		case *format.Delete:
			handler("DELETE", msg.TableName, nil, msg.OldDecoded)
		case *format.Update:
			handler("UPDATE", msg.TableName, msg.OldDecoded, msg.NewDecoded)
		}

		if err := ctx.Ack(); err != nil {
			c.logger.Error("ack", "error", err)
		}
	})
	if err != nil {
		slog.Error("new connector", "error", err)
		os.Exit(1)
	}

	go connector.Start(ctx)
	c.connector = &connector
	c.logger.Info("CDC stream started", "host", c.config.Host, "database", c.config.Database)
	return c
}

func parseDatabaseURL(databaseURL string) (host, username, password, database string, err error) {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return "", "", "", "", err
	}

	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		password, _ = parsedURL.User.Password()
	}

	host = parsedURL.Hostname()
	database = parsedURL.Path[1:] // Remove the leading slash

	return host, username, password, database, nil
}
