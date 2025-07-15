package kafka

import (
	"github.com/IBM/sarama"
	"time"
)

// Config holds the configuration for the Kafka publisher
type Config struct {
	// Kafka broker addresses
	Brokers []string `json:"brokers"`
	
	// Client ID for this producer
	ClientID string `json:"client_id"`
	
	// Whether to require acknowledgment from all replicas
	RequireAllAcks bool `json:"require_all_acks"`
	
	// Maximum number of retries for failed messages
	MaxRetries int `json:"max_retries"`
	
	// Timeout for broker connections
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	
	// Timeout for message delivery
	DeliveryTimeout time.Duration `json:"delivery_timeout"`
	
	// Whether to enable TLS
	EnableTLS bool `json:"enable_tls"`
	
	// Whether to enable SASL
	EnableSASL bool `json:"enable_sasl"`
	
	// SASL username
	SASLUsername string `json:"sasl_username"`
	
	// SASL password
	SASLPassword string `json:"sasl_password"`
	
	// SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)
	SASLMechanism string `json:"sasl_mechanism"`
}

// DefaultConfig returns a default configuration for the Kafka publisher
func DefaultConfig() Config {
	return Config{
		Brokers:           []string{"localhost:9092"},
		ClientID:          "payloop-kafka-publisher",
		RequireAllAcks:    true,
		MaxRetries:        3,
		ConnectionTimeout: 10 * time.Second,
		DeliveryTimeout:   30 * time.Second,
		EnableTLS:         false,
		EnableSASL:        false,
		SASLMechanism:     "PLAIN",
	}
}

// NewSaramaConfig creates a Sarama configuration from the Kafka config
func NewSaramaConfig(config Config) *sarama.Config {
	saramaConfig := sarama.NewConfig()
	
	// Set client ID
	saramaConfig.ClientID = config.ClientID
	
	// Configure producer
	if config.RequireAllAcks {
		saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	} else {
		saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	}
	
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.Retry.Max = config.MaxRetries
	saramaConfig.Producer.Partitioner = sarama.NewHashPartitioner
	
	// Configure timeouts
	saramaConfig.Net.DialTimeout = config.ConnectionTimeout
	saramaConfig.Net.ReadTimeout = config.DeliveryTimeout
	saramaConfig.Net.WriteTimeout = config.DeliveryTimeout
	
	// Configure TLS if enabled
	if config.EnableTLS {
		saramaConfig.Net.TLS.Enable = true
	}
	
	// Configure SASL if enabled
	if config.EnableSASL {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = config.SASLUsername
		saramaConfig.Net.SASL.Password = config.SASLPassword
		
		switch config.SASLMechanism {
		case "SCRAM-SHA-256":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		default:
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}
	
	return saramaConfig
}