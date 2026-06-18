package lib

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Env has environment stored
type Env struct {
	ServerPort     string `mapstructure:"SERVER_PORT"`
	WorkflowEngine string `mapstructure:"WORKFLOW_ENGINE"`
	Env            string `mapstructure:"ENV"`
	LogOutput      string `mapstructure:"LOG_OUTPUT"`
	LogLevel       string `mapstructure:"GETPAIDHQ_LOG_LEVEL"`
	LogFormat      string `mapstructure:"LOG_FORMAT"`
	GormLogLevel   string `mapstructure:"GORM_LOG_LEVEL"`
	DBUrl          string `mapstructure:"DATABASE_URL"`
	// DBDriver selects the storage adapter implementation: "gorm" (default) or
	// "pgx". Both open DATABASE_URL/USAGE_DATABASE_URL; only one runs at a time.
	DBDriver        string `mapstructure:"DB_DRIVER"`
	CedarPolicyFile string `mapstructure:"CEDAR_POLICY"`

	// UsageEventStore selects the backend for usage events: "postgres"
	// (default) or "clickhouse". See docs/internal/clickhouse-primer.md.
	UsageEventStore string `mapstructure:"USAGE_EVENT_STORE"`

	// UsageDatabaseURL is the DSN for the Postgres usage-event store. When
	// empty the operational DBUrl is reused (the v1 fallback) so meter_events
	// live in the operational DB until the usage store is split out.
	UsageDatabaseURL string `mapstructure:"USAGE_DATABASE_URL"`

	// ClickhouseDSN is the clickhouse-go connection string for the ClickHouse
	// usage-event store. Only opened when UsageEventStore is "clickhouse".
	// Example: clickhouse://user:pass@localhost:9000/getpaidhq_usage
	ClickhouseDSN string `mapstructure:"CLICKHOUSE_DSN"`

	// UsageIngestMode selects the durable write path for usage events: "sync"
	// (default — write inline on the request) or "jetstream" (publish durably to
	// NATS JetStream; a background consumer drains into the event store). jetstream
	// requires JetStream enabled on the NATS server.
	UsageIngestMode string `mapstructure:"USAGE_INGEST_MODE"`

	// UsageIngestBatchSize is how many events the jetstream consumer writes per
	// IngestBatch. Min 1 (1 = effectively single-row). Ignored in sync mode.
	UsageIngestBatchSize int `mapstructure:"USAGE_INGEST_BATCH_SIZE"`

	JWTSecret      string `mapstructure:"JWT_SECRET"`
	PaystackSecret string `mapstructure:"PAYSTACK_SECRET"`

	// CheckoutWebhookSecret is the HMAC-SHA256 signing key configured
	// in the Checkout.com merchant dashboard for webhook delivery. The
	// Cko-Signature header carries the signature of the raw body
	// signed with this secret.
	CheckoutWebhookSecret string `mapstructure:"CHECKOUT_WEBHOOK_SECRET"`

	// ApiKeyPepper is the server-side secret used to HMAC API keys
	// before storage. Without it a DB compromise would expose every
	// key in plaintext; with it the attacker also needs the pepper
	// (a separate compromise) to use the leaked hashes.
	ApiKeyPepper string `mapstructure:"API_KEY_PEPPER"`

	// SecretsEncryptionKey is a base64-encoded 32-byte key used to
	// AES-256-GCM-encrypt stored PSP credentials. Like the pepper, it
	// keeps a DB compromise from yielding usable gateway secrets: the
	// attacker also needs this key. Generate with `openssl rand -base64 32`.
	SecretsEncryptionKey string `mapstructure:"SECRETS_ENCRYPTION_KEY"`

	// NatsURL is the external NATS server the pubsub adapter connects to.
	NatsURL string `mapstructure:"NATS_URL"`

	CognitoClientId string `mapstructure:"COGNITO_CLIENT_ID"`
	CognitoPoolId   string `mapstructure:"COGNITO_POOL_ID"`
	CognitoRegion   string `mapstructure:"COGNITO_REGION"`

	PaystackApiKey string `mapstructure:"PAYSTACK_API_KEY"`

	ClerkSecretKey string `mapstructure:"CLERK_SECRET"`

	HatchetClientToken string `mapstructure:"HATCHET_CLIENT_TOKEN"`
	HatchetHostPort    string `mapstructure:"HATCHET_CLIENT_HOST_PORT"`
	HatchetNamespace   string `mapstructure:"HATCHET_CLIENT_NAMESPACE"`
	HatchetTLSStrategy string `mapstructure:"HATCHET_CLIENT_TLS_STRATEGY"`

	HatchetBillingSweepInterval time.Duration `mapstructure:"HATCHET_BILLING_SWEEP_INTERVAL"`
	HatchetLogLevel             string        `mapstructure:"HATCHET_LOG_LEVEL"`
	HatchetTracingEnabled       bool          `mapstructure:"HATCHET_TRACING_ENABLED"`

	TemporalHost      string `mapstructure:"TEMPORAL_HOST"`
	TemporalNamespace string `mapstructure:"TEMPORAL_NAMESPACE"`
	TemporalTaskQueue string `mapstructure:"TEMPORAL_TASK_QUEUE"`

	// AllowedOrigins is a comma-separated list of CORS origins. When empty,
	// only same-origin requests succeed; "*" enables open CORS (dev only).
	AllowedOrigins string `mapstructure:"ALLOWED_ORIGINS"`

	// TrustedProxies is a comma-separated list of CIDR blocks whose
	// requests are allowed to set X-Forwarded-For / X-Real-IP. When
	// empty, those headers are IGNORED and RemoteAddr is the source of
	// truth — anything else is a forge attempt. Set to your load
	// balancer / WAF CIDR in prod (e.g. "10.0.0.0/8,127.0.0.1/32").
	TrustedProxies string `mapstructure:"TRUSTED_PROXIES"`

	// RateLimitRPS is the sustained per-client (per-IP) request rate the
	// API allows, in requests per second. A value <= 0 DISABLES rate
	// limiting entirely (the middleware becomes a pass-through), which is
	// the default. Set it in prod to protect the auth path and backends
	// from abuse — e.g. 20.
	RateLimitRPS int `mapstructure:"RATE_LIMIT_RPS"`

	// RateLimitBurst is the maximum burst (token-bucket capacity) a single
	// client may consume before being throttled to RateLimitRPS. When <= 0
	// it defaults to RateLimitRPS. Set it a few× RPS to tolerate normal
	// client bursts — e.g. 40.
	RateLimitBurst int `mapstructure:"RATE_LIMIT_BURST"`
}

