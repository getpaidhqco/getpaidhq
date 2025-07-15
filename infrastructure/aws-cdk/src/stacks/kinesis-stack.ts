import * as cdk from 'aws-cdk-lib';
import * as kinesis from 'aws-cdk-lib/aws-kinesis';
import * as kms from 'aws-cdk-lib/aws-kms';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import { Construct } from 'constructs';

export class KinesisStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // KMS key for encryption
    const kmsKey = new kms.Key(this, 'KinesisEncryptionKey', {
      description: 'KMS key for Kinesis stream encryption',
      enableKeyRotation: true,
      alias: 'alias/payloop-kinesis-key',
    });

    // Kinesis stream for usage events
    const usageStream = new kinesis.Stream(this, 'UsageEventsStream', {
      streamName: 'payloop-usage-events',
      shardCount: 2, // Start with 2 shards as per spec
      retentionPeriod: cdk.Duration.hours(168), // 7 days as per spec
      encryption: kinesis.StreamEncryption.KMS,
      encryptionKey: kmsKey,
      streamMode: kinesis.StreamMode.PROVISIONED,
    });

    // Enable enhanced monitoring with shard-level metrics
    usageStream.enableEnhancedMonitoring([
      kinesis.ShardLevelMetrics.ALL
    ]);

    // CloudWatch alarms for monitoring
    // 1. Shard utilization alarm
    const shardUtilizationAlarm = new cloudwatch.Alarm(this, 'ShardUtilizationAlarm', {
      metric: usageStream.metricIncomingBytes({
        statistic: 'Sum',
        period: cdk.Duration.minutes(5),
      }),
      threshold: 1000000 * 0.8, // 80% of 1MB/s per shard (2 shards = 2MB/s)
      evaluationPeriods: 3,
      datapointsToAlarm: 2,
      alarmDescription: 'Alarm when shard utilization exceeds 80%',
      comparisonOperator: cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
    });

    // 2. PutRecord failures alarm
    const putRecordFailuresAlarm = new cloudwatch.Alarm(this, 'PutRecordFailuresAlarm', {
      metric: usageStream.metricPutRecordSuccess({
        statistic: 'Average',
        period: cdk.Duration.minutes(5),
      }),
      threshold: 0.95, // 95% success rate (5% failure)
      evaluationPeriods: 3,
      datapointsToAlarm: 2,
      alarmDescription: 'Alarm when PutRecord success rate drops below 95%',
      comparisonOperator: cloudwatch.ComparisonOperator.LESS_THAN_THRESHOLD,
    });

    // 3. Iterator age alarm (consumer lag)
    const iteratorAgeAlarm = new cloudwatch.Alarm(this, 'IteratorAgeAlarm', {
      metric: usageStream.metricGetRecordsIteratorAgeMilliseconds({
        statistic: 'Maximum',
        period: cdk.Duration.minutes(5),
      }),
      threshold: 3600000, // 1 hour in milliseconds
      evaluationPeriods: 3,
      datapointsToAlarm: 2,
      alarmDescription: 'Alarm when iterator age exceeds 1 hour',
      comparisonOperator: cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
    });

    // IAM role for ECS tasks to access Kinesis
    const ecsTaskRole = new iam.Role(this, 'EcsKinesisTaskRole', {
      assumedBy: new iam.ServicePrincipal('ecs-tasks.amazonaws.com'),
      description: 'Role for ECS tasks to access Kinesis streams',
    });

    // Grant permissions to the ECS task role
    usageStream.grantRead(ecsTaskRole);
    usageStream.grantWrite(ecsTaskRole);
    kmsKey.grantEncryptDecrypt(ecsTaskRole);

    // Output the stream ARN and name
    new cdk.CfnOutput(this, 'UsageStreamArn', {
      value: usageStream.streamArn,
      description: 'ARN of the Kinesis usage events stream',
      exportName: 'PayloopUsageStreamArn',
    });

    new cdk.CfnOutput(this, 'UsageStreamName', {
      value: usageStream.streamName,
      description: 'Name of the Kinesis usage events stream',
      exportName: 'PayloopUsageStreamName',
    });

    new cdk.CfnOutput(this, 'EcsTaskRoleArn', {
      value: ecsTaskRole.roleArn,
      description: 'ARN of the ECS task role for Kinesis access',
      exportName: 'PayloopEcsKinesisTaskRoleArn',
    });
  }
}