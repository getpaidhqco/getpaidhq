package lib

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log"
)

// Env has environment stored
type Env struct {
	// Server configuration
	ServerPort   string `mapstructure:"GPHQ_SERVER_PORT"`
	ServerHost   string `mapstructure:"GPHQ_SERVER_HOST"`
	McpSsePort   string `mapstructure:"GPHQ_MCP_SSE_PORT"`
	TemporalHost string `mapstructure:"GPHQ_TEMPORAL_HOST"`
	Env          string `mapstructure:"GPHQ_ENV"`

	// Logging configuration
	LogOutput string `mapstructure:"GPHQ_LOG_OUTPUT"`
	LogLevel  string `mapstructure:"GPHQ_PAYLOOP_LOG_LEVEL"`
	LogFormat string `mapstructure:"GPHQ_LOG_FORMAT"`

	// Database configuration
	DBUrl           string `mapstructure:"GPHQ_DATABASE_URL"`
	CedarPolicyFile string `mapstructure:"GPHQ_CEDAR_POLICY"`

	// Auth configuration
	JWTSecret      string `mapstructure:"GPHQ_JWT_SECRET"`
	TokenExpiry    string `mapstructure:"GPHQ_TOKEN_EXPIRY"`
	PaystackSecret string `mapstructure:"GPHQ_PAYSTACK_SECRET"`

	CognitoClientId string `mapstructure:"GPHQ_COGNITO_CLIENT_ID"`
	CognitoPoolId   string `mapstructure:"GPHQ_COGNITO_POOL_ID"`
	CognitoRegion   string `mapstructure:"GPHQ_COGNITO_REGION"`

	PaystackApiKey string `mapstructure:"GPHQ_PAYSTACK_API_KEY"`

	ClerkSecretKey string `mapstructure:"GPHQ_CLERK_SECRET"`
	ClerkDomain    string `mapstructure:"GPHQ_CLERK_DOMAIN"`

	// Email configuration
	EmailProvider    string `mapstructure:"GPHQ_EMAIL_PROVIDER"`
	LoopsApiKey      string `mapstructure:"LOOPS_API_KEY"`
	LoopsApiEndpoint string `mapstructure:"GPHQ_LOOPS_API_ENDPOINT"`
	EmailFromEmail   string `mapstructure:"GPHQ_EMAIL_FROM_EMAIL"`
	EmailFromName    string `mapstructure:"GPHQ_EMAIL_FROM_NAME"`

	// Token Vault configuration
	TokenVaultType      string `mapstructure:"GPHQ_TOKEN_VAULT_TYPE"`       // "aes" or "aws_secrets_manager"
	TokenVaultAESKey    string `mapstructure:"GPHQ_TOKEN_VAULT_AES_KEY"`    // 32-byte AES encryption key
	TokenVaultAWSRegion string `mapstructure:"GPHQ_TOKEN_VAULT_AWS_REGION"` // AWS region for Secrets Manager
	TokenVaultAWSPath   string `mapstructure:"GPHQ_TOKEN_VAULT_AWS_PATH"`   // Base path for AWS Secrets Manager

	// Pubsub configuration
	PubsubProvider string `mapstructure:"GPHQ_PUBSUB_PROVIDER"`

	// S3 Storage configuration
	S3Bucket string `mapstructure:"GPHQ_S3_BUCKET"`
	S3Region string `mapstructure:"GPHQ_S3_REGION"`
}

// NewEnv creates a new environment
func NewEnv() Env {

	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
	viper.AutomaticEnv()

	var env Env

	viper.BindEnv("GPHQ_SERVER_PORT")
	viper.BindEnv("GPHQ_SERVER_PORT")
	viper.BindEnv("GPHQ_SERVER_HOST")
	viper.BindEnv("GPHQ_MCP_SSE_PORT")
	viper.BindEnv("GPHQ_TEMPORAL_HOST")
	viper.BindEnv("GPHQ_ENV")
	viper.BindEnv("GPHQ_LOG_OUTPUT")
	viper.BindEnv("GPHQ_PAYLOOP_LOG_LEVEL")
	viper.BindEnv("GPHQ_LOG_FORMAT")
	viper.BindEnv("GPHQ_DATABASE_URL")
	viper.BindEnv("GPHQ_USAGE_DATABASE_URL")
	viper.BindEnv("GPHQ_REPORTING_DATABASE_URL")
	viper.BindEnv("GPHQ_CEDAR_POLICY")
	viper.BindEnv("GPHQ_JWT_SECRET")
	viper.BindEnv("GPHQ_TOKEN_EXPIRY")
	viper.BindEnv("GPHQ_PAYSTACK_SECRET")
	viper.BindEnv("GPHQ_COGNITO_CLIENT_ID")
	viper.BindEnv("GPHQ_COGNITO_POOL_ID")
	viper.BindEnv("GPHQ_COGNITO_REGION")
	viper.BindEnv("GPHQ_PAYSTACK_API_KEY")
	viper.BindEnv("GPHQ_CLERK_SECRET")
	viper.BindEnv("GPHQ_CLERK_DOMAIN")
	viper.BindEnv("GPHQ_EMAIL_PROVIDER")
	viper.BindEnv("LOOPS_API_KEY")
	viper.BindEnv("GPHQ_LOOPS_API_ENDPOINT")
	viper.BindEnv("GPHQ_EMAIL_FROM_EMAIL")
	viper.BindEnv("GPHQ_EMAIL_FROM_NAME")
	viper.BindEnv("GPHQ_TOKEN_VAULT_TYPE")
	viper.BindEnv("GPHQ_TOKEN_VAULT_AES_KEY")
	viper.BindEnv("GPHQ_TOKEN_VAULT_AWS_REGION")
	viper.BindEnv("GPHQ_TOKEN_VAULT_AWS_PATH")
	viper.BindEnv("GPHQ_PUBSUB_PROVIDER")
	viper.BindEnv("GPHQ_S3_BUCKET")
	viper.BindEnv("GPHQ_S3_REGION")
	err = viper.Unmarshal(&env)
	if err != nil {
		log.Println("☠️ cannot read configuration file, reading from environment")
		env.CedarPolicyFile = "./policy.cedar"

		// Server configuration
		env.DBUrl = viper.GetString("GPHQ_DATABASE_URL")
		env.ServerPort = viper.GetString("GPHQ_SERVER_PORT")
		env.ServerHost = viper.GetString("GPHQ_SERVER_HOST")
		env.Env = viper.GetString("GPHQ_ENV")
		env.TemporalHost = viper.GetString("GPHQ_TEMPORAL_HOST")
		env.McpSsePort = viper.GetString("GPHQ_MCP_SSE_PORT")

		// Logging configuration
		env.LogLevel = viper.GetString("GPHQ_PAYLOOP_LOG_LEVEL")
		env.LogFormat = viper.GetString("GPHQ_LOG_FORMAT")
		env.LogOutput = viper.GetString("GPHQ_LOG_OUTPUT")

		// Auth configuration
		env.JWTSecret = viper.GetString("GPHQ_JWT_SECRET")
		env.TokenExpiry = viper.GetString("GPHQ_TOKEN_EXPIRY")
		env.ClerkSecretKey = viper.GetString("GPHQ_CLERK_SECRET")
		env.ClerkDomain = viper.GetString("GPHQ_CLERK_DOMAIN")

		// Email configuration
		env.EmailProvider = viper.GetString("GPHQ_EMAIL_PROVIDER")
		env.LoopsApiKey = viper.GetString("LOOPS_API_KEY")
		env.LoopsApiEndpoint = viper.GetString("GPHQ_LOOPS_API_ENDPOINT")
		env.EmailFromEmail = viper.GetString("GPHQ_EMAIL_FROM_EMAIL")
		env.EmailFromName = viper.GetString("GPHQ_EMAIL_FROM_NAME")

		// Token Vault configuration
		env.TokenVaultType = viper.GetString("GPHQ_TOKEN_VAULT_TYPE")
		env.TokenVaultAESKey = viper.GetString("GPHQ_TOKEN_VAULT_AES_KEY")
		env.TokenVaultAWSRegion = viper.GetString("GPHQ_TOKEN_VAULT_AWS_REGION")
		env.TokenVaultAWSPath = viper.GetString("GPHQ_TOKEN_VAULT_AWS_PATH")

		// Pubsub configuration
		env.PubsubProvider = viper.GetString("GPHQ_PUBSUB_PROVIDER")

		// S3 Storage configuration
		env.S3Bucket = viper.GetString("GPHQ_S3_BUCKET")
		env.S3Region = viper.GetString("GPHQ_S3_REGION")

		return env
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		log.Fatal("☠️ environment can't be loaded: ", err)
	}
	a := env.Get("GPHQ_DATABASE_URL")
	log.Println(a)
	return env
}

func (e Env) Get(key string) string {
	// Check if the key already has the GPHQ_ prefix
	if len(key) >= 5 && key[:5] == "GPHQ_" {
		return viper.GetString(key)
	}
	return viper.GetString("GPHQ_" + key)
}
