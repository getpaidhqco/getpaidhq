package lib

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log"
)

// Env has environment stored
type Env struct {
	// Server configuration
	ServerPort      string `mapstructure:"GETPAIDHQ_SERVER_PORT"`
	ServerHost      string `mapstructure:"GETPAIDHQ_SERVER_HOST"`
	McpSsePort      string `mapstructure:"GETPAIDHQ_MCP_SSE_PORT"`
	TemporalHost    string `mapstructure:"GETPAIDHQ_TEMPORAL_HOST"`
	Env             string `mapstructure:"GETPAIDHQ_ENV"`

	// Logging configuration
	LogOutput       string `mapstructure:"GETPAIDHQ_LOG_OUTPUT"`
	LogLevel        string `mapstructure:"GETPAIDHQ_PAYLOOP_LOG_LEVEL"`
	LogFormat       string `mapstructure:"GETPAIDHQ_LOG_FORMAT"`

	// Database configuration
	DBUrl           string `mapstructure:"GETPAIDHQ_DATABASE_URL"`
	CedarPolicyFile string `mapstructure:"GETPAIDHQ_CEDAR_POLICY"`

	// Auth configuration
	JWTSecret      string `mapstructure:"GETPAIDHQ_JWT_SECRET"`
	TokenExpiry    string `mapstructure:"GETPAIDHQ_TOKEN_EXPIRY"`
	PaystackSecret string `mapstructure:"GETPAIDHQ_PAYSTACK_SECRET"`

	CognitoClientId string `mapstructure:"GETPAIDHQ_COGNITO_CLIENT_ID"`
	CognitoPoolId   string `mapstructure:"GETPAIDHQ_COGNITO_POOL_ID"`
	CognitoRegion   string `mapstructure:"GETPAIDHQ_COGNITO_REGION"`

	PaystackApiKey string `mapstructure:"GETPAIDHQ_PAYSTACK_API_KEY"`

	ClerkSecretKey string `mapstructure:"GETPAIDHQ_CLERK_SECRET"`

	// Email configuration
	EmailProvider   string `mapstructure:"GETPAIDHQ_EMAIL_PROVIDER"`
	LoopsApiKey     string `mapstructure:"GETPAIDHQ_LOOPS_API_KEY"`
	LoopsApiEndpoint string `mapstructure:"GETPAIDHQ_LOOPS_API_ENDPOINT"`
	EmailFromEmail  string `mapstructure:"GETPAIDHQ_EMAIL_FROM_EMAIL"`
	EmailFromName   string `mapstructure:"GETPAIDHQ_EMAIL_FROM_NAME"`

	// Token Vault configuration
	TokenVaultType      string `mapstructure:"GETPAIDHQ_TOKEN_VAULT_TYPE"`       // "aes" or "aws_secrets_manager"
	TokenVaultAESKey    string `mapstructure:"GETPAIDHQ_TOKEN_VAULT_AES_KEY"`    // 32-byte AES encryption key
	TokenVaultAWSRegion string `mapstructure:"GETPAIDHQ_TOKEN_VAULT_AWS_REGION"` // AWS region for Secrets Manager
	TokenVaultAWSPath   string `mapstructure:"GETPAIDHQ_TOKEN_VAULT_AWS_PATH"`   // Base path for AWS Secrets Manager

	// Pubsub configuration
	PubsubProvider string `mapstructure:"GETPAIDHQ_PUBSUB_PROVIDER"`
	PubsubTopic    string `mapstructure:"GETPAIDHQ_PUBSUB_TOPIC"`

	// Subscriptions configuration
	SubscriptionsMaxRetries int `mapstructure:"GETPAIDHQ_SUBSCRIPTIONS_MAX_RETRIES"`

	// S3 Storage configuration
	S3Bucket string `mapstructure:"GETPAIDHQ_S3_BUCKET"`
	S3Region string `mapstructure:"GETPAIDHQ_S3_REGION"`
}

