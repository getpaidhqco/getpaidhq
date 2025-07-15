package kinesis

import (
	"time"
)

// Config holds the configuration for the Kinesis publisher
type Config struct {
	// AWS region for Kinesis streams
	Region string `json:"region"`

	// Stream name prefix for Kinesis streams
	StreamPrefix string `json:"stream_prefix"`

	// Stream name for usage events
	UsageStreamName string `json:"usage_stream_name"`

	// Maximum number of retry attempts for failed operations
	MaxRetryAttempts int `json:"max_retry_attempts"`

	// Processing timeout for Kinesis operations
	ProcessingTimeout time.Duration `json:"processing_timeout"`

	// Whether to enable CloudWatch metrics
	EnableMetrics bool `json:"enable_metrics"`

	// KMS key ID for encryption (if empty, uses AWS managed CMK)
	KmsKeyId string `json:"kms_key_id"`

	// Whether to use explicit AWS credentials (false = use IAM role)
	UseExplicitCredentials bool `json:"use_explicit_credentials"`

	// AWS access key ID (only used if UseExplicitCredentials is true)
	AccessKeyId string `json:"access_key_id"`

	// AWS secret access key (only used if UseExplicitCredentials is true)
	SecretAccessKey string `json:"secret_access_key"`

	// AWS session token (only used if UseExplicitCredentials is true)
	SessionToken string `json:"session_token"`
}

// DefaultConfig returns a default configuration for the Kinesis publisher
func DefaultConfig() Config {
	return Config{
		Region:                 "eu-west-1",
		StreamPrefix:           "payloop-",
		UsageStreamName:        "payloop-usage-events",
		MaxRetryAttempts:       3,
		ProcessingTimeout:      30 * time.Second,
		EnableMetrics:          true,
		UseExplicitCredentials: false,
	}
}

// GetUsageStreamName returns the full stream name for usage events
func (c *Config) GetUsageStreamName() string {
	if c.UsageStreamName != "" {
		return c.UsageStreamName
	}
	return c.StreamPrefix + "usage-events"
}
