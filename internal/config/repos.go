package config

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/adapter/clickhouse"
	"getpaidhq/internal/adapter/db/postgrespgx"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// repoSet is the full set of persistence ports the rest of NewApp consumes. It
// is built by the pgx adapter — every downstream service depends on the port
// interfaces, never a concrete adapter.
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

	// operationalDB is the raw operational pool, retained only for App.DB. close
	// tears down any pools opened by the builder.
	operationalDB *pgxpool.Pool
	close         func()
}

// newRepoSet wires the pgx storage adapter.
func newRepoSet(env lib.Env, logger lib.Logger) (*repoSet, error) {
	pool, err := postgrespgx.NewDatabase(env.Get("DATABASE_URL"), logger)
	if err != nil {
		return nil, err
	}
	eventStore, closeUsage, err := buildEventStore(env, pool, logger)
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

// buildEventStore selects the usage-event backend (postgres | clickhouse). The
// returned closer tears down a separate usage pool when one was opened.
func buildEventStore(env lib.Env, operational *pgxpool.Pool, logger lib.Logger) (port.EventStore, func(), error) {
	switch env.UsageEventStore {
	case "", "postgres":
		if env.UsageDatabaseURL == "" {
			return postgrespgx.NewEventStore(operational), nil, nil
		}
		usage, err := postgrespgx.NewDatabase(env.UsageDatabaseURL, logger)
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