// NewEnv creates a new environment
func NewEnv() Env {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file: %v", err)
	}
	viper.AutomaticEnv()

	var env Env

	viper.BindEnv("GETPAIDHQ_SERVER_PORT")
	viper.BindEnv("GETPAIDHQ_SERVER_HOST")
	viper.BindEnv("GETPAIDHQ_MCP_SSE_PORT")
	viper.BindEnv("GETPAIDHQ_TEMPORAL_HOST")
	viper.BindEnv("GETPAIDHQ_ENV")
	viper.BindEnv("GETPAIDHQ_LOG_OUTPUT")
	viper.BindEnv("GETPAIDHQ_PAYLOOP_LOG_LEVEL")
	viper.BindEnv("GETPAIDHQ_LOG_FORMAT")
	viper.BindEnv("GETPAIDHQ_DATABASE_URL")
	viper.BindEnv("GETPAIDHQ_CEDAR_POLICY")
	viper.BindEnv("GETPAIDHQ_JWT_SECRET")
	viper.BindEnv("GETPAIDHQ_TOKEN_EXPIRY")
	viper.BindEnv("GETPAIDHQ_PAYSTACK_SECRET")
	viper.BindEnv("GETPAIDHQ_COGNITO_CLIENT_ID")
	viper.BindEnv("GETPAIDHQ_COGNITO_POOL_ID")
	viper.BindEnv("GETPAIDHQ_COGNITO_REGION")
	viper.BindEnv("GETPAIDHQ_PAYSTACK_API_KEY")
	viper.BindEnv("GETPAIDHQ_CLERK_SECRET")
	viper.BindEnv("GETPAIDHQ_EMAIL_PROVIDER")
	viper.BindEnv("GETPAIDHQ_LOOPS_API_KEY")
	viper.BindEnv("GETPAIDHQ_LOOPS_API_ENDPOINT")
	viper.BindEnv("GETPAIDHQ_EMAIL_FROM_EMAIL")
	viper.BindEnv("GETPAIDHQ_EMAIL_FROM_NAME")
	viper.BindEnv("GETPAIDHQ_TOKEN_VAULT_TYPE")
	viper.BindEnv("GETPAIDHQ_TOKEN_VAULT_AES_KEY")
	viper.BindEnv("GETPAIDHQ_TOKEN_VAULT_AWS_REGION")
	viper.BindEnv("GETPAIDHQ_TOKEN_VAULT_AWS_PATH")
	viper.BindEnv("GETPAIDHQ_PUBSUB_PROVIDER")
	viper.BindEnv("GETPAIDHQ_PUBSUB_TOPIC")
	viper.BindEnv("GETPAIDHQ_SUBSCRIPTIONS_MAX_RETRIES")
	viper.BindEnv("GETPAIDHQ_S3_BUCKET")
	viper.BindEnv("GETPAIDHQ_S3_REGION")
	err = viper.Unmarshal(&env)
	if err != nil {
		log.Println("☠️ cannot read configuration file, reading from environment")
		env.CedarPolicyFile = "./policy.cedar"

		// Server configuration
		env.DBUrl = viper.GetString("GETPAIDHQ_DATABASE_URL")
		env.ServerPort = viper.GetString("GETPAIDHQ_SERVER_PORT")
		env.ServerHost = viper.GetString("GETPAIDHQ_SERVER_HOST")
		env.Env = viper.GetString("GETPAIDHQ_ENV")
		env.TemporalHost = viper.GetString("GETPAIDHQ_TEMPORAL_HOST")
		env.McpSsePort = viper.GetString("GETPAIDHQ_MCP_SSE_PORT")

		// Logging configuration
		env.LogLevel = viper.GetString("GETPAIDHQ_PAYLOOP_LOG_LEVEL")
		env.LogFormat = viper.GetString("GETPAIDHQ_LOG_FORMAT")
		env.LogOutput = viper.GetString("GETPAIDHQ_LOG_OUTPUT")

		// Auth configuration
		env.JWTSecret = viper.GetString("GETPAIDHQ_JWT_SECRET")
		env.TokenExpiry = viper.GetString("GETPAIDHQ_TOKEN_EXPIRY")
		env.ClerkSecretKey = viper.GetString("GETPAIDHQ_CLERK_SECRET")

		// Email configuration
		env.EmailProvider = viper.GetString("GETPAIDHQ_EMAIL_PROVIDER")
		env.LoopsApiKey = viper.GetString("GETPAIDHQ_LOOPS_API_KEY")
		env.LoopsApiEndpoint = viper.GetString("GETPAIDHQ_LOOPS_API_ENDPOINT")
		env.EmailFromEmail = viper.GetString("GETPAIDHQ_EMAIL_FROM_EMAIL")
		env.EmailFromName = viper.GetString("GETPAIDHQ_EMAIL_FROM_NAME")

		// Token Vault configuration
		env.TokenVaultType = viper.GetString("GETPAIDHQ_TOKEN_VAULT_TYPE")
		env.TokenVaultAESKey = viper.GetString("GETPAIDHQ_TOKEN_VAULT_AES_KEY")
		env.TokenVaultAWSRegion = viper.GetString("GETPAIDHQ_TOKEN_VAULT_AWS_REGION")
		env.TokenVaultAWSPath = viper.GetString("GETPAIDHQ_TOKEN_VAULT_AWS_PATH")

		// Pubsub configuration
		env.PubsubProvider = viper.GetString("GETPAIDHQ_PUBSUB_PROVIDER")
		env.PubsubTopic = viper.GetString("GETPAIDHQ_PUBSUB_TOPIC")

		// Subscriptions configuration
		env.SubscriptionsMaxRetries = viper.GetInt("GETPAIDHQ_SUBSCRIPTIONS_MAX_RETRIES")

		// S3 Storage configuration
		env.S3Bucket = viper.GetString("GETPAIDHQ_S3_BUCKET")
		env.S3Region = viper.GetString("GETPAIDHQ_S3_REGION")

		return env
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		log.Fatal("☠️ environment can't be loaded: ", err)
	}

	return env
}

func (e Env) Get(key string) string {
	return viper.GetString("GETPAIDHQ_" + key)
}
