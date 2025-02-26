package lib

import (
	"log"

	"github.com/spf13/viper"
)

// Env has environment stored
type Env struct {
	ServerPort      string `mapstructure:"SERVER_PORT"`
	TemporalHost    string `mapstructure:"TEMPORAL_HOST"`
	Environment     string `mapstructure:"ENV"`
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

	env := Env{}
	viper.AutomaticEnv()
	viper.AddConfigPath("./")
	viper.AddConfigPath("./../")
	viper.AddConfigPath("./../../")
	viper.AddConfigPath("./../../../")
	viper.AddConfigPath("./../../../../")
	viper.AddConfigPath("./../../../../../")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	err := viper.ReadInConfig()
	if err != nil {
		log.Println("☠️ cannot read configuration file, reading from environment")
		env.CedarPolicyFile = "./policy.cedar"
		env.DBUrl = viper.GetString("DATABASE_URL")
		env.ServerPort = viper.GetString("SERVER_PORT")
		env.Environment = viper.GetString("ENV")
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
