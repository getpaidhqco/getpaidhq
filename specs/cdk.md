# GPHQ CDK Infrastructure Specification

## Project Overview
Create AWS CDK infrastructure for GPHQ backend deployment in TEST environment using existing resources where possible.

## Account Details
- **AWS Account**: 329237115630
- **Region**: eu-west-1
- **Environment**: test

## Resource Strategy

### Existing Resources (Import/Reuse)
- **VPC**: vpc-132c3475 (172.31.0.0/16)
- **Subnets**: Use existing default VPC subnets
- **RDS**: database-1 (PostgreSQL)
- **Redis**: checkoutjoy-0001-001 (Valkey)
- **ECS Cluster**: cj-frontend (existing)

### New Resources (Create)
- **ECS Service**: GPHQ backend service
- **Task Definition**: Container configuration
- **Target Group**: Load balancer integration
- **Security Groups**: Application-specific rules
- **ECR Repository**: Docker image storage
- **CloudWatch Logs**: Application logging

## Infrastructure Components

### 1. Network Configuration
- Import existing VPC: vpc-132c3475
- Use existing subnets:
  - Public: subnet-9a60f7c0, subnet-f9d7f2b1
  - Private: subnet-830a5fe5, subnet-0e29041f41bd7e43f
- Create security group for GPHQ application

### 2. Database Configuration
- Import existing RDS instance: database-1
- Import existing Redis cluster: checkoutjoy-0001-001
- No new database resources required

### 3. Container Platform
- Import existing ECS cluster: cj-frontend
- Create new Fargate service for GPHQ
- Create task definition with minimal resources

### 4. Application Configuration
- **Container Image**: To be built and pushed to ECR
- **Port**: 8081 (GPHQ Go application)
- **Resource Allocation**: Smallest available (256 CPU, 512 MB memory)
- **Instance Count**: 1 task (minimal for test)

### 5. Load Balancing
- Create Application Load Balancer target group
- Health check on /health endpoint
- HTTP protocol on port 8081

### 6. Logging & Monitoring
- CloudWatch log group: /ecs/gphq
- Log retention: 7 days (minimal for test)

## Resource Sizing (Minimal)
- **ECS Task**: 256 CPU units, 512 MB memory
- **Desired Count**: 1
- **Auto Scaling**: Disabled (test environment)
- **Load Balancer**: Application Load Balancer (shared if possible)

## Environment Variables & Secrets

### Parameter Store Structure
```
/gphq/test/
├── environment              (String)
├── database/
│   ├── host                 (String)
│   ├── name                 (String)
│   ├── username             (SecureString)
│   └── password             (SecureString)
├── redis/
│   ├── host                 (String)
│   └── port                 (String)
└── secrets/
    ├── clerk-secret-key     (SecureString)
    └── api-signing-key      (SecureString)
```

### CDK Implementation
- Reference existing parameters via `ssm.StringParameter.valueFromLookup()`
- Reference secrets via `ecs.Secret.fromSsmParameter()`
- Never create parameters with actual values in CDK code
- Pre-deployment: Create parameters separately via AWS CLI or scripts

### Environment Variables (Non-Sensitive)
- ENV=test
- PORT=8081
- DATABASE_HOST=[from Parameter Store]
- REDIS_HOST=[from Parameter Store]
- AWS_REGION=eu-west-1

### Secrets (Sensitive)
- DATABASE_USERNAME=[from Parameter Store SecureString]
- DATABASE_PASSWORD=[from Parameter Store SecureString]
- CLERK_SECRET_KEY=[from Parameter Store SecureString]

## Security Configuration
- Security group allowing inbound 8081 from ALB
- Security group allowing outbound to RDS (5432)
- Security group allowing outbound to Redis (6379)
- Task execution role for ECR and CloudWatch
- Task role for application runtime permissions

## CDK Stack Structure
```
gphq-infrastructure/
├── lib/
│   ├── gphq-stack.ts
├── bin/
│   └── gphq.ts
├── config/
│   └── test-environment.ts
├── package.json
├── cdk.json
└── tsconfig.json
```

## Configuration Parameters
```typescript
test: {
  account: "329237115630",
          region: "eu-west-1",
          existingResources: {
    vpcId: "vpc-132c3475",
            databaseIdentifier: "database-1",
            redisClusterId: "checkoutjoy-0001-001",
            ecsClusterName: "cj-frontend"
  },
  newResources: {
    ecsService: true,
            targetGroup: true,
            securityGroups: true,
            ecrRepository: true,
            cloudWatchLogs: true
  },
  sizing: {
    cpu: 256,
            memory: 512,
            desiredCount: 1
  }
}
```

## Deployment Commands
```bash
# Initialize CDK project
cdk init app --language typescript

# Install dependencies
npm install

# Deploy to test environment
cdk deploy --context env=test

# Verify deployment
aws ecs describe-services --cluster cj-frontend --services gphq
```

## ECR Setup
- Repository name: gphq
- Image scanning: enabled
- Lifecycle policy: keep last 1 images

## Outputs Required
- ECR repository URI
- ECS service ARN
- Target group ARN
- Application URL

## Cost Optimization
- Use smallest instance types
- Single task instance
- 7-day log retention
- No NAT gateway dependencies (use existing)
- No redundancy (test environment)