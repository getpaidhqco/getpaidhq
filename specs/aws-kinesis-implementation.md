---
title: AWS Kinesis Implementation Specification
description: Specification for implementing AWS Kinesis as an alternative to Kafka for durable event storage in Payloop
---

# AWS Kinesis Implementation Specification

## Overview

This specification outlines the implementation of AWS Kinesis Data Streams as an alternative to Apache Kafka for durable event storage in the Payloop subscription billing platform. The implementation will maintain the existing dual-publisher architecture while providing AWS-native event streaming capabilities.

## Current Architecture Analysis

### Existing Event System
- **Dual Publishing**: NATS (real-time notifications) + Kafka (durable events)
- **Event Types**: 100+ predefined topics across subscription, payment, billing, dunning domains
- **Multi-tenancy**: All events include `orgId` for tenant isolation
- **Event Sourcing**: Complete audit trail with structured domain events

### Kafka Usage Patterns
- **Primary Focus**: `gphq.usage.recorded` topic for usage event processing
- **Partitioning**: By `orgId` for tenant isolation
- **Retention**: 7-day retention for usage events
- **Consumers**: Usage event processing for billing calculations

## Implementation Goals

### Primary Objectives
1. **Drop-in Replacement**: Replace Kafka with Kinesis without changing business logic
2. **Maintain Interface**: Preserve existing `DurableEventPublisher` interface
3. **AWS Integration**: Leverage AWS ecosystem for monitoring, scaling, and operations
4. **Cost Optimization**: Reduce operational overhead of self-managed Kafka
5. **Scalability**: Auto-scaling capabilities for variable workloads

### Non-Goals
- Replace NATS (real-time notifications remain unchanged)
- Modify existing event schemas or domain logic
- Change consumer patterns significantly

## Technical Specification

### 1. Stream Architecture

#### Stream Mapping
The current implementation uses only the `gphq.usage.recorded` topic for usage event processing. This will be mapped to a single Kinesis stream:

- `gphq.usage.recorded` → `payloop-usage-events`

#### Stream Configuration Requirements
The `payloop-usage-events` stream must be configured with:
- **Retention Period**: 7 days (168 hours) minimum
- **Shard Count**: Auto-scaling based on throughput requirements
- **Encryption**: KMS encryption enabled
- **Metrics**: Shard-level metrics for monitoring
- **Partitioning**: By `orgId` to maintain tenant isolation

### 2. Interface Implementation

#### DurableEventPublisher Interface Requirement
**CRITICAL**: The implementation must maintain 100% compatibility with the existing `DurableEventPublisher` interface without any modifications. All method signatures and behaviors must remain identical to ensure zero impact on current implementations.

**Key Requirements:**
- All existing event publishing methods must be implemented exactly as defined
- Event serialization and partitioning must maintain current behavior
- For the current scope, only `PublishUsageEvent` method needs functional implementation
- All other methods can have stub implementations during initial phase
- Error handling patterns must match existing Kafka implementation
- Retry logic must be compatible with current timeout expectations

### 3. Consumer Implementation

#### Consumer Service Requirements
**Scope**: Initial implementation focuses solely on consuming `gphq.usage.recorded` events from the `payloop-usage-events` stream.

**Key Requirements:**
- Must integrate with existing event handler patterns
- Support checkpointing for reliable event processing
- Handle shard rebalancing automatically
- Maintain compatibility with current event processing timeouts
- Support graceful shutdown and restart
- Error handling must match existing Kafka consumer behavior
- Must run efficiently in ECS container environment

### 4. Configuration Management

#### Configuration Requirements
**ECS Environment Variables**: All configuration must be externalized through environment variables suitable for ECS task definitions.

**Required Configuration:**
- AWS region and credentials (via IAM roles preferred)
- Stream naming configuration
- Retry and timeout settings
- Monitoring and metrics configuration
- ECS-specific settings (memory limits, connection pooling)

**Environment Variable Structure:**
- `AWS_REGION`: AWS region for Kinesis streams
- `KINESIS_STREAM_PREFIX`: Prefix for stream names
- `KINESIS_RETRY_ATTEMPTS`: Number of retry attempts
- `KINESIS_PROCESSING_TIMEOUT`: Event processing timeout
- `KINESIS_ENABLE_METRICS`: Enable CloudWatch metrics

### 5. Module Integration

#### FX Module Integration Requirements
**Dependency Injection**: Must integrate seamlessly with existing Uber FX dependency injection framework.

**Key Requirements:**
- Module must provide implementation of `DurableEventPublisher` interface
- Must register consumer service with existing event handling system
- Configuration must be injectable and testable
- Should support graceful shutdown through FX lifecycle hooks
- Must be conditionally enabled/disabled based on configuration

**Bootstrap Integration**: The module must be integrated into `internal/application/bootstrap/modules.go` with configuration-based activation to allow switching between Kafka and Kinesis implementations without code changes.

### 6. Monitoring and Observability

#### CloudWatch Integration
**Metrics Requirements**: Must integrate with existing CloudWatch monitoring infrastructure for ECS deployments.

**Key Metrics to Track:**
- Event publishing success/failure rates
- Event processing latency
- Consumer lag and throughput
- Kinesis API throttling events
- Shard utilization metrics

**Health Check Integration**: Must implement health checks compatible with ECS health check mechanisms and existing application health endpoints.

**Alerting**: Must integrate with existing alerting infrastructure for operational monitoring.

### 7. Testing Strategy

#### Testing Requirements
**Unit Testing**: Must provide comprehensive unit tests with mocked AWS SDK clients to ensure reliability without AWS dependencies.

**Integration Testing**: Must include integration tests that can run against actual AWS Kinesis streams in test environments.




#### Stream Lifecycle Management
**Operational Requirements**: Must provide stream lifecycle management capabilities including creation, configuration updates, and monitoring.

**Key Features:**
- Automatic stream creation if not exists
- Configuration updates without downtime
- Proper cleanup and resource management

### 10. Security Considerations

#### IAM Permissions
**ECS Task Role Requirements**: The ECS task role must have the following IAM permissions for Kinesis operations:

**Publisher Permissions:**
- `kinesis:PutRecord` - For publishing individual events
- `kinesis:PutRecords` - For batch publishing
- `kinesis:DescribeStream` - For stream validation
- `kinesis:ListStreams` - For stream discovery

**Consumer Permissions:**
- `kinesis:GetRecords` - For consuming events
- `kinesis:GetShardIterator` - For shard iteration
- `kinesis:ListShards` - For shard discovery

**KMS Permissions (for encryption):**
- `kms:Decrypt` - For decrypting stream data
- `kms:GenerateDataKey` - For encrypting stream data

**Resource Constraints**: All permissions should be scoped to `payloop-usage-events` stream only.

#### Encryption Configuration
**KMS Encryption**: Must use KMS encryption for data at rest with customer-managed keys where possible. Encryption configuration must be externalized through environment variables.


## AWS Kinesis Provisioning

### Stream Creation Requirements
**Stream Name**: `payloop-usage-events`

**Stream Configuration**:
- **Mode**: Provisioned (for predictable performance)
- **Shard Count**: Start with 2 shards, scale based on throughput
- **Retention Period**: 168 hours (7 days)
- **Encryption**: KMS encryption with customer-managed key
- **Shard-Level Metrics**: Enabled for monitoring

### Provisioning Methods
**Option 1: AWS CLI**
```bash
aws kinesis create-stream \
    --stream-name payloop-usage-events \
    --shard-count 2 \
    --region us-east-1

aws kinesis put-retention-period \
    --stream-name payloop-usage-events \
    --retention-period-hours 168
```

**Option 2: AWS CDK/CloudFormation**
- Use infrastructure-as-code for reproducible deployments
- Include stream configuration in existing infrastructure stack
- Enable automatic shard scaling through Application Auto Scaling


### Monitoring Setup
**CloudWatch Metrics**: Enable shard-level metrics for detailed monitoring
**CloudWatch Alarms**: Set up alarms for:
- Shard utilization > 80%
- PutRecord failures
- Consumer lag
- Iterator age




### Environment Variables
**Required Environment Variables for ECS:**
- `AWS_REGION`: AWS region for Kinesis streams
- `KINESIS_STREAM_PREFIX`: Stream prefix for naming
- `KINESIS_RETRY_ATTEMPTS`: Retry configuration
- `KINESIS_PROCESSING_TIMEOUT`: Processing timeout
- `KINESIS_ENABLE_METRICS`: Enable CloudWatch metrics

### IAM Task Role
**Task Role ARN**: Must be assigned to ECS task for Kinesis access
**Service Role**: ECS service role must allow task role assumption
**Cross-Account Access**: Configure if Kinesis streams are in different AWS account



### Resource Limits
**Memory**: Set appropriate memory limits for Go application + AWS SDK
**CPU**: Configure CPU limits based on expected event processing load
**Network**: Ensure adequate network bandwidth for Kinesis API calls

## Conclusion

This specification provides a focused roadmap for implementing AWS Kinesis as a drop-in replacement for Kafka specifically for the `gphq.usage.recorded` event stream in the Payloop event system. The implementation maintains the existing `DurableEventPublisher` interface without modifications while providing AWS-native streaming capabilities optimized for ECS deployment.

**Critical Success Factors:**
- Strict adherence to existing interface contracts
- Focus on `gphq.usage.recorded` stream only
- Proper IAM role configuration for ECS deployment
