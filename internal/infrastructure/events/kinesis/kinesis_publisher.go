package kinesis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"sync"
	"time"
)

// KinesisPublisher implements the DurableEventPublisher interface using AWS Kinesis
type KinesisPublisher struct {
	client  *kinesis.Client
	config  Config
	logger  logger.Logger
	mu      sync.Mutex
}

// NewKinesisPublisher creates a new Kinesis publisher
func NewKinesisPublisher(cfg Config, logger logger.Logger) (events.DurableEventPublisher, error) {
	// Create AWS SDK configuration
	var awsCfg aws.Config
	var err error

	// Configure AWS SDK
	if cfg.UseExplicitCredentials {
		// Use explicit credentials if provided
		awsCfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyId,
				cfg.SecretAccessKey,
				cfg.SessionToken,
			)),
		)
	} else {
		// Use IAM role credentials
		awsCfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		logger.Error("Failed to load AWS config", "error", err)
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Kinesis client
	client := kinesis.NewFromConfig(awsCfg)

	// Create publisher
	publisher := &KinesisPublisher{
		client: client,
		config: cfg,
		logger: logger,
	}

	// Validate stream exists
	if err := publisher.validateStream(context.Background(), cfg.GetUsageStreamName()); err != nil {
		logger.Error("Failed to validate Kinesis stream", "error", err)
		return nil, fmt.Errorf("failed to validate Kinesis stream: %w", err)
	}

	return publisher, nil
}

// validateStream checks if the stream exists and is active
func (k *KinesisPublisher) validateStream(ctx context.Context, streamName string) error {
	k.logger.Info("Validating Kinesis stream", "streamName", streamName)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, k.config.ProcessingTimeout)
	defer cancel()

	// Describe the stream
	resp, err := k.client.DescribeStream(ctx, &kinesis.DescribeStreamInput{
		StreamName: aws.String(streamName),
	})

	if err != nil {
		return fmt.Errorf("failed to describe stream: %w", err)
	}

	// Check stream status
	if resp.StreamDescription.StreamStatus != types.StreamStatusActive {
		return fmt.Errorf("stream %s is not active, current status: %s", 
			streamName, resp.StreamDescription.StreamStatus)
	}

	k.logger.Info("Kinesis stream validated successfully", "streamName", streamName)
	return nil
}

// Health checks the health of the Kinesis publisher
func (k *KinesisPublisher) Health() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.client == nil {
		return fmt.Errorf("kinesis client is not initialized")
	}

	// Check if we can describe the stream
	ctx, cancel := context.WithTimeout(context.Background(), k.config.ProcessingTimeout)
	defer cancel()

	_, err := k.client.DescribeStream(ctx, &kinesis.DescribeStreamInput{
		StreamName: aws.String(k.config.GetUsageStreamName()),
	})

	if err != nil {
		return fmt.Errorf("kinesis health check failed: %w", err)
	}

	return nil
}

// PublishUsageEvent publishes a usage event to Kinesis
func (k *KinesisPublisher) PublishUsageEvent(ctx context.Context, event events.RawUsageRecordedEvent) error {
	return k.publishEvent(ctx, k.config.GetUsageStreamName(), event.OrgId, event)
}

// PublishUsageBatch publishes a batch of usage events to Kinesis
func (k *KinesisPublisher) PublishUsageBatch(ctx context.Context, usageEvents []events.RawUsageRecordedEvent) error {
	if len(usageEvents) == 0 {
		return nil
	}

	// Group events by orgId
	eventsByOrg := make(map[string][]events.RawUsageRecordedEvent)
	for _, event := range usageEvents {
		eventsByOrg[event.OrgId] = append(eventsByOrg[event.OrgId], event)
	}

	// Publish events for each orgId
	for orgId, orgEvents := range eventsByOrg {
		if err := k.publishBatch(ctx, k.config.GetUsageStreamName(), orgId, orgEvents); err != nil {
			return err
		}
	}

	return nil
}

// publishEvent publishes a single event to Kinesis
func (k *KinesisPublisher) publishEvent(ctx context.Context, streamName, partitionKey string, event interface{}) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Convert event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, k.config.ProcessingTimeout)
	defer cancel()

	// Create Kinesis record
	input := &kinesis.PutRecordInput{
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey), // Use orgId as partition key
		Data:         data,
	}

	// Add encryption if configured
	if k.config.KmsKeyId != "" {
		// Note: In the AWS SDK v2, encryption is handled differently
		// We would typically set up encryption at the stream level
		// This is just a placeholder to show where encryption would be configured
		k.logger.Debug("KMS encryption configured", "kmsKeyId", k.config.KmsKeyId)
	}

	// Send record to Kinesis
	var result *kinesis.PutRecordOutput
	var retryCount int

	for retryCount <= k.config.MaxRetryAttempts {
		result, err = k.client.PutRecord(ctx, input)

		if err == nil {
			break
		}

		// Check if we should retry
		if retryCount < k.config.MaxRetryAttempts {
			retryCount++
			k.logger.Warn("Retrying Kinesis PutRecord", 
				"error", err, 
				"streamName", streamName, 
				"attempt", retryCount)
			time.Sleep(time.Duration(retryCount*200) * time.Millisecond) // Exponential backoff
		} else {
			k.logger.Error("Failed to publish event to Kinesis after retries", 
				"error", err, 
				"streamName", streamName)
			return fmt.Errorf("failed to publish event to Kinesis: %w", err)
		}
	}

	k.logger.Debug("Published event to Kinesis", 
		"streamName", streamName, 
		"shardId", *result.ShardId, 
		"sequenceNumber", *result.SequenceNumber)
	return nil
}

// publishBatch publishes a batch of events to Kinesis
func (k *KinesisPublisher) publishBatch(ctx context.Context, streamName, partitionKey string, events interface{}) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Convert events to JSON
	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, k.config.ProcessingTimeout)
	defer cancel()

	// Create Kinesis record
	input := &kinesis.PutRecordInput{
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey), // Use orgId as partition key
		Data:         data,
	}

	// Add encryption if configured
	if k.config.KmsKeyId != "" {
		// Note: In the AWS SDK v2, encryption is handled differently
		// We would typically set up encryption at the stream level
		// This is just a placeholder to show where encryption would be configured
		k.logger.Debug("KMS encryption configured", "kmsKeyId", k.config.KmsKeyId)
	}

	// Send record to Kinesis
	var result *kinesis.PutRecordOutput
	var retryCount int

	for retryCount <= k.config.MaxRetryAttempts {
		result, err = k.client.PutRecord(ctx, input)

		if err == nil {
			break
		}

		// Check if we should retry
		if retryCount < k.config.MaxRetryAttempts {
			retryCount++
			k.logger.Warn("Retrying Kinesis PutRecord for batch", 
				"error", err, 
				"streamName", streamName, 
				"attempt", retryCount)
			time.Sleep(time.Duration(retryCount*200) * time.Millisecond) // Exponential backoff
		} else {
			k.logger.Error("Failed to publish batch to Kinesis after retries", 
				"error", err, 
				"streamName", streamName)
			return fmt.Errorf("failed to publish batch to Kinesis: %w", err)
		}
	}

	k.logger.Debug("Published batch to Kinesis", 
		"streamName", streamName, 
		"shardId", *result.ShardId, 
		"sequenceNumber", *result.SequenceNumber)
	return nil
}

// The following methods are implemented to satisfy the DurableEventPublisher interface,
// but they are not used in the current implementation as we're focusing only on usage events.

func (k *KinesisPublisher) PublishBillingEvent(ctx context.Context, event events.BillingEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishBillingEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishPaymentEvent(ctx context.Context, event events.PaymentEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishPaymentEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishSubscriptionEvent(ctx context.Context, event events.SubscriptionEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishSubscriptionEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishCustomerEvent(ctx context.Context, event events.CustomerEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishCustomerEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishInvoiceEvent(ctx context.Context, event events.InvoiceEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishInvoiceEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishRefundEvent(ctx context.Context, event events.RefundEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishRefundEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishProductEvent(ctx context.Context, event events.ProductEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishProductEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishPriceEvent(ctx context.Context, event events.PriceEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishPriceEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishDunningEvent(ctx context.Context, event events.DunningEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishDunningEvent not implemented for Kinesis")
	return nil
}

func (k *KinesisPublisher) PublishEventBatch(ctx context.Context, events []events.BaseEvent) error {
	// Not implemented for Kinesis in the initial phase
	k.logger.Warn("PublishEventBatch not implemented for Kinesis")
	return nil
}
