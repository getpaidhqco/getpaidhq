package config

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"

	"getpaidhq/internal/adapter/clickhouse"
	"getpaidhq/internal/adapter/db/postgresgorm"
	"getpaidhq/internal/adapter/db/postgrespgx"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// repoSet is the full set of persistence ports the rest of NewApp consumes. It
// is built by exactly one driver-specific builder (gorm or pgx) selected by
// DB_DRIVER, so swapping drivers touches only this file — every downstream
// service depends on the port interfaces, never a concrete adapter.
type repoSet struct {
	subscription        port.SubscriptionRepository
	order               port.OrderRepository
	customer            port.CustomerRepository
	payment             port.PaymentRepository
	paymentMethod       port.PaymentMethodRepository
	product             port.ProductRepository
	variant             port.VariantRepository
	price               port.PriceRepository
	session             port.SessionRepository
	cart                port.CartRepository
	org                 port.OrgRepository
	user                port.UserRepository
	setting             port.SettingRepository
	webhookSub          port.WebhookSubscriptionRepository
	apiKey              port.ApiKeyRepository
	idempotency         port.IdempotencyKeyRepository
	idempotencyStore    port.IdempotencyStore
	psp                 port.PspRepository
	metadata            port.MetadataStoreRepository
	dunning             port.DunningRepository
	invoice             port.InvoiceRepository
	meter               port.MeterRepository
	coupon              port.CouponRepository
	couponCode          port.CouponCodeRepository
	couponReservation   port.CouponReservationRepository
	discount            port.DiscountRepository
	priorPaymentChecker port.PriorPaymentChecker
	tx                  port.TxManager
	eventStore          port.EventStore
	outbox              port.OutboxRepository

	// operationalDB is the raw operational handle (*gorm.DB or *pgxpool.Pool),
	// retained only for App.DB. close tears down any pools opened by the builder.
	operationalDB any
	close         func()
}

// newRepoSet selects the db adapter implementation from DB_DRIVER.
func newRepoSet(env lib.Env, logger lib.Logger) (*repoSet, error) {
	switch env.DBDriver {
	case "", "gorm":
		return newGormRepoSet(env, logger)
	case "pgx":
		return newPgxRepoSet(env, logger)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q (want 'gorm' or 'pgx')", env.DBDriver)
	}
}

// newGormRepoSet wires the GORM adapter (the default).
func newGormRepoSet(env lib.Env, logger lib.Logger) (*repoSet, error) {
	db, err := postgresgorm.NewDatabase(env.Get("DATABASE_URL"), logger, env.GormLogLevel)
	if err != nil {
		return nil, err
	}
	eventStore, err := buildGormEventStore(env, db, logger)
	if err != nil {
		return nil, err
	}
	return &repoSet{
		subscription:        postgresgorm.NewSubscriptionRepo(db),
		order:               postgresgorm.NewOrderRepo(db),
		customer:            postgresgorm.NewCustomerRepo(db),
		payment:             postgresgorm.NewPaymentRepo(db),
		paymentMethod:       postgresgorm.NewPaymentMethodRepo(db),
		product:             postgresgorm.NewProductRepo(db),
		variant:             postgresgorm.NewVariantRepo(db),
		price:               postgresgorm.NewPriceRepo(db),
		session:             postgresgorm.NewSessionRepo(db),
		cart:                postgresgorm.NewCartRepo(db),
		org:                 postgresgorm.NewOrgRepo(db),
		user:                postgresgorm.NewUserRepo(db),
		setting:             postgresgorm.NewSettingRepo(db),
		webhookSub:          postgresgorm.NewWebhookSubscriptionRepo(db),
		apiKey:              postgresgorm.NewApiKeyRepo(db),
		idempotency:         postgresgorm.NewIdempotencyKeyRepo(db),
		idempotencyStore:    postgresgorm.NewIdempotencyStore(db, env.IdempotencyLockTTL, env.IdempotencyRetentionTTL),
		psp:                 postgresgorm.NewPspRepo(db),
		metadata:            postgresgorm.NewMetadataStoreRepo(db),
		dunning:             postgresgorm.NewDunningRepo(db),
		invoice:             postgresgorm.NewInvoiceRepo(db),
		meter:               postgresgorm.NewMeterRepo(db),
		coupon:              postgresgorm.NewCouponRepo(db),
		couponCode:          postgresgorm.NewCouponCodeRepo(db),
		couponReservation:   postgresgorm.NewCouponReservationRepo(db),
		discount:            postgresgorm.NewDiscountRepo(db),
		priorPaymentChecker: postgresgorm.NewPriorPaymentChecker(db),
		tx:                  postgresgorm.NewTxManager(db),
		eventStore:          eventStore,
		outbox:              postgresgorm.NewOutboxRepo(db),
		operationalDB:       db,
		close: func() {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		},
	}, nil
}

// newPgxRepoSet wires the hand-written pgx adapter.
func newPgxRepoSet(env lib.Env, logger lib.Logger) (*repoSet, error) {
	pool, err := postgrespgx.NewDatabase(env.Get("DATABASE_URL"), logger, env.GormLogLevel)
	if err != nil {
		return nil, err
	}
	eventStore, closeUsage, err := buildPgxEventStore(env, pool, logger)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &repoSet{
		subscription:        postgrespgx.NewSubscriptionRepo(pool),
		order:               postgrespgx.NewOrderRepo(pool),
		customer:            postgrespgx.NewCustomerRepo(pool),
		payment:             postgrespgx.NewPaymentRepo(pool),
		paymentMethod:       postgrespgx.NewPaymentMethodRepo(pool),
		product:             postgrespgx.NewProductRepo(pool),
		variant:             postgrespgx.NewVariantRepo(pool),
		price:               postgrespgx.NewPriceRepo(pool),
		session:             postgrespgx.NewSessionRepo(pool),
		cart:                postgrespgx.NewCartRepo(pool),
		org:                 postgrespgx.NewOrgRepo(pool),
		user:                postgrespgx.NewUserRepo(pool),
		setting:             postgrespgx.NewSettingRepo(pool),
		webhookSub:          postgrespgx.NewWebhookSubscriptionRepo(pool),
		apiKey:              postgrespgx.NewApiKeyRepo(pool),
		idempotency:         postgrespgx.NewIdempotencyKeyRepo(pool),
		idempotencyStore:    postgrespgx.NewIdempotencyStore(pool, env.IdempotencyLockTTL, env.IdempotencyRetentionTTL),
		psp:                 postgrespgx.NewPspRepo(pool),
		metadata:            postgrespgx.NewMetadataStoreRepo(pool),
		dunning:             postgrespgx.NewDunningRepo(pool),
		invoice:             postgrespgx.NewInvoiceRepo(pool),
		meter:               postgrespgx.NewMeterRepo(pool),
		coupon:              postgrespgx.NewCouponRepo(pool),
		couponCode:          postgrespgx.NewCouponCodeRepo(pool),
		couponReservation:   postgrespgx.NewCouponReservationRepo(pool),
		discount:            postgrespgx.NewDiscountRepo(pool),
		priorPaymentChecker: postgrespgx.NewPriorPaymentChecker(pool),
		tx:                  postgrespgx.NewTxManager(pool),
		eventStore:          eventStore,
		outbox:              postgrespgx.NewOutboxRepo(pool),
		operationalDB:       pool,
		close: func() {
			pool.Close()
			if closeUsage != nil {
				closeUsage()
			}
		},
	}, nil
}

// usageGormDB returns the GORM handle backing the Postgres usage-event store —
// a SEPARATE pool when USAGE_DATABASE_URL is set, else the operational handle.
func usageGormDB(env lib.Env, operational *gorm.DB, logger lib.Logger) (*gorm.DB, error) {
	if env.UsageDatabaseURL == "" {
		return operational, nil
	}
	separate, err := postgresgorm.NewDatabase(env.UsageDatabaseURL, logger, env.GormLogLevel)
	if err != nil {
		return nil, fmt.Errorf("open usage database: %w", err)
	}
	return separate, nil
}

// buildGormEventStore selects the usage-event backend (postgres-gorm | clickhouse).
func buildGormEventStore(env lib.Env, db *gorm.DB, logger lib.Logger) (port.EventStore, error) {
	switch env.UsageEventStore {
	case "", "postgres":
		udb, err := usageGormDB(env, db, logger)
		if err != nil {
			return nil, err
		}
		return postgresgorm.NewEventStore(udb), nil
	case "clickhouse":
		return clickhouse.NewEventStore(env.ClickhouseDSN)
	default:
		return nil, fmt.Errorf("unsupported USAGE_EVENT_STORE %q (want 'postgres' or 'clickhouse')", env.UsageEventStore)
	}
}

// buildPgxEventStore selects the usage-event backend for the pgx driver. The
// returned closer tears down a separate usage pool when one was opened.
func buildPgxEventStore(env lib.Env, operational *pgxpool.Pool, logger lib.Logger) (port.EventStore, func(), error) {
	switch env.UsageEventStore {
	case "", "postgres":
		if env.UsageDatabaseURL == "" {
			return postgrespgx.NewEventStore(operational), nil, nil
		}
		usage, err := postgrespgx.NewDatabase(env.UsageDatabaseURL, logger, env.GormLogLevel)
		if err != nil {
			return nil, nil, fmt.Errorf("open usage database: %w", err)
		}
		return postgrespgx.NewEventStore(usage), usage.Close, nil
	case "clickhouse":
		ch, err := clickhouse.NewEventStore(env.ClickhouseDSN)
		if err != nil {
			return nil, nil, err
		}
		return ch, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported USAGE_EVENT_STORE %q (want 'postgres' or 'clickhouse')", env.UsageEventStore)
	}
}

// closerFunc adapts a plain teardown func to io.Closer for the App.closers list.
type closerFunc func() error

func (f closerFunc) Close() error { return f() }
