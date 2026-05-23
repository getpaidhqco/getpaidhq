package lib

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log"
)

// Env has environment stored
type Env struct {
	ServerPort      string `mapstructure:"SERVER_PORT"`
	WorkflowEngine  string `mapstructure:"WORKFLOW_ENGINE"`
	Env             string `mapstructure:"ENV"`
	LogOutput       string `mapstructure:"LOG_OUTPUT"`
	LogLevel        string `mapstructure:"GETPAIDHQ_LOG_LEVEL"`
	DBUrl           string `mapstructure:"DATABASE_URL"`
	CedarPolicyFile string `mapstructure:"CEDAR_POLICY"`

	JWTSecret      string `mapstructure:"JWT_SECRET"`
	PaystackSecret string `mapstructure:"PAYSTACK_SECRET"`

	CognitoClientId string `mapstructure:"COGNITO_CLIENT_ID"`
	CognitoPoolId   string `mapstructure:"COGNITO_POOL_ID"`
	CognitoRegion   string `mapstructure:"COGNITO_REGION"`

	PaystackApiKey string `mapstructure:"PAYSTACK_API_KEY"`

	ClerkSecretKey string `mapstructure:"CLERK_SECRET"`

	HatchetClientToken string `mapstructure:"HATCHET_CLIENT_TOKEN"`
	HatchetHostPort    string `mapstructure:"HATCHET_CLIENT_HOST_PORT"`
	HatchetNamespace   string `mapstructure:"HATCHET_CLIENT_NAMESPACE"`
	HatchetTLSStrategy string `mapstructure:"HATCHET_CLIENT_TLS_STRATEGY"`

	TemporalHost      string `mapstructure:"TEMPORAL_HOST"`
	TemporalNamespace string `mapstructure:"TEMPORAL_NAMESPACE"`
	TemporalTaskQueue string `mapstructure:"TEMPORAL_TASK_QUEUE"`

	// AllowedOrigins is a comma-separated list of CORS origins. When empty,
	// only same-origin requests succeed; "*" enables open CORS (dev only).
	AllowedOrigins string `mapstructure:"ALLOWED_ORIGINS"`
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
	viper.SetDefault("HATCHET_CLIENT_HOST_PORT", "localhost:7077")
	viper.SetDefault("HATCHET_CLIENT_NAMESPACE", "getpaidhq")
	viper.SetDefault("HATCHET_CLIENT_TLS_STRATEGY", "none")
	viper.SetDefault("TEMPORAL_HOST", "localhost:7233")
	viper.SetDefault("TEMPORAL_NAMESPACE", "getpaidhq")
	viper.SetDefault("TEMPORAL_TASK_QUEUE", "getpaidhq-events")

	viper.BindEnv("SERVER_PORT")
	viper.BindEnv("WORKFLOW_ENGINE")
	viper.BindEnv("ENV")
	viper.BindEnv("LOG_OUTPUT")
	viper.BindEnv("GETPAIDHQ_LOG_LEVEL")
	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("CEDAR_POLICY")
	viper.BindEnv("JWT_SECRET")
	viper.BindEnv("PAYSTACK_SECRET")
	viper.BindEnv("COGNITO_CLIENT_ID")
	viper.BindEnv("COGNITO_POOL_ID")
	viper.BindEnv("COGNITO_REGION")
	viper.BindEnv("PAYSTACK_API_KEY")
	viper.BindEnv("CLERK_SECRET")
	viper.BindEnv("HATCHET_CLIENT_TOKEN")
	viper.BindEnv("HATCHET_CLIENT_HOST_PORT")
	viper.BindEnv("HATCHET_CLIENT_NAMESPACE")
	viper.BindEnv("HATCHET_CLIENT_TLS_STRATEGY")
	viper.BindEnv("TEMPORAL_HOST")
	viper.BindEnv("TEMPORAL_NAMESPACE")
	viper.BindEnv("TEMPORAL_TASK_QUEUE")
	viper.BindEnv("ALLOWED_ORIGINS")
	err = viper.Unmarshal(&env)
	if err != nil {
		log.Println("☠️ cannot read configuration file, reading from environment")
		env.CedarPolicyFile = "./policy.cedar"
		env.DBUrl = viper.GetString("DATABASE_URL")
		env.ServerPort = viper.GetString("SERVER_PORT")
		env.Env = viper.GetString("ENV")
		env.LogLevel = viper.GetString("GETPAIDHQ_LOG_LEVEL")
		env.ClerkSecretKey = viper.GetString("CLERK_SECRET")
		env.WorkflowEngine = viper.GetString("WORKFLOW_ENGINE")
		env.HatchetClientToken = viper.GetString("HATCHET_CLIENT_TOKEN")
		env.HatchetHostPort = viper.GetString("HATCHET_CLIENT_HOST_PORT")
		env.HatchetNamespace = viper.GetString("HATCHET_CLIENT_NAMESPACE")
		env.HatchetTLSStrategy = viper.GetString("HATCHET_CLIENT_TLS_STRATEGY")
		env.TemporalHost = viper.GetString("TEMPORAL_HOST")
		env.TemporalNamespace = viper.GetString("TEMPORAL_NAMESPACE")
		env.TemporalTaskQueue = viper.GetString("TEMPORAL_TASK_QUEUE")

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