// NewEnv creates a new environment
func NewEnv() Env {

	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
	viper.AutomaticEnv()

	var env Env

	viper.SetDefault("WORKFLOW_ENGINE", "hatchet")
	viper.SetDefault("HATCHET_BILLING_SWEEP_INTERVAL", "5m")
	viper.SetDefault("HATCHET_LOG_LEVEL", "warn")
	viper.SetDefault("GORM_LOG_LEVEL", "warn")
	viper.SetDefault("DB_DRIVER", "gorm")
	viper.SetDefault("HATCHET_CLIENT_HOST_PORT", "localhost:7077")
	viper.SetDefault("HATCHET_CLIENT_NAMESPACE", "getpaidhq")
	viper.SetDefault("HATCHET_CLIENT_TLS_STRATEGY", "none")
	viper.SetDefault("TEMPORAL_HOST", "localhost:7233")
	viper.SetDefault("TEMPORAL_NAMESPACE", "getpaidhq")
	viper.SetDefault("TEMPORAL_TASK_QUEUE", "getpaidhq-events")
	viper.SetDefault("NATS_URL", "nats://localhost:4222")
	viper.SetDefault("USAGE_EVENT_STORE", "postgres")
	viper.SetDefault("USAGE_INGEST_MODE", "sync")
	viper.SetDefault("USAGE_INGEST_BATCH_SIZE", 100)
	// Per-IP API rate limiting is ON by default with conservative values.
	// Override per environment; set RATE_LIMIT_RPS=0 to disable entirely.
	viper.SetDefault("RATE_LIMIT_RPS", 20)
	viper.SetDefault("RATE_LIMIT_BURST", 40)

	viper.BindEnv("SERVER_PORT")
	viper.BindEnv("WORKFLOW_ENGINE")
	viper.BindEnv("ENV")
	viper.BindEnv("LOG_OUTPUT")
	viper.BindEnv("LOG_FORMAT")
	viper.BindEnv("GETPAIDHQ_LOG_LEVEL")
	viper.BindEnv("GORM_LOG_LEVEL")
	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("CEDAR_POLICY")
	viper.BindEnv("JWT_SECRET")
	viper.BindEnv("PAYSTACK_SECRET")
	viper.BindEnv("CHECKOUT_WEBHOOK_SECRET")
	viper.BindEnv("API_KEY_PEPPER")
	viper.BindEnv("SECRETS_ENCRYPTION_KEY")
	viper.BindEnv("COGNITO_CLIENT_ID")
	viper.BindEnv("COGNITO_POOL_ID")
	viper.BindEnv("COGNITO_REGION")
	viper.BindEnv("PAYSTACK_API_KEY")
	viper.BindEnv("CLERK_SECRET")
	viper.BindEnv("HATCHET_CLIENT_TOKEN")
	viper.BindEnv("HATCHET_CLIENT_HOST_PORT")
	viper.BindEnv("HATCHET_CLIENT_NAMESPACE")
	viper.BindEnv("HATCHET_CLIENT_TLS_STRATEGY")
	viper.BindEnv("HATCHET_BILLING_SWEEP_INTERVAL")
	viper.BindEnv("HATCHET_LOG_LEVEL")
	viper.BindEnv("HATCHET_TRACING_ENABLED")
	viper.BindEnv("TEMPORAL_HOST")
	viper.BindEnv("TEMPORAL_NAMESPACE")
	viper.BindEnv("TEMPORAL_TASK_QUEUE")
	viper.BindEnv("NATS_URL")
	viper.BindEnv("USAGE_EVENT_STORE")
	viper.BindEnv("USAGE_DATABASE_URL")
	viper.BindEnv("CLICKHOUSE_DSN")
	viper.BindEnv("USAGE_INGEST_MODE")
	viper.BindEnv("USAGE_INGEST_BATCH_SIZE")
	viper.BindEnv("ALLOWED_ORIGINS")
	viper.BindEnv("TRUSTED_PROXIES")
	viper.BindEnv("RATE_LIMIT_RPS")
	viper.BindEnv("RATE_LIMIT_BURST")
	err = viper.Unmarshal(&env)
	if err != nil {
		log.Println("☠️ cannot read configuration file, reading from environment")
		env.CedarPolicyFile = "./policy.cedar"
		env.DBUrl = viper.GetString("DATABASE_URL")
		env.ServerPort = viper.GetString("SERVER_PORT")
		env.Env = viper.GetString("ENV")
		env.LogLevel = viper.GetString("GETPAIDHQ_LOG_LEVEL")
		env.LogFormat = viper.GetString("LOG_FORMAT")
		env.GormLogLevel = viper.GetString("GORM_LOG_LEVEL")
		env.DBDriver = viper.GetString("DB_DRIVER")
		env.ClerkSecretKey = viper.GetString("CLERK_SECRET")
		env.SecretsEncryptionKey = viper.GetString("SECRETS_ENCRYPTION_KEY")
		env.WorkflowEngine = viper.GetString("WORKFLOW_ENGINE")
		env.HatchetClientToken = viper.GetString("HATCHET_CLIENT_TOKEN")
		env.HatchetHostPort = viper.GetString("HATCHET_CLIENT_HOST_PORT")
		env.HatchetNamespace = viper.GetString("HATCHET_CLIENT_NAMESPACE")
		env.HatchetTLSStrategy = viper.GetString("HATCHET_CLIENT_TLS_STRATEGY")
		env.HatchetBillingSweepInterval = viper.GetDuration("HATCHET_BILLING_SWEEP_INTERVAL")
		env.HatchetLogLevel = viper.GetString("HATCHET_LOG_LEVEL")
		env.HatchetTracingEnabled = viper.GetBool("HATCHET_TRACING_ENABLED")
		env.TemporalHost = viper.GetString("TEMPORAL_HOST")
		env.TemporalNamespace = viper.GetString("TEMPORAL_NAMESPACE")
		env.TemporalTaskQueue = viper.GetString("TEMPORAL_TASK_QUEUE")
		env.NatsURL = viper.GetString("NATS_URL")
		env.UsageEventStore = viper.GetString("USAGE_EVENT_STORE")
		env.UsageDatabaseURL = viper.GetString("USAGE_DATABASE_URL")
		env.ClickhouseDSN = viper.GetString("CLICKHOUSE_DSN")
		env.UsageIngestMode = viper.GetString("USAGE_INGEST_MODE")
		env.UsageIngestBatchSize = viper.GetInt("USAGE_INGEST_BATCH_SIZE")

		return env
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		log.Fatal("☠️ environment can't be loaded: ", err)
	}

	return env
}

func (e Env) Get(key string) string {
	return viper.GetString(key)
}
