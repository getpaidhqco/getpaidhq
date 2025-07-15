# Kinesis Implementation for Payloop

This document provides an overview of the AWS Kinesis implementation for Payloop's event system, which serves as a drop-in replacement for Kafka specifically for the `gphq.usage.recorded` event stream.

## Implementation Overview

The implementation consists of two main components:

1. **Kinesis Publisher**: A Go implementation of the `DurableEventPublisher` interface using AWS Kinesis Data Streams
2. **AWS CDK Project**: TypeScript code to provision the required AWS infrastructure

## Kinesis Publisher

The Kinesis publisher is implemented in the `internal/infrastructure/events/kinesis` package and follows the same pattern as the existing Kafka implementation. It implements the `DurableEventPublisher` interface, focusing on the `PublishUsageEvent` and `PublishUsageBatch` methods.

### Key Files

- `config.go`: Configuration options for the Kinesis publisher
- `kinesis_publisher.go`: Implementation of the `DurableEventPublisher` interface
- `fx.go`: Integration with the Uber FX dependency injection framework

### Integration with Existing Code

The Kinesis publisher is integrated with the existing event system through the `internal/infrastructure/events/fx.go` file, which now selects the appropriate publisher based on the `EVENT_PUBLISHER` environment variable:

```go
// getEventPublisherModule returns the appropriate event publisher module based on configuration
func getEventPublisherModule() fx.Option {
    // Check if Kinesis is enabled via environment variable
    eventPublisher := os.Getenv("EVENT_PUBLISHER")
    if strings.ToLower(eventPublisher) == "kinesis" {
        return kinesis.Module
    }
    
    // Default to Kafka
    return kafka.Module
}
```

## AWS CDK Project

The AWS CDK project is located in the `infrastructure/aws-cdk` directory and provides infrastructure-as-code for provisioning the required AWS resources.

### Key Files

- `src/app.ts`: Entry point for the CDK application
- `src/stacks/kinesis-stack.ts`: Definition of the Kinesis stream and related resources
- `package.json`: Project dependencies and scripts
- `tsconfig.json`: TypeScript configuration

### Resources Provisioned

- Kinesis stream with 2 shards and 7-day retention
- KMS key for encryption
- CloudWatch alarms for monitoring
- IAM role for ECS tasks

## Usage Instructions

### Deploying the Infrastructure

1. Navigate to the CDK project directory:

```bash
cd infrastructure/aws-cdk
```

2. Install dependencies:

```bash
pnpm install
```

3. Configure environment variables by creating a `.env` file based on `.env.example`.

4. Deploy the infrastructure:

```bash
pnpm run deploy
```

### Configuring Payloop to Use Kinesis

1. Set the `EVENT_PUBLISHER` environment variable to `kinesis`:

```
EVENT_PUBLISHER=kinesis
```

2. Set the required Kinesis configuration environment variables:

```
AWS_REGION=us-east-1
KINESIS_STREAM_PREFIX=payloop-
KINESIS_RETRY_ATTEMPTS=3
KINESIS_PROCESSING_TIMEOUT=30
KINESIS_ENABLE_METRICS=true
```

3. Ensure the ECS task role has the required permissions as defined in the CDK stack.

### Adding the Required Dependencies

The Kinesis implementation requires the AWS SDK for Go v2 Kinesis package. If it's not already in the project, add it:

```bash
go get github.com/aws/aws-sdk-go-v2/service/kinesis
```

## Monitoring and Maintenance

### CloudWatch Metrics

The following CloudWatch metrics are available for monitoring the Kinesis stream:

- Shard utilization
- PutRecord success rate
- Iterator age (consumer lag)

### CloudWatch Alarms

The CDK stack configures the following CloudWatch alarms:

- Shard utilization exceeding 80%
- PutRecord success rate dropping below 95%
- Iterator age exceeding 1 hour

### Scaling

The Kinesis stream is initially provisioned with 2 shards. To scale the stream:

1. Update the `shardCount` parameter in `src/stacks/kinesis-stack.ts`
2. Redeploy the CDK stack

## Conclusion

This implementation provides a fully-managed, scalable alternative to Kafka for the `gphq.usage.recorded` event stream in Payloop. It maintains compatibility with the existing codebase while leveraging AWS-native services for improved operational efficiency.