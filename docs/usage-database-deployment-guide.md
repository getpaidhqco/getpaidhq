# Usage Database Deployment Guide

## AWS RDS Deployment (Recommended for Production)

### Important: pg_cron Limitation
AWS RDS PostgreSQL **does not support** the `pg_cron` extension. This guide provides alternative scheduling solutions.

### 1. Database Setup

```bash
# Step 1: Create RDS PostgreSQL instance
# - Engine: PostgreSQL 15+
# - Instance: db.t3.medium or higher
# - Storage: GP3 with appropriate IOPS for your usage volume
# - Multi-AZ: Recommended for production

# Step 2: Set up usage database schema
export USAGE_DATABASE_URL="postgres://username:password@your-rds-endpoint:5432/payloop_usage"

# Generate Prisma client
pnpm dlx prisma generate --schema=schemas/usage/schema.prisma

# Push schema to database
pnpm dlx prisma db push --schema=schemas/usage/schema.prisma

# Set up partitioning
psql $USAGE_DATABASE_URL -f scripts/setup-usage-partitions.sql

# Create materialized views
psql $USAGE_DATABASE_URL -f scripts/create-materialized-views.sql
```

### 2. Automated Scheduling Options

#### Option A: Application-Level Scheduler (Recommended)

Add the partition scheduler to your main application:

```go
// In your main.go or application bootstrap
func main() {
    // ... existing setup

    // Initialize usage database connection
    usageDB := setupUsageDatabase()
    
    // Start partition scheduler
    partitionScheduler := maintenance.NewPartitionScheduler(usageDB, logger)
    if err := partitionScheduler.Start(); err != nil {
        log.Fatal("Failed to start partition scheduler:", err)
    }
    defer partitionScheduler.Stop()

    // ... rest of application
}
```

**Pros:**
- No additional infrastructure
- Runs with your application
- Easy to monitor and debug

**Cons:**
- Requires application to be running
- Single point of failure

#### Option B: AWS Lambda + EventBridge

1. **Deploy Lambda Function:**
```bash
# Package the Lambda function
zip lambda-partition-maintenance.zip scripts/aws-lambda-partition-maintenance.js

# Deploy using AWS CLI or CDK
aws lambda create-function \
    --function-name usage-partition-maintenance \
    --runtime nodejs18.x \
    --handler index.handler \
    --zip-file fileb://lambda-partition-maintenance.zip \
    --environment Variables='{USAGE_DATABASE_URL=your-connection-string}'
```

2. **Set up EventBridge Rules:**
```bash
# Monthly partition maintenance (1st of month at 2:00 AM UTC)
aws events put-rule \
    --name "usage-partition-maintenance" \
    --schedule-expression "cron(0 2 1 * ? *)"

# Every 5 minutes for materialized view refresh
aws events put-rule \
    --name "usage-materialized-view-refresh" \
    --schedule-expression "cron(*/5 * * * ? *)"
```

**Pros:**
- Serverless - no infrastructure to manage
- Highly reliable
- Cost-effective for low-frequency jobs

**Cons:**
- Additional AWS services
- Cold start latency
- More complex setup

#### Option C: External Cron Server

Set up a dedicated server or container for database maintenance:

```bash
# On your cron server
# Add to crontab: crontab -e

# Monthly partition maintenance (1st of month at 2:00 AM)
0 2 1 * * psql $USAGE_DATABASE_URL -c "SELECT maintain_usage_partitions();"

# Materialized view refresh (every 5 minutes)
*/5 * * * * psql $USAGE_DATABASE_URL -c "SELECT refresh_usage_aggregates();"
```

**Pros:**
- Simple and reliable
- Easy to debug
- Independent of main application

**Cons:**
- Additional server to manage
- Network connectivity requirements

### 3. Monitoring and Alerts

#### Database Monitoring
```sql
-- Check partition health
SELECT * FROM get_partition_info('usage_events');

-- Monitor materialized view freshness
SELECT * FROM get_materialized_view_stats();

-- Check recent maintenance logs
SELECT * FROM usage_event_log 
WHERE event_type IN ('partition_maintenance', 'materialized_views_refreshed') 
ORDER BY timestamp DESC LIMIT 10;
```

#### CloudWatch Metrics (if using Lambda)
- Lambda duration and error rates
- Database connection metrics
- Partition count and size metrics

### 4. Backup and Recovery

#### RDS Automated Backups
- Enable automated backups with 7-day retention
- Consider point-in-time recovery needs
- Test restore procedures regularly

#### Cross-Region Replication
For critical systems, consider setting up read replicas in other regions.

### 5. Performance Tuning

#### Connection Pool Settings
```env
# For application-level scheduler
USAGE_DATABASE_URL="postgres://user:pass@host:5432/db?pool_max_conns=20&pool_min_conns=5"
```

#### RDS Parameter Groups
Recommended parameter adjustments:
```
shared_preload_libraries = 'pg_stat_statements'
log_statement = 'mod'
log_min_duration_statement = 1000
```

### 6. Security Considerations

#### Network Security
- Use VPC with private subnets for RDS
- Security groups with minimal required access
- SSL/TLS encryption in transit

#### Access Control
```sql
-- Create dedicated maintenance user with limited privileges
CREATE USER partition_maintenance WITH PASSWORD 'secure_password';
GRANT USAGE ON SCHEMA public TO partition_maintenance;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO partition_maintenance;
```

### 7. Troubleshooting

#### Common Issues

**Issue: Partition creation fails**
```sql
-- Check if base table exists and is partitioned
SELECT 
    schemaname, tablename, 
    pg_get_expr(c.relpartbound, c.oid) as partition_bounds
FROM pg_tables t
JOIN pg_class c ON c.relname = t.tablename
WHERE tablename LIKE 'usage_events%';
```

**Issue: Materialized view refresh takes too long**
```sql
-- Check view size and refresh duration
SELECT 
    view_name, 
    row_count, 
    size_pretty,
    last_refresh
FROM get_materialized_view_stats();
```

**Issue: Application-level scheduler not running**
```go
// Add health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
    status := partitionScheduler.GetStatus()
    json.NewEncoder(w).Encode(status)
}
```

### 8. Migration from Development to Production

1. **Export partition structure:**
```bash
pg_dump --schema-only $DEV_USAGE_DATABASE_URL > usage_schema.sql
```

2. **Import to production:**
```bash
psql $PROD_USAGE_DATABASE_URL < usage_schema.sql
```

3. **Verify setup:**
```sql
SELECT * FROM get_partition_info('usage_events');
SELECT setup_partition_maintenance_job(); -- Will show scheduling options
```

## Conclusion

While AWS RDS doesn't support `pg_cron`, the alternatives provide reliable automation for partition management. The **application-level scheduler** is recommended for most use cases due to its simplicity and integration with your existing infrastructure.

For high-availability requirements, consider using **AWS Lambda + EventBridge** for critical maintenance tasks, as it provides better fault tolerance and doesn't depend on your application being running.