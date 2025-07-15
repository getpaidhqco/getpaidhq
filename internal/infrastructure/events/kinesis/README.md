# Kinesis Event Publisher for Payloop

This package implements the `DurableEventPublisher` interface using AWS Kinesis Data Streams as the underlying event storage mechanism. It serves as a drop-in replacement for the Kafka implementation, focusing specifically on the `gphq.usage.recorded` event stream.

## Overview

The Kinesis implementation provides a scalable, fully-managed alternative to Kafka for durable event storage in AWS environments. It maintains the same interface contract as the existing Kafka implementation, ensuring compatibility with the rest of the application.

## Configuration

The Kinesis publisher can be configured using environment variables:

| Environment Variable | Description | Default |
|----------------------|-------------|---------|
| `AWS_REGION` | AWS region for Kinesis streams | `us-east-1` |
| `KINESIS_STREAM_PREFIX` | Prefix for stream names | `payloop-` |
| `KINESIS_STREAM_NAME` | Full stream name (overrides prefix) | `payloop-usage-events` |
| `KINESIS_RETRY_ATTEMPTS` | Number of retry attempts | `3` |
| `KINESIS_PROCESSING_TIMEOUT` | Timeout for operations (seconds) | `30` |
| `KINESIS_ENABLE_METRICS` | Enable CloudWatch metrics | `true` |
| `KINESIS_KMS_KEY_ID` | KMS key ID for encryption | AWS managed CMK |

## Usage

To use the Kinesis publisher instead of Kafka, set the `EVENT_PUBLISHER` environment variable to `kinesis`:

```
EVENT_PUBLISHER=kinesis
```

The application will automatically use the Kinesis implementation for the `DurableEventPublisher` interface.

## Implementation Details

### Publisher

The Kinesis publisher implements the `DurableEventPublisher` interface, focusing on the `PublishUsageEvent` and `PublishUsageBatch` methods for the `gphq.usage.recorded` event stream. Other methods are stubbed out in the initial implementation.

Key features:
- Uses `orgId` as the partition key to maintain tenant isolation
- Implements retry logic with exponential backoff
- Supports KMS encryption for data at rest
- Includes health check functionality

### AWS Infrastructure

The Kinesis stream and related AWS resources are provisioned using AWS CDK. See the `/infrastructure/aws-cdk` directory for details.

## Dependencies

This implementation requires the following AWS SDK dependencies:

```
github.com/aws/aws-sdk-go-v2/service/kinesis
```

If these dependencies are not already in the project, they need to be added:

```bash
go get github.com/aws/aws-sdk-go-v2/service/kinesis
```

## IAM Permissions

The application requires the following IAM permissions to interact with Kinesis:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "kinesis:PutRecord",
        "kinesis:PutRecords",
        "kinesis:DescribeStream",
        "kinesis:ListStreams"
      ],
      "Resource": "arn:aws:kinesis:*:*:stream/payloop-usage-events"
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt",
        "kms:GenerateDataKey"
      ],
      "Resource": "arn:aws:kms:*:*:key/*"
    }
  ]
}
```

These permissions should be attached to the ECS task role used by the application.