# Payloop Dunning Feature - Comprehensive Implementation Specification

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Design Decisions](#design-decisions)
3. [Database Schema](#database-schema)
4. [Workflow Implementation](#workflow-implementation)
5. [Configuration System](#configuration-system)
6. [Token Security System](#token-security-system)
7. [API Endpoints](#api-endpoints)
8. [Event System](#event-system)
9. [Use Cases & Examples](#use-cases--examples)
10. [Integration Patterns](#integration-patterns)

## Architecture Overview

### Core Concept
**Dunning = Complete Payment Failure Recovery**, including:
- Immediate technical retries (1 min, 5 min, 15 min)
- Progressive customer communication retries (Day 3, 7, 14, 30)
- Account suspension logic
- Payment method update flows
- Customer service escalation
- Final cancellation or recovery

### Workflow Responsibility Split

#### SubscriptionWorkflow
- ✅ Calculate billing dates
- ✅ Attempt single payment charge
- ✅ Handle successful payments
- ✅ Spawn DunningWorkflow on ANY failure
- ❌ NO retry logic

#### DunningWorkflow
- ✅ ALL payment retries (immediate + progressive)
- ✅ Customer communication strategy
- ✅ Account state management during recovery
- ✅ Recovery success/failure determination
- ✅ Subscription reactivation or cancellation

### System Integration Flow
```
Payment Fails → SubscriptionWorkflow → DunningWorkflow
                     ↓                       ↓
               Updates subscription       Publishes events
               to "past_due"             to NATS
                     ↓                       ↓
               Skips future billing    NotificationService
               until status changes    sends communications
```

## Design Decisions

### 1. One DunningWorkflow = One Campaign
Every payment failure spawns a new DunningWorkflow instance and creates a corresponding campaign record. This provides:
- **Clear audit trail**: Each failure event has distinct recovery effort
- **Configuration snapshots**: Historical accuracy of strategies used
- **Analytics tracking**: Campaign-level insights and success rates
- **Workflow recovery**: Campaign serves as checkpoint for restarts

### 2. Independent Workflow Execution
DunningWorkflow runs completely independently from SubscriptionWorkflow using `PARENT_CLOSE_POLICY_ABANDON`. Benefits:
- **Fault tolerance**: Dunning continues even if subscription workflow fails
- **Long-running capability**: Can run for months without blocking other operations
- **Independent monitoring**: Each workflow can be monitored separately
- **Flexible timing**: Different retry schedules don't affect billing cycles

### 3. State-Based Coordination
Communication between workflows happens through:
- **Database state**: Subscription status determines billing behavior
- **External signals**: DunningWorkflow signals back to SubscriptionWorkflow
- **Event publication**: Events trigger external systems (notifications, analytics)

### 4. Event-Driven Messaging
Messaging is handled by separate NotificationService consuming NATS events rather than built into workflows:
- **Separation of concerns**: Business logic separate from communication
- **Independent scaling**: Message processing scales independently
- **Multiple consumers**: Analytics, audit, webhooks can all consume same events
- **Provider flexibility**: Easy to swap email/SMS providers

### 5. Secure Token-Based Payment Updates
Payment update links use cryptographically signed tokens with:
- **No login required**: Minimal friction during stressful payment failure
- **Configurable limits**: Usage count and time-based expiration
- **Action scoping**: Tokens only allow specific permitted actions
- **Complete audit trail**: All token usage tracked for security

## Database Schema

### Prisma Schema Extensions

```prisma
// Add to existing schema.prisma

// Dunning campaign tracking
model DunningCampaign {
  orgId             String   @map("org_id")
  id                String   @default(cuid())
  
  // Relationships
  subscriptionId    String   @map("subscription_id")
  customerId        String   @map("customer_id")
  
  // Workflow metadata
  temporalWorkflowId String  @map("temporal_workflow_id")
  temporalRunId     String   @map("temporal_run_id")
  parentWorkflowId  String?  @map("parent_workflow_id")
  
  // Campaign details
  status            DunningStatus @default(active)
  failedAmount      Int      @map("failed_amount")
  currency          String
  initialFailureReason String? @map("initial_failure_reason")
  
  // Attempt tracking
  totalAttempts     Int      @default(0) @map("total_attempts")
  immediateAttempts Int      @default(0) @map("immediate_attempts")
  progressiveAttempts Int    @default(0) @map("progressive_attempts")
  
  // Timeline
  startedAt         DateTime @default(now()) @map("started_at")
  lastAttemptAt     DateTime? @map("last_attempt_at")
  nextAttemptAt     DateTime? @map("next_attempt_at")
  completedAt       DateTime? @map("completed_at")
  
  // Outcomes
  recoveryMethod    String?  @map("recovery_method")
  recoveredAmount   Int?     @map("recovered_amount")
  recoveredAt       DateTime? @map("recovered_at")
  finalFailureReason String? @map("final_failure_reason")
  
  // Configuration snapshot
  configSnapshot    Json?    @map("config_snapshot")
  
  // Metadata
  metadata          Json?
  
  createdAt         DateTime @default(now()) @map("created_at")
  updatedAt         DateTime @updatedAt @map("updated_at")
  
  // Relationships
  subscription      Subscription @relation(fields: [orgId, subscriptionId], references: [orgId, id])
  customer          Customer @relation(fields: [orgId, customerId], references: [orgId, id])
  attempts          DunningAttempt[]
  communications    DunningCommunication[]
  tokens            PaymentUpdateToken[]
  
  @@id([orgId, id])
  @@map("dunning_campaigns")
}

enum DunningStatus {
  active
  paused
  recovered
  failed
  cancelled
  expired
}

// Individual payment retry attempts
model DunningAttempt {
  orgId             String   @map("org_id")
  id                String   @default(cuid())
  
  // Relationships
  dunningCampaignId String   @map("dunning_campaign_id")
  subscriptionId    String   @map("subscription_id")
  
  // Attempt details
  attemptNumber     Int      @map("attempt_number")
  attemptType       DunningAttemptType @map("attempt_type")
  
  // Payment details
  amount            Int
  currency          String
  paymentMethodId   String?  @map("payment_method_id")
  
  // Results
  status            PaymentStatus
  failureReason     String?  @map("failure_reason")
  failureCode       String?  @map("failure_code")
  processorResponse Json?    @map("processor_response")
  
  // Performance metrics
  processingTimeMs  Int?     @map("processing_time_ms")
  attemptedAt       DateTime @default(now()) @map("attempted_at")
  completedAt       DateTime? @map("completed_at")
  
  // Context
  triggeredBy       String?  @map("triggered_by")
  metadata          Json?
  
  createdAt         DateTime @default(now()) @map("created_at")
  
  // Relationships
  campaign          DunningCampaign @relation(fields: [orgId, dunningCampaignId], references: [orgId, id])
  
  @@id([orgId, id])
  @@map("dunning_attempts")
}

enum DunningAttemptType {
  immediate
  progressive
  manual
  triggered
}

// Customer communications during dunning
model DunningCommunication {
  orgId             String   @map("org_id")
  id                String   @default(cuid())
  
  // Relationships
  dunningCampaignId String   @map("dunning_campaign_id")
  customerId        String   @map("customer_id")
  
  // Communication details
  channel           CommunicationChannel
  templateId        String   @map("template_id")
  attemptNumber     Int      @map("attempt_number")
  
  // Content
  subject           String?
  contentPreview    String?  @map("content_preview")
  personalizationData Json?  @map("personalization_data")
  
  // Delivery tracking
  sentAt            DateTime? @map("sent_at")
  deliveredAt       DateTime? @map("delivered_at")
  openedAt          DateTime? @map("opened_at")
  clickedAt         DateTime? @map("clicked_at")
  bouncedAt         DateTime? @map("bounced_at")
  
  // Provider details
  provider          String
  providerMessageId String?  @map("provider_message_id")
  providerResponse  Json?    @map("provider_response")
  
  // Status
  status            CommunicationStatus @default(pending)
  failureReason     String?  @map("failure_reason")
  
  createdAt         DateTime @default(now()) @map("created_at")
  updatedAt         DateTime @updatedAt @map("updated_at")
  
  // Relationships
  campaign          DunningCampaign @relation(fields: [orgId, dunningCampaignId], references: [orgId, id])
  customer          Customer @relation(fields: [orgId, customerId], references: [orgId, id])
  
  @@id([orgId, id])
  @@map("dunning_communications")
}

enum CommunicationChannel {
  email
  sms
  push
  webhook
  in_app
}

enum CommunicationStatus {
  pending
  sent
  delivered
  failed
  bounced
}

// Secure payment update tokens
model PaymentUpdateToken {
  orgId             String   @map("org_id")
  tokenId           String   @map("token_id")
  
  // Relationships
  subscriptionId    String   @map("subscription_id")
  customerId        String   @map("customer_id")
  dunningCampaignId String?  @map("dunning_campaign_id")
  
  // Token data
  tokenData         Json     @map("token_data")
  signature         String
  
  // Security & usage
  expiresAt         DateTime @map("expires_at")
  maxUses           Int      @default(5) @map("max_uses")
  usedCount         Int      @default(0) @map("used_count")
  status            TokenStatus @default(active)
  
  // Allowed actions
  allowedActions    Json     @map("allowed_actions")
  
  // Admin generation tracking
  adminGenerated    Boolean  @default(false) @map("admin_generated")
  adminUserId       String?  @map("admin_user_id")
  adminReason       String?  @map("admin_reason")
  adminNotes        String?  @map("admin_notes")
  
  // Audit trail
  createdBy         String   @map("created_by")
  createdAt         DateTime @default(now()) @map("created_at")
  lastUsedAt        DateTime? @map("last_used_at")
  lastUsedIp        String?  @map("last_used_ip")
  
  // Relationships
  subscription      Subscription @relation(fields: [orgId, subscriptionId], references: [orgId, id])
  customer          Customer @relation(fields: [orgId, customerId], references: [orgId, id])
  campaign          DunningCampaign? @relation(fields: [orgId, dunningCampaignId], references: [orgId, id])
  usageHistory      PaymentTokenUsage[]
  
  @@id([orgId, tokenId])
  @@map("payment_update_tokens")
}

enum TokenStatus {
  active
  expired
  revoked
  max_uses_reached
}

// Token usage tracking
model PaymentTokenUsage {
  orgId       String   @map("org_id")
  tokenId     String   @map("token_id")
  usedAt      DateTime @default(now()) @map("used_at")
  ipAddress   String?  @map("ip_address")
  userAgent   String?  @map("user_agent")
  actionTaken String?  @map("action_taken")
  success     Boolean?
  
  // Relationships
  token       PaymentUpdateToken @relation(fields: [orgId, tokenId], references: [orgId, tokenId])
  
  @@id([orgId, tokenId, usedAt])
  @@map("payment_token_usage")
}

// Dunning configuration
model DunningConfiguration {
  orgId       String   @map("org_id")
  id          String   @default(cuid())
  
  // Configuration hierarchy
  name        String
  description String?
  priority    Int      @default(0)
  
  // Targeting rules
  appliesTo   DunningConfigScope @default(organization) @map("applies_to")
  targetRules Json?    @map("target_rules")
  
  // The actual configuration
  config      Json
  
  // Status and testing
  status      ConfigStatus @default(active)
  isAbTest    Boolean  @default(false) @map("is_ab_test")
  abTestPercentage Decimal? @map("ab_test_percentage")
  
  // Metadata
  createdBy   String?  @map("created_by")
  createdAt   DateTime @default(now()) @map("created_at")
  updatedAt   DateTime @updatedAt @map("updated_at")
  
  @@id([orgId, id])
  @@map("dunning_configurations")
}

enum DunningConfigScope {
  organization
  customer_segment
  subscription_tier
  customer
  ab_test
}

enum ConfigStatus {
  active
  inactive
  archived
}

// Analytics tables
model DunningAnalyticsDaily {
  orgId                  String   @map("org_id")
  date                   DateTime @db.Date
  
  // Volume metrics
  campaignsStarted       Int      @default(0) @map("campaigns_started")
  campaignsCompleted     Int      @default(0) @map("campaigns_completed")
  totalAttempts          Int      @default(0) @map("total_attempts")
  
  // Recovery metrics
  immediateRecoveries    Int      @default(0) @map("immediate_recoveries")
  progressiveRecoveries  Int      @default(0) @map("progressive_recoveries")
  manualRecoveries       Int      @default(0) @map("manual_recoveries")
  totalRecoveries        Int      @default(0) @map("total_recoveries")
  
  // Financial metrics
  amountAtRisk           Int      @default(0) @map("amount_at_risk")
  amountRecovered        Int      @default(0) @map("amount_recovered")
  amountLost             Int      @default(0) @map("amount_lost")
  
  // Communication metrics
  emailsSent             Int      @default(0) @map("emails_sent")
  smsSent                Int      @default(0) @map("sms_sent")
  totalCommunications    Int      @default(0) @map("total_communications")
  
  // Performance metrics
  avgRecoveryTimeHours   Decimal? @map("avg_recovery_time_hours")
  avgAttemptsToRecovery  Decimal? @map("avg_attempts_to_recovery")
  
  // Segmentation
  customerSegment        String?  @map("customer_segment")
  subscriptionTier       String?  @map("subscription_tier")
  failureReasonCategory  String?  @map("failure_reason_category")
  
  createdAt              DateTime @default(now()) @map("created_at")
  updatedAt              DateTime @updatedAt @map("updated_at")
  
  @@id([orgId, date, customerSegment, subscriptionTier, failureReasonCategory])
  @@map("dunning_analytics_daily")
}

// Customer dunning history summary
model CustomerDunningHistory {
  orgId                    String   @map("org_id")
  customerId               String   @map("customer_id")
  
  // Lifetime stats
  totalDunningCampaigns    Int      @default(0) @map("total_dunning_campaigns")
  successfulRecoveries     Int      @default(0) @map("successful_recoveries")
  failedCampaigns          Int      @default(0) @map("failed_campaigns")
  
  // Financial impact
  totalAmountAtRisk        Int      @default(0) @map("total_amount_at_risk")
  totalAmountRecovered     Int      @default(0) @map("total_amount_recovered")
  totalAmountLost          Int      @default(0) @map("total_amount_lost")
  
  // Behavior patterns
  avgRecoveryTimeHours     Decimal? @map("avg_recovery_time_hours")
  preferredRecoveryMethod  String?  @map("preferred_recovery_method")
  mostResponsiveChannel    CommunicationChannel? @map("most_responsive_channel")
  
  // Risk scoring
  paymentReliabilityScore  Decimal? @map("payment_reliability_score")
  dunningRiskTier          String?  @map("dunning_risk_tier")
  
  // Dates
  firstDunningAt           DateTime? @map("first_dunning_at")
  lastDunningAt            DateTime? @map("last_dunning_at")
  lastRecoveryAt           DateTime? @map("last_recovery_at")
  
  updatedAt                DateTime @updatedAt @map("updated_at")
  
  // Relationships
  customer                 Customer @relation(fields: [orgId, customerId], references: [orgId, id])
  
  @@id([orgId, customerId])
  @@map("customer_dunning_history")
}

// Extend existing models
model Subscription {
  // ... existing fields ...
  
  // Dunning-related additions
  dunningStatus         String?    @map("dunning_status")
  dunningStartedAt      DateTime?  @map("dunning_started_at")
  dunningCompletedAt    DateTime?  @map("dunning_completed_at")
  dunningCampaignCount  Int        @default(0) @map("dunning_campaign_count")
  lastDunningRecoveryAt DateTime?  @map("last_dunning_recovery_at")
  
  // Relationships
  dunningCampaigns      DunningCampaign[]
  paymentTokens         PaymentUpdateToken[]
  
  // ... rest of existing fields ...
}

model Customer {
  // ... existing fields ...
  
  // Dunning preferences
  dunningPreferences    Json?      @map("dunning_preferences")
  
  // Relationships
  dunningCampaigns      DunningCampaign[]
  dunningCommunications DunningCommunication[]
  paymentTokens         PaymentUpdateToken[]
  dunningHistory        CustomerDunningHistory?
  
  // ... rest of existing fields ...
}

model Payment {
  // ... existing fields ...
  
  // Dunning context
  dunningCampaignId     String?    @map("dunning_campaign_id")
  dunningAttemptNumber  Int?       @map("dunning_attempt_number")
  isDunningRecovery     Boolean    @default(false) @map("is_dunning_recovery")
  
  // ... rest of existing fields ...
}
```

## Workflow Implementation

### DunningWorkflow Core Structure

```go
type DunningWorkflowInput struct {
    SubscriptionID       string
    CustomerID          string
    OrgID              string
    FailedAmount        int
    Currency           string
    InitialFailure     ChargeResult
    ParentWorkflowID   string
    DunningCampaignID  string
}

type DunningWorkflowState struct {
    CampaignID      string
    SubscriptionID  string
    CustomerID      string
    OrgID          string
    Status         DunningStatus
    AttemptCount   int
    FailedAmount   int
    Currency       string
    Config         DunningConfig
    StartedAt      time.Time
    LastAttemptAt  time.Time
    NextAttemptAt  time.Time
}

func DunningWorkflow(ctx workflow.Context, input DunningWorkflowInput) error {
    logger := workflow.GetLogger(ctx)
    
    // Initialize campaign state
    campaign := &DunningWorkflowState{
        CampaignID:     input.DunningCampaignID,
        SubscriptionID: input.SubscriptionID,
        CustomerID:     input.CustomerID,
        OrgID:         input.OrgID,
        Status:        DunningStatusActive,
        AttemptCount:  0,
        FailedAmount:  input.FailedAmount,
        Currency:      input.Currency,
        StartedAt:     workflow.Now(ctx),
    }
    
    // Load dunning configuration
    var config DunningConfig
    err := workflow.ExecuteActivity(ctx, ResolveDunningConfigActivity,
        ConfigResolutionInput{
            OrgID:            input.OrgID,
            CustomerID:       input.CustomerID,
            SubscriptionID:   input.SubscriptionID,
        }).Get(ctx, &config)
    if err != nil {
        return err
    }
    campaign.Config = config
    
    // Create campaign record
    err = workflow.ExecuteActivity(ctx, CreateDunningCampaignActivity, campaign).Get(ctx, nil)
    if err != nil {
        return err
    }
    
    // Setup signal handlers
    setupSignalHandlers(ctx, campaign)
    
    // Phase 1: Immediate retries for transient failures
    if config.ImmediateRetries.Enabled && shouldAttemptImmediateRetries(input.InitialFailure) {
        recovered, err := executeImmediateRetries(ctx, campaign)
        if err != nil {
            logger.Error("Immediate retries failed", "error", err)
        }
        if recovered {
            return finalizeCampaign(ctx, campaign, DunningStatusRecovered)
        }
    }
    
    // Phase 2: Progressive retries with customer communication
    err = executeProgressiveRetries(ctx, campaign)
    if err != nil {
        logger.Error("Progressive retries failed", "error", err)
        return finalizeCampaign(ctx, campaign, DunningStatusFailed)
    }
    
    return nil
}

func executeImmediateRetries(ctx workflow.Context, campaign *DunningWorkflowState) (bool, error) {
    logger := workflow.GetLogger(ctx)
    
    for _, interval := range campaign.Config.ImmediateRetries.Intervals {
        if campaign.AttemptCount >= campaign.Config.ImmediateRetries.MaxAttempts {
            break
        }
        
        // Wait for retry interval
        if campaign.AttemptCount > 0 {
            workflow.Sleep(ctx, interval)
        }
        
        campaign.AttemptCount++
        campaign.LastAttemptAt = workflow.Now(ctx)
        
        // Execute payment retry
        var result ChargeResult
        err := workflow.ExecuteActivity(ctx, ChargeCustomerActivity,
            ChargeInput{
                SubscriptionID: campaign.SubscriptionID,
                Amount:        campaign.FailedAmount,
                Currency:      campaign.Currency,
                AttemptNumber: campaign.AttemptCount,
                AttemptType:   "immediate",
                CampaignID:    campaign.CampaignID,
            }).Get(ctx, &result)
        
        // Record attempt
        attempt := DunningAttempt{
            CampaignID:    campaign.CampaignID,
            AttemptNumber: campaign.AttemptCount,
            AttemptType:   "immediate",
            Status:        result.Status,
            FailureReason: result.FailureReason,
        }
        workflow.ExecuteActivity(ctx, RecordDunningAttemptActivity, attempt)
        
        if err != nil {
            logger.Error("Immediate retry attempt failed", "attempt", campaign.AttemptCount, "error", err)
            continue
        }
        
        if result.Status == "succeeded" {
            logger.Info("Immediate retry successful", "attempt", campaign.AttemptCount)
            
            // Publish success event
            workflow.ExecuteActivity(ctx, PublishEventActivity, DunningEvent{
                EventType:      "dunning.immediate_recovery",
                CampaignID:     campaign.CampaignID,
                SubscriptionID: campaign.SubscriptionID,
                CustomerID:     campaign.CustomerID,
                AttemptNumber:  campaign.AttemptCount,
                Amount:         campaign.FailedAmount,
            })
            
            return true, nil
        }
        
        // Publish attempt failed event
        workflow.ExecuteActivity(ctx, PublishEventActivity, DunningEvent{
            EventType:      "dunning.immediate_attempt_failed",
            CampaignID:     campaign.CampaignID,
            SubscriptionID: campaign.SubscriptionID,
            CustomerID:     campaign.CustomerID,
            AttemptNumber:  campaign.AttemptCount,
            FailureReason:  result.FailureReason,
        })
    }
    
    return false, nil
}

func executeProgressiveRetries(ctx workflow.Context, campaign *DunningWorkflowState) error {
    logger := workflow.GetLogger(ctx)
    
    // Publish progressive dunning started event
    workflow.ExecuteActivity(ctx, PublishEventActivity, DunningEvent{
        EventType:      "dunning.progressive_started",
        CampaignID:     campaign.CampaignID,
        SubscriptionID: campaign.SubscriptionID,
        CustomerID:     campaign.CustomerID,
    })
    
    for attempt := 1; attempt <= campaign.Config.ProgressiveRetries.MaxAttempts; attempt++ {
        if campaign.Status != DunningStatusActive {
            break
        }
        
        // Wait for progressive interval
        if attempt > 1 {
            intervalIndex := min(attempt-2, len(campaign.Config.ProgressiveRetries.Intervals)-1)
            interval := campaign.Config.ProgressiveRetries.Intervals[intervalIndex]
            
            campaign.NextAttemptAt = campaign.LastAttemptAt.Add(interval)
            
            // Wait with signal handling
            waitWithSignals(ctx, interval, campaign)
        }
        
        if campaign.Status != DunningStatusActive {
            break
        }
        
        campaign.AttemptCount++
        campaign.LastAttemptAt = workflow.Now(ctx)
        
        // Execute payment retry
        var result ChargeResult
        err := workflow.ExecuteActivity(ctx, ChargeCustomerActivity,
            ChargeInput{
                SubscriptionID: campaign.SubscriptionID,
                Amount:        campaign.FailedAmount,
                Currency:      campaign.Currency,
                AttemptNumber: campaign.AttemptCount,
                AttemptType:   "progressive",
                CampaignID:    campaign.CampaignID,
            }).Get(ctx, &result)
        
        // Record attempt
        attempt := DunningAttempt{
            CampaignID:    campaign.CampaignID,
            AttemptNumber: campaign.AttemptCount,
            AttemptType:   "progressive",
            Status:        result.Status,
            FailureReason: result.FailureReason,
        }
        workflow.ExecuteActivity(ctx, RecordDunningAttemptActivity, attempt)
        
        if err != nil {
            logger.Error("Progressive retry attempt failed", "attempt", campaign.AttemptCount, "error", err)
            
            // Publish attempt failed event (triggers notifications)
            workflow.ExecuteActivity(ctx, PublishEventActivity, DunningEvent{
                EventType:      "dunning.attempt_failed",
                CampaignID:     campaign.CampaignID,
                SubscriptionID: campaign.SubscriptionID,
                CustomerID:     campaign.CustomerID,
                AttemptNumber:  campaign.AttemptCount,
                FailureReason:  result.FailureReason,
                Metadata: map[string]interface{}{
                    "should_suspend": attempt == campaign.Config.EscalationRules.SuspendAfterAttempt,
                    "is_final_notice": attempt == campaign.Config.EscalationRules.FinalNoticeAttempt,
                },
            })
            
            // Handle suspension
            if attempt == campaign.Config.EscalationRules.SuspendAfterAttempt {
                workflow.ExecuteActivity(ctx, SuspendSubscriptionActivity,
                    SuspendSubscriptionInput{
                        SubscriptionID: campaign.SubscriptionID,
                        Reason:        "dunning_suspension",
                        CampaignID:    campaign.CampaignID,
                    })
            }
            
            continue
        }
        
        if result.Status == "succeeded" {
            logger.Info("Progressive retry successful", "attempt", campaign.AttemptCount)
            
            // Publish recovery event
            workflow.ExecuteActivity(ctx, PublishEventActivity, DunningEvent{
                EventType:      "dunning.payment_recovered",
                CampaignID:     campaign.CampaignID,
                SubscriptionID: campaign.SubscriptionID,
                CustomerID:     campaign.CustomerID,
                AttemptNumber:  campaign.AttemptCount,
                Amount:         campaign.FailedAmount,
                RecoveryMethod: "progressive_retry",
            })
            
            // Reactivate subscription
            workflow.ExecuteActivity(ctx, ReactivateSubscriptionActivity,
                ReactivateSubscriptionInput{
                    SubscriptionID: campaign.SubscriptionID,
                    CampaignID:    campaign.CampaignID,
                })
            
            // Signal parent workflow
            workflow.SignalExternalWorkflow(ctx, campaign.ParentWorkflowID, "",
                "subscription.state_changed",
                SubscriptionStateChangedSignal{
                    SubscriptionID: campaign.SubscriptionID,
                    NewStatus:     SubscriptionStatusActive,
                    ChangedBy:     "dunning_recovery",
                })
            
            return finalizeCampaign(ctx, campaign, DunningStatusRecovered)
        }
    }
    
    // All attempts exhausted
    logger.Info("Dunning failed - all attempts exhausted")
    
    workflow.ExecuteActivity(ctx, PublishEventActivity, DunningEvent{
        EventType:      "dunning.final_failure",
        CampaignID:     campaign.CampaignID,
        SubscriptionID: campaign.SubscriptionID,
        CustomerID:     campaign.CustomerID,
        AttemptNumber:  campaign.AttemptCount,
    })
    
    // Handle final cancellation if configured
    if campaign.Config.EscalationRules.CancelAfterAttempt > 0 && 
       campaign.AttemptCount >= campaign.Config.EscalationRules.CancelAfterAttempt {
        
        workflow.ExecuteActivity(ctx, CancelSubscriptionActivity,
            CancelSubscriptionInput{
                SubscriptionID: campaign.SubscriptionID,
                Reason:        "dunning_final_failure",
                CampaignID:    campaign.CampaignID,
            })
        
        workflow.SignalExternalWorkflow(ctx, campaign.ParentWorkflowID, "",
            "subscription.state_changed",
            SubscriptionStateChangedSignal{
                SubscriptionID: campaign.SubscriptionID,
                NewStatus:     SubscriptionStatusCancelled,
                ChangedBy:     "dunning_failure",
            })
    }
    
    return finalizeCampaign(ctx, campaign, DunningStatusFailed)
}

func setupSignalHandlers(ctx workflow.Context, campaign *DunningWorkflowState) {
    // Handle external signals
    workflow.Go(ctx, func(ctx workflow.Context) {
        for {
            selector := workflow.NewSelector(ctx)
            
            // Payment method updated signal
            paymentMethodUpdated := workflow.GetSignalChannel(ctx, "payment_method.updated")
            selector.AddReceive(paymentMethodUpdated, func(c workflow.ReceiveChannel, more bool) {
                var signal PaymentMethodUpdatedSignal
                c.Receive(ctx, &signal)
                // Trigger immediate retry
                campaign.NextAttemptAt = workflow.Now(ctx)
            })
            
            // Manual pause signal
            pauseSignal := workflow.GetSignalChannel(ctx, "dunning.pause")
            selector.AddReceive(pauseSignal, func(c workflow.ReceiveChannel, more bool) {
                c.Receive(ctx, nil)
                campaign.Status = DunningStatusPaused
            })
            
            // Manual resume signal
            resumeSignal := workflow.GetSignalChannel(ctx, "dunning.resume")
            selector.AddReceive(resumeSignal, func(c workflow.ReceiveChannel, more bool) {
                c.Receive(ctx, nil)
                if campaign.Status == DunningStatusPaused {
                    campaign.Status = DunningStatusActive
                }
            })
            
            // Manual cancellation signal
            cancelSignal := workflow.GetSignalChannel(ctx, "dunning.cancel")
            selector.AddReceive(cancelSignal, func(c workflow.ReceiveChannel, more bool) {
                c.Receive(ctx, nil)
                campaign.Status = DunningStatusCancelled
            })
            
            selector.Select(ctx)
            
            if campaign.Status == DunningStatusRecovered ||
               campaign.Status == DunningStatusFailed ||
               campaign.Status == DunningStatusCancelled {
                break
            }
        }
    })
}
```

### SubscriptionWorkflow Integration

```go
func SubscriptionWorkflow(ctx workflow.Context, input SubscriptionInput) error {
    subscription := initializeSubscription(input)
    
    for {
        // Wait for next billing date OR subscription state change
        selector := workflow.NewSelector(ctx)
        
        // Timer for next billing
        nextBilling := calculateNextBillingDate(subscription)
        billingTimer := workflow.NewTimer(ctx, time.Until(nextBilling))
        
        selector.AddFuture(billingTimer, func(f workflow.Future) {
            // Time to attempt billing
        })
        
        // Listen for subscription state changes
        stateChangeSignal := workflow.GetSignalChannel(ctx, "subscription.state_changed")
        selector.AddReceive(stateChangeSignal, func(c workflow.ReceiveChannel, more bool) {
            var signal SubscriptionStateChangedSignal
            c.Receive(ctx, &signal)
            subscription.Status = signal.NewStatus
        })
        
        selector.Select(ctx)
        
        // Only attempt billing if subscription is in billable state
        if !subscription.IsInBillableState() {
            continue // Skip billing while in past_due, cancelled, etc.
        }
        
        // Attempt single payment
        var chargeResult ChargeResult
        err := workflow.ExecuteActivity(ctx, ChargeCustomerForBillingPeriod,
            ChargeInput{
                SubscriptionID: subscription.ID,
                Amount:        subscription.Amount,
                Currency:      subscription.Currency,
            }).Get(ctx, &chargeResult)
        
        if err != nil {
            return err
        }
        
        switch chargeResult.Status {
        case "succeeded":
            subscription.Status = SubscriptionStatusActive
            subscription.LastSuccessfulCharge = workflow.Now(ctx)
            continue
            
        case "failed", "requires_payment_method":
            // Spawn independent dunning workflow
            err = spawnDunningWorkflow(ctx, subscription, chargeResult)
            if err != nil {
                return err
            }
            
            subscription.Status = SubscriptionStatusPastDue
            subscription.DunningStartedAt = workflow.Now(ctx)
            continue
        }
    }
}

func spawnDunningWorkflow(ctx workflow.Context, subscription *Subscription, chargeResult ChargeResult) error {
    // Generate campaign ID
    campaignID := fmt.Sprintf("camp_%s_%d", subscription.ID, workflow.Now(ctx).Unix())
    
    // Start independent dunning workflow
    dunningOptions := workflow.ChildWorkflowOptions{
        WorkflowID: fmt.Sprintf("dunning-%s", campaignID),
        ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
    }
    
    ctx = workflow.WithChildOptions(ctx, dunningOptions)
    
    // Fire and forget - don't wait for completion
    workflow.ExecuteChildWorkflow(ctx, DunningWorkflow, DunningWorkflowInput{
        SubscriptionID:    subscription.ID,
        CustomerID:        subscription.CustomerID,
        OrgID:            subscription.OrgID,
        FailedAmount:      subscription.Amount,
        Currency:          subscription.Currency,
        InitialFailure:    chargeResult,
        ParentWorkflowID:  workflow.GetInfo(ctx).WorkflowExecution.ID,
        DunningCampaignID: campaignID,
    })
    
    return nil
}
```

## Configuration System

### Configuration JSON Structure

```json
{
  "name": "Standard Dunning Strategy",
  "version": "1.0.0",
  
  "immediate_retries": {
    "enabled": true,
    "max_attempts": 3,
    "intervals": ["2m", "10m", "30m"],
    "failure_types": [
      "card_declined",
      "insufficient_funds", 
      "do_not_honor",
      "generic_decline"
    ]
  },
  
  "progressive_retries": {
    "enabled": true,
    "max_attempts": 5,
    "intervals": ["3d", "4d", "7d", "14d", "30d"]
  },
  
  "escalation_rules": {
    "suspend_after_attempt": 3,
    "final_notice_attempt": 4,
    "cancel_after_attempt": 5
  },
  
  "communication_strategy": {
    "channels": {
      "email": {
        "enabled": true,
        "templates": {
          "attempt_1": "dunning_gentle_reminder",
          "attempt_2": "dunning_urgent_action",
          "attempt_3": "dunning_critical_notice",
          "attempt_4": "dunning_final_notice",
          "recovery_success": "dunning_thank_you"
        }
      },
      "sms": {
        "enabled": true,
        "start_after_attempt": 3,
        "templates": {
          "attempt_3": "sms_critical",
          "attempt_4": "sms_final"
        }
      }
    },
    
    "timing": {
      "send_immediately_after_attempt": true,
      "respect_timezone": true,
      "avoid_weekends": false
    }
  },
  
  "business_rules": {
    "grace_period": {
      "suspend_service_after": "7d",
      "maintain_data_for": "90d"
    },
    "payment_flexibility": {
      "allow_partial_payments": false,
      "auto_update_expired_cards": true
    }
  },
  
  "token_settings": {
    "default_max_uses": 5,
    "default_expiry_hours": 72,
    "customer_tier_overrides": {
      "vip": {
        "max_uses": 10,
        "expiry_hours": 168
      },
      "premium": {
        "max_uses": 7,
        "expiry_hours": 120
      }
    }
  }
}
```

### Configuration Resolution Service

```go
type DunningConfigService struct {
    db     Database
    cache  Cache
    logger Logger
}

func (s *DunningConfigService) ResolveConfiguration(input ConfigResolutionInput) (*DunningConfig, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("dunning_config:%s:%s:%s", input.OrgID, input.CustomerSegment, input.SubscriptionTier)
    if cached := s.cache.Get(cacheKey); cached != nil {
        return cached.(*DunningConfig), nil
    }
    
    // Get applicable configurations in priority order
    configs, err := s.getApplicableConfigs(input)
    if err != nil {
        return nil, err
    }
    
    // Merge configurations
    finalConfig := s.mergeConfigurations(configs)
    
    // Apply defaults and validate
    finalConfig = s.applyDefaults(finalConfig)
    err = s.validateConfig(finalConfig)
    if err != nil {
        return nil, fmt.Errorf("invalid merged configuration: %w", err)
    }
    
    // Cache result
    s.cache.Set(cacheKey, finalConfig, 1*time.Hour)
    
    return finalConfig, nil
}
```

## Token Security System

### Token Generation Service

```go
type PaymentUpdateTokenService struct {
    secretKey []byte
    db        Database
    cache     Cache
}

type PaymentUpdateToken struct {
    TokenID           string                 `json:"token_id"`
    SubscriptionID    string                 `json:"subscription_id"`
    CustomerID        string                 `json:"customer_id"`
    OrgID            string                 `json:"org_id"`
    DunningCampaignID string                 `json:"dunning_campaign_id"`
    ExpiresAt         time.Time              `json:"expires_at"`
    MaxUses           int                    `json:"max_uses"`
    UsedCount         int                    `json:"used_count"`
    AllowedActions    map[string]bool        `json:"allowed_actions"`
    CreatedAt         time.Time              `json:"created_at"`
    CreatedBy         string                 `json:"created_by"`
    Signature         string                 `json:"signature"`
}

func (s *PaymentUpdateTokenService) GeneratePaymentUpdateLink(
    ctx context.Context,
    req GenerateTokenRequest) (*PaymentUpdateLink, error) {
    
    // Get token limits from configuration
    limits := s.getTokenLimits(req.OrgID, req.CustomerTier)
    
    // Create token
    token := &PaymentUpdateToken{
        TokenID:           generateSecureTokenID(),
        SubscriptionID:    req.SubscriptionID,
        CustomerID:        req.CustomerID,
        OrgID:            req.OrgID,
        DunningCampaignID: req.DunningCampaignID,
        ExpiresAt:        time.Now().Add(time.Duration(limits.ExpiryHours) * time.Hour),
        MaxUses:          limits.MaxUses,
        UsedCount:        0,
        AllowedActions:   req.AllowedActions,
        CreatedAt:        time.Now(),
        CreatedBy:        req.CreatedBy,
    }
    
    // Sign token
    token.Signature = s.signToken(token)
    
    // Store in database
    err := s.storeToken(ctx, token)
    if err != nil {
        return nil, err
    }
    
    // Build secure URL
    link := &PaymentUpdateLink{
        URL:       s.buildSecureURL(token),
        TokenID:   token.TokenID,
        ExpiresAt: token.ExpiresAt,
        MaxUses:   token.MaxUses,
    }
    
    return link, nil
}

func (s *PaymentUpdateTokenService) signToken(token *PaymentUpdateToken) string {
    payload := fmt.Sprintf("%s:%s:%s:%s:%d:%d",
        token.TokenID,
        token.SubscriptionID,
        token.CustomerID,
        token.OrgID,
        token.ExpiresAt.Unix(),
        token.MaxUses)
    
    h := hmac.New(sha256.New, s.secretKey)
    h.Write([]byte(payload))
    return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func (s *PaymentUpdateTokenService) ValidateToken(
    ctx context.Context,
    tokenData string,
    clientIP string,
    checkOnly bool) (*SecurePaymentSession, error) {
    
    // Decode token
    token, err := s.decodeToken(tokenData)
    if err != nil {
        return nil, ErrInvalidToken
    }
    
    // Verify signature
    expectedSignature := s.signToken(token)
    if !hmac.Equal([]byte(token.Signature), []byte(expectedSignature)) {
        return nil, ErrTokenTampered
    }
    
    // Check expiration
    if time.Now().After(token.ExpiresAt) {
        return nil, ErrTokenExpired
    }
    
    // Check usage limit
    if token.UsedCount >= token.MaxUses {
        return nil, ErrTokenMaxUsesExceeded
    }
    
    // Increment usage if not just checking
    if !checkOnly {
        err = s.incrementTokenUsage(ctx, token.TokenID, clientIP)
        if err != nil {
            return nil, err
        }
    }
    
    // Create session
    session := &SecurePaymentSession{
        SessionID:      generateSessionID(),
        TokenID:        token.TokenID,
        SubscriptionID: token.SubscriptionID,
        CustomerID:     token.CustomerID,
        AllowedActions: token.AllowedActions,
        ExpiresAt:      token.ExpiresAt,
        RemainingUses:  token.MaxUses - token.UsedCount - 1,
    }
    
    return session, nil
}
```

## API Endpoints

### Token Verification APIs

```go
// POST /api/v1/payment-tokens/verify
func (h *PaymentTokenHandler) VerifyToken(w http.ResponseWriter, r *http.Request) {
    var req VerifyTokenRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
        return
    }
    
    session, err := h.tokenService.ValidateToken(r.Context(), req.Token, req.ClientIP, true)
    if err != nil {
        response := &VerifyTokenResponse{
            Valid: false,
            Error: h.mapTokenError(err),
        }
        h.writeJSON(w, http.StatusOK, response)
        return
    }
    
    response := &VerifyTokenResponse{
        Valid:   true,
        Session: session,
    }
    h.writeJSON(w, http.StatusOK, response)
}

// POST /api/v1/payment-tokens/activate
func (h *PaymentTokenHandler) ActivateToken(w http.ResponseWriter, r *http.Request) {
    var req ActivateTokenRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
        return
    }
    
    session, err := h.tokenService.ValidateToken(r.Context(), req.Token, req.ClientIP, false)
    if err != nil {
        response := &ActivateTokenResponse{
            Success: false,
            Error:   h.mapTokenError(err),
        }
        h.writeJSON(w, http.StatusOK, response)
        return
    }
    
    // Create server-side session
    sessionID, err := h.sessionService.CreatePaymentSession(r.Context(), session)
    if err != nil {
        h.writeError(w, http.StatusInternalServerError, "session_creation_failed", "Failed to create session")
        return
    }
    
    response := &ActivateTokenResponse{
        Success:   true,
        SessionID: sessionID,
        Session:   session,
    }
    h.writeJSON(w, http.StatusOK, response)
}
```

### Admin Token Generation API

```go
// POST /api/v1/admin/subscriptions/{subscription_id}/payment-tokens
func (h *AdminHandler) GeneratePaymentToken(w http.ResponseWriter, r *http.Request) {
    subscriptionID := chi.URLParam(r, "subscription_id")
    orgID := r.Context().Value("org_id").(string)
    adminUserID := r.Context().Value("user_id").(string)
    
    var req AdminGenerateTokenRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
        return
    }
    
    // Validate permissions
    if !h.authService.CanGeneratePaymentTokens(adminUserID, orgID) {
        h.writeError(w, http.StatusForbidden, "insufficient_permissions", "Cannot generate payment tokens")
        return
    }
    
    // Get subscription details
    subscription, err := h.subscriptionService.GetSubscription(r.Context(), orgID, subscriptionID)
    if err != nil {
        h.writeError(w, http.StatusNotFound, "subscription_not_found", "Subscription not found")
        return
    }
    
    // Generate token
    tokenReq := GenerateTokenRequest{
        SubscriptionID: subscriptionID,
        CustomerID:     subscription.CustomerID,
        OrgID:         orgID,
        AllowedActions: req.AllowedActions,
        CreatedBy:     fmt.Sprintf("admin:%s", adminUserID),
    }
    
    if req.MaxUses != nil {
        tokenReq.MaxUsesOverride = req.MaxUses
    }
    if req.ExpiryHours != nil {
        tokenReq.ExpiryOverride = time.Duration(*req.ExpiryHours) * time.Hour
    }
    
    token, err := h.tokenService.GeneratePaymentUpdateLink(r.Context(), tokenReq)
    if err != nil {
        h.writeError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate token")
        return
    }
    
    // Audit log
    h.auditLog.LogAdminAction(r.Context(), AuditLogEntry{
        AdminUserID:  adminUserID,
        Action:       "generate_payment_token",
        ResourceType: "subscription",
        ResourceID:   subscriptionID,
        Details: map[string]interface{}{
            "reason":     req.Reason,
            "token_id":   token.TokenID,
            "max_uses":   token.MaxUses,
            "expires_at": token.ExpiresAt,
        },
    })
    
    response := &AdminGenerateTokenResponse{
        Success: true,
        Token:   token,
    }
    h.writeJSON(w, http.StatusOK, response)
}
```

## Event System

### Event Types and Structure

```go
type DunningEvent struct {
    EventType      string                 `json:"event_type"`
    CampaignID     string                 `json:"campaign_id"`
    SubscriptionID string                 `json:"subscription_id"`
    CustomerID     string                 `json:"customer_id"`
    OrgID          string                 `json:"org_id"`
    AttemptNumber  int                    `json:"attempt_number"`
    Amount         int                    `json:"amount"`
    Currency       string                 `json:"currency"`
    FailureReason  string                 `json:"failure_reason,omitempty"`
    RecoveryMethod string                 `json:"recovery_method,omitempty"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
    Timestamp      time.Time              `json:"timestamp"`
}

// Event types
const (
    EventDunningStarted           = "dunning.started"
    EventDunningImmediateRecovery = "dunning.immediate_recovery"
    EventDunningProgressiveStarted = "dunning.progressive_started"
    EventDunningAttemptFailed     = "dunning.attempt_failed"
    EventDunningPaymentRecovered  = "dunning.payment_recovered"
    EventDunningFinalFailure      = "dunning.final_failure"
    EventDunningPaused            = "dunning.paused"
    EventDunningResumed           = "dunning.resumed"
)
```

### Notification Service

```go
type NotificationService struct {
    emailProvider EmailProvider
    smsProvider   SMSProvider
    templateSvc   TemplateService
    prefSvc       CustomerPreferenceService
    pubsub        PubSubClient
}

func (ns *NotificationService) Start() {
    ns.pubsub.Subscribe("dunning.*", ns.handleDunningEvent)
}

func (ns *NotificationService) handleDunningEvent(event DunningEvent) error {
    switch event.EventType {
    case EventDunningAttemptFailed:
        return ns.handleAttemptFailed(event)
    case EventDunningPaymentRecovered:
        return ns.handlePaymentRecovered(event)
    case EventDunningFinalFailure:
        return ns.handleFinalFailure(event)
    }
    return nil
}

func (ns *NotificationService) handleAttemptFailed(event DunningEvent) error {
    // Get customer preferences
    prefs, err := ns.prefSvc.GetNotificationPreferences(event.CustomerID)
    if err != nil {
        return err
    }
    
    // Get communication strategy
    strategy := ns.getCommunicationStrategy(event.AttemptNumber)
    
    // Send notifications concurrently
    var wg sync.WaitGroup
    
    if prefs.EmailEnabled && strategy.SendEmail {
        wg.Add(1)
        go func() {
            defer wg.Done()
            ns.sendDunningEmail(event, strategy.EmailTemplate)
        }()
    }
    
    if prefs.SMSEnabled && strategy.SendSMS {
        wg.Add(1)
        go func() {
            defer wg.Done()
            ns.sendDunningSMS(event, strategy.SMSTemplate)
        }()
    }
    
    wg.Wait()
    return nil
}
```

## Use Cases & Examples

### Use Case 1: Standard Payment Failure Recovery

**Scenario**: Monthly $29.99 subscription payment fails due to expired credit card.

**Timeline**:
```
March 1, 09:00: Subscription billing attempt fails
March 1, 09:01: DunningWorkflow starts, creates campaign
March 1, 09:02: Immediate retry #1 (1 min later) - fails
March 1, 09:07: Immediate retry #2 (5 min later) - fails  
March 1, 09:22: Immediate retry #3 (15 min later) - fails
March 1, 09:23: Event published → Email sent "Payment Failed - Update Required"

March 4, 09:23: Progressive retry #1 (3 days later) - fails
March 4, 09:24: Event published → Email sent "Urgent: Payment Action Required"

March 8, 09:23: Progressive retry #2 (4 days later) - fails
March 8, 09:24: Event published → Email + SMS sent "Critical: Account Suspension Warning"

March 15, 09:23: Progressive retry #3 (7 days later) - fails
March 15, 09:24: Subscription suspended
March 15, 09:25: Event published → Email + SMS "Account Suspended"

March 29, 09:23: Progressive retry #4 (14 days later) - fails
March 29, 09:24: Event published → Email + SMS "Final Notice"

April 12, 09:23: Progressive retry #5 (14 days later) - fails
April 12, 09:24: Subscription cancelled
April 12, 09:25: DunningWorkflow ends
```

### Use Case 2: Quick Recovery via Payment Method Update

**Scenario**: Customer receives email, updates payment method, payment succeeds.

**Timeline**:
```
March 1, 09:00: Payment fails, dunning starts
March 1, 09:23: Email sent with secure payment update link
March 1, 14:30: Customer clicks link, updates payment method
March 1, 14:31: Payment method update triggers signal to DunningWorkflow
March 1, 14:32: DunningWorkflow attempts immediate retry - succeeds
March 1, 14:33: Subscription reactivated, recovery email sent
March 1, 14:34: DunningWorkflow ends successfully
```

### Use Case 3: VIP Customer Treatment

**Scenario**: High-value customer gets enhanced dunning strategy.

**Configuration**:
```json
{
  "customer_segment": "vip",
  "progressive_retries": {
    "max_attempts": 8,
    "intervals": ["1d", "3d", "7d", "7d", "14d", "14d", "30d", "30d"]
  },
  "escalation_rules": {
    "suspend_after_attempt": 6,
    "cancel_after_attempt": 8
  },
  "communication_strategy": {
    "channels": {
      "email": {
        "templates": {
          "attempt_1": "dunning_vip_gentle",
          "attempt_2": "dunning_vip_personal"
        }
      },
      "phone_call": {
        "enabled": true,
        "start_after_attempt": 3
      }
    }
  }
}
```

### Use Case 4: Admin-Generated Recovery Link

**Scenario**: Customer calls support, agent generates new payment link.

**Admin Action**:
```typescript
// Admin generates token via API
const response = await fetch('/api/v1/admin/subscriptions/sub_123/payment-tokens', {
  method: 'POST',
  body: JSON.stringify({
    reason: 'customer_request',
    max_uses: 3,
    expiry_hours: 48,
    allowed_actions: {
      update_payment_method: true,
      retry_payment: true,
      pause_subscription: true
    },
    notes: 'Customer called, card was stolen, needs new payment method'
  })
});

// Agent emails customer the secure link
```

### Use Case 5: Long-Running Subscription During Dunning

**Scenario**: Monthly subscription in dunning for 45 days, billing continues after recovery.

**Timeline**:
```
March 1: Payment fails, dunning starts, subscription status = "past_due"
April 1: SubscriptionWorkflow checks status = "past_due" → skips April billing
May 1: SubscriptionWorkflow checks status = "past_due" → skips May billing
May 15: Payment recovers, subscription status = "active"
June 1: SubscriptionWorkflow checks status = "active" → bills June normally ($29.99, not $89.97)
```

## Integration Patterns

### Remix Backend Integration

#### Token Verification in Loader

```typescript
// app/routes/payment.update.tsx
import { json, type LoaderFunctionArgs } from "@remix-run/node";

export async function loader({ request }: LoaderFunctionArgs) {
  const url = new URL(request.url);
  const token = url.searchParams.get("token");
  
  if (!token) {
    throw new Response("Missing payment token", { status: 400 });
  }
  
  // Verify token (no usage increment)
  const response = await fetch(`${process.env.API_BASE_URL}/api/v1/payment-tokens/verify`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${process.env.API_SECRET}`,
    },
    body: JSON.stringify({
      token,
      client_ip: getClientIP(request),
      user_agent: request.headers.get("User-Agent"),
    }),
  });
  
  const result = await response.json();
  
  if (!result.valid) {
    switch (result.error?.code) {
      case "token_expired":
        throw new Response("This payment link has expired", { status: 410 });
      case "token_max_uses_exceeded":
        throw new Response("This payment link has been used too many times", { status: 410 });
      default:
        throw new Response("Invalid payment link", { status: 400 });
    }
  }
  
  return json({
    session: result.session,
    token: token,
  });
}
```

#### Token Activation in Action

```typescript
// app/routes/payment.update.tsx
export async function action({ request }: ActionFunctionArgs) {
  const formData = await request.formData();
  const action = formData.get("_action") as string;
  const token = formData.get("token") as string;
  
  if (action === "update_payment_method") {
    // Activate token (increments usage)
    const activateResponse = await fetch(`${process.env.API_BASE_URL}/api/v1/payment-tokens/activate`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json", 
        "Authorization": `Bearer ${process.env.API_SECRET}`,
      },
      body: JSON.stringify({
        token,
        client_ip: getClientIP(request),
        user_agent: request.headers.get("User-Agent"),
      }),
    });
    
    const activateResult = await activateResponse.json();
    
    if (!activateResult.success) {
      return json({ error: activateResult.error }, { status: 400 });
    }
    
    // Use session for subsequent operations
    const sessionId = activateResult.session_id;
    
    // Update payment method using session
    const updateResponse = await fetch(`${process.env.API_BASE_URL}/api/v1/payment-sessions/${sessionId}/update-payment-method`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${process.env.API_SECRET}`,
      },
      body: JSON.stringify({
        payment_method_id: formData.get("payment_method_id"),
      }),
    });
    
    return json(await updateResponse.json());
  }
  
  return json({ error: "Unknown action" }, { status: 400 });
}
```

### Temporal Activity Implementations

```go
// Activities for DunningWorkflow

func ChargeCustomerActivity(ctx context.Context, input ChargeInput) (ChargeResult, error) {
    // Implement payment processing
    return paymentService.ProcessPayment(ctx, input)
}

func CreateDunningCampaignActivity(ctx context.Context, campaign *DunningWorkflowState) error {
    // Create campaign record in database
    return campaignService.CreateCampaign(ctx, campaign)
}

func RecordDunningAttemptActivity(ctx context.Context, attempt DunningAttempt) error {
    // Record attempt in database
    return attemptService.RecordAttempt(ctx, attempt)
}

func PublishEventActivity(ctx context.Context, event DunningEvent) error {
    // Publish event to NATS
    return eventService.PublishEvent(ctx, event)
}

func ResolveDunningConfigActivity(ctx context.Context, input ConfigResolutionInput) (DunningConfig, error) {
    // Resolve configuration for this specific context
    return configService.ResolveConfiguration(ctx, input)
}

func SuspendSubscriptionActivity(ctx context.Context, input SuspendSubscriptionInput) error {
    // Suspend subscription service
    return subscriptionService.SuspendSubscription(ctx, input)
}

func ReactivateSubscriptionActivity(ctx context.Context, input ReactivateSubscriptionInput) error {
    // Reactivate subscription service  
    return subscriptionService.ReactivateSubscription(ctx, input)
}

func GeneratePaymentUpdateTokenActivity(ctx context.Context, req GenerateTokenRequest) (*PaymentUpdateLink, error) {
    // Generate secure payment update token
    return tokenService.GeneratePaymentUpdateLink(ctx, req)
}
```

### Database Indexes for Performance

```sql
-- Core dunning indexes
CREATE INDEX idx_dunning_campaigns_subscription ON dunning_campaigns(org_id, subscription_id);
CREATE INDEX idx_dunning_campaigns_customer ON dunning_campaigns(org_id, customer_id);
CREATE INDEX idx_dunning_campaigns_status ON dunning_campaigns(org_id, status);
CREATE INDEX idx_dunning_campaigns_next_attempt ON dunning_campaigns(next_attempt_at) WHERE status = 'active';
CREATE INDEX idx_dunning_campaigns_temporal ON dunning_campaigns(temporal_workflow_id);

-- Attempt tracking indexes
CREATE INDEX idx_dunning_attempts_campaign ON dunning_attempts(org_id, dunning_campaign_id);
CREATE INDEX idx_dunning_attempts_date ON dunning_attempts(attempted_at);
CREATE INDEX idx_dunning_attempts_status ON dunning_attempts(status);

-- Communication indexes
CREATE INDEX idx_dunning_communications_campaign ON dunning_communications(org_id, dunning_campaign_id);
CREATE INDEX idx_dunning_communications_customer ON dunning_communications(org_id, customer_id);
CREATE INDEX idx_dunning_communications_sent ON dunning_communications(sent_at);

-- Token indexes
CREATE INDEX idx_payment_tokens_expires_at ON payment_update_tokens(expires_at);
CREATE INDEX idx_payment_tokens_subscription ON payment_update_tokens(org_id, subscription_id);
CREATE INDEX idx_payment_tokens_admin ON payment_update_tokens(org_id, admin_generated, created_at);

-- Analytics indexes
CREATE INDEX idx_dunning_analytics_org_date ON dunning_analytics_daily(org_id, date);
CREATE INDEX idx_customer_dunning_risk ON customer_dunning_history(org_id, dunning_risk_tier);
```

### Configuration Management

```go
// Example configuration service implementation
type ConfigurationService struct {
    db    Database
    cache Cache
}

func (s *ConfigurationService) GetDunningConfiguration(orgID, customerTier string) (*DunningConfig, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("dunning_config:%s:%s", orgID, customerTier)
    if cached := s.cache.Get(cacheKey); cached != nil {
        return cached.(*DunningConfig), nil
    }
    
    // Query configurations in priority order
    configs, err := s.queryConfigurations(orgID, customerTier)
    if err != nil {
        return nil, err
    }
    
    // Merge configurations
    merged := s.mergeConfigurations(configs)
    
    // Cache result
    s.cache.Set(cacheKey, merged, 30*time.Minute)
    
    return merged, nil
}

func (s *ConfigurationService) mergeConfigurations(configs []DunningConfiguration) *DunningConfig {
    // Start with defaults
    result := s.getDefaultConfiguration()
    
    // Apply each configuration in priority order
    for _, config := range configs {
        result = s.deepMerge(result, config.Config)
    }
    
    return result
}
```

## Architecture Impact

### Performance Considerations
- **Workflow Scaling**: Each subscription can have independent dunning without affecting others
- **Database Load**: Proper indexing ensures efficient queries even with millions of campaigns
- **Event Processing**: NATS handles high-volume event processing for notifications
- **Caching**: Configuration caching reduces database load for common queries

### Monitoring & Observability
- **Workflow Metrics**: Track dunning workflow execution times, success rates
- **Campaign Analytics**: Monitor recovery rates, communication effectiveness
- **Token Security**: Track token usage patterns, detect abuse
- **System Health**: Monitor event processing lag, database performance

### Security Considerations
- **Token Security**: HMAC signatures prevent tampering, usage limits prevent abuse
- **API Authentication**: All admin APIs require proper authentication and authorization
- **Audit Trail**: Complete logging of all dunning activities for compliance
- **Data Privacy**: Customer communication preferences respected

### Scalability Design
- **Independent Workflows**: Dunning scales independently from billing
- **Event-Driven**: Messaging system scales independently from business logic
- **Database Partitioning**: Tables can be partitioned by org_id for large installations
- **Configuration Caching**: Reduces database load for configuration resolution

This specification provides a comprehensive foundation for implementing the dunning feature in Payloop, with clear separation of concerns, robust security, and scalable architecture patterns.