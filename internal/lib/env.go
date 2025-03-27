package lib

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log"
)

// Env has environment stored
type Env struct {
	ServerPort      string `mapstructure:"SERVER_PORT"`
	TemporalHost    string `mapstructure:"TEMPORAL_HOST"`
	Env             string `mapstructure:"ENV"`
	LogOutput       string `mapstructure:"LOG_OUTPUT"`
	LogLevel        string `mapstructure:"LOG_LEVEL"`
	DBUrl           string `mapstructure:"DATABASE_URL"`
	CedarPolicyFile string `mapstructure:"CEDAR_POLICY"`

	JWTSecret      string `mapstructure:"JWT_SECRET"`
	PaystackSecret string `mapstructure:"PAYSTACK_SECRET"`

	CognitoClientId string `mapstructure:"COGNITO_CLIENT_ID"`
	CognitoPoolId   string `mapstructure:"COGNITO_POOL_ID"`
	CognitoRegion   string `mapstructure:"COGNITO_REGION"`

	PaystackApiKey string `mapstructure:"PAYSTACK_API_KEY"`

	ClerkSecretKey string `mapstructure:"CLERK_SECRET"`
}

// NewEnv creates a new environment
func NewEnv() Env {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading ..env file: %v", err)
	}
	viper.AutomaticEnv()

	var env Env

	viper.BindEnv("SERVER_PORT")
	viper.BindEnv("TEMPORAL_HOST")
	viper.BindEnv("ENV")
	viper.BindEnv("LOG_OUTPUT")
	viper.BindEnv("LOG_LEVEL")
	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("CEDAR_POLICY")
	viper.BindEnv("JWT_SECRET")
	viper.BindEnv("PAYSTACK_SECRET")
	viper.BindEnv("COGNITO_CLIENT_ID")
	viper.BindEnv("COGNITO_POOL_ID")
	viper.BindEnv("COGNITO_REGION")
	viper.BindEnv("PAYSTACK_API_KEY")
	viper.BindEnv("CLERK_SECRET")
	err = viper.Unmarshal(&env)
	if err != nil {
		log.Println("☠️ cannot read configuration file, reading from environment")
		env.CedarPolicyFile = "./policy.cedar"
		env.DBUrl = viper.GetString("DATABASE_URL")
		env.ServerPort = viper.GetString("SERVER_PORT")
		env.Env = viper.GetString("ENV")
		env.LogLevel = viper.GetString("LOG_LEVEL")
		env.ClerkSecretKey = viper.GetString("CLERK_SECRET")
		env.TemporalHost = viper.GetString("TEMPORAL_HOST")

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
