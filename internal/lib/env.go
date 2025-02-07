package lib

import (
	"log"

	"github.com/spf13/viper"
)

// Env has environment stored
type Env struct {
	ServerPort  string `mapstructure:"SERVER_PORT"`
	Environment string `mapstructure:"ENV"`
	LogOutput   string `mapstructure:"LOG_OUTPUT"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	DBUrl       string `mapstructure:"DATABASE_URL"`

	JWTSecret      string `mapstructure:"JWT_SECRET"`
	PaystackSecret string `mapstructure:"PAYSTACK_SECRET"`

	CognitoClientId string `mapstructure:"COGNITO_CLIENT_ID"`
	CognitoPoolId   string `mapstructure:"COGNITO_POOL_ID"`
	CognitoRegion   string `mapstructure:"COGNITO_REGION"`

	PaystackApiKey string `mapstructure:"PAYSTACK_API_KEY"`
}

// NewEnv creates a new environment
func NewEnv() Env {

	env := Env{}
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
		log.Fatal("☠️ cannot read configuration")
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		log.Fatal("☠️ environment can't be loaded: ", err)
	}

	return env
}
