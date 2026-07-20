import { Metadata } from './common';

/** Dunning campaign (spec: DunningCampaignResponse). */
export interface DunningCampaignResponse {
  completed_at: string;
  created_at: string;
  currency: string;
  customer_id: string;
  failed_amount: number;
  final_failure_reason: string;
  id: string;
  immediate_attempts: number;
  initial_failure_reason: string;
  last_attempt_at: string;
  metadata: Metadata;
  next_attempt_at: string;
  progressive_attempts: number;
  recovered_amount: number;
  recovered_at: string;
  recovery_method: string;
  started_at: string;
  status: string;
  subscription_id: string;
  total_attempts: number;
  updated_at: string;
}

/** Dunning attempt (spec: DunningAttemptResponse). */
export interface DunningAttemptResponse {
  amount: number;
  attempt_number: number;
  attempt_type: string;
  attempted_at: string;
  completed_at: string;
  created_at: string;
  currency: string;
  dunning_campaign_id: string;
  failure_code: string;
  failure_reason: string;
  id: string;
  metadata: Metadata;
  payment_method_id: string;
  processing_time_ms: number;
  status: string;
  subscription_id: string;
  triggered_by: string;
}

/** Dunning configuration (spec: DunningConfigurationResponse). */
export interface DunningConfigurationResponse {
  ab_test_percentage: number;
  applies_to: string;
  config: Record<string, any>;
  created_at: string;
  created_by: string;
  description: string;
  id: string;
  is_ab_test: boolean;
  name: string;
  priority: number;
  status: string;
  target_rules: Record<string, any>;
  updated_at: string;
}

/** Communication channel config (spec: config.communication_strategy.channels values). */
export interface DunningCommunicationChannel {
  enabled: boolean;
  start_after_attempt: number;
  templates: Record<string, string>;
}

/** Communication strategy (spec: config.communication_strategy). */
export interface DunningCommunicationStrategy {
  channels: Record<string, DunningCommunicationChannel>;
}

/** Escalation rules (spec: config.escalation_rules). */
export interface DunningEscalationRules {
  cancel_after_attempt: number;
  final_notice_attempt: number;
  suspend_after_attempt: number;
}

/** Retry rules (spec: config.immediate_retries / config.progressive_retries). */
export interface DunningRetryRules {
  enabled: boolean;
  failure_types: string[];
  intervals: string[];
  max_attempts: number;
}

/** Token settings (spec: config.token_settings). */
export interface DunningTokenSettings {
  default_expiry_hours: number;
  default_max_uses: number;
}

/** Dunning configuration body (spec: CreateDunningConfigurationRequest.config). */
export interface DunningConfig {
  communication_strategy?: DunningCommunicationStrategy;
  escalation_rules?: DunningEscalationRules;
  immediate_retries?: DunningRetryRules;
  progressive_retries?: DunningRetryRules;
  token_settings?: DunningTokenSettings;
}

/** Create dunning configuration input (spec: CreateDunningConfigurationRequest). */
export interface CreateDunningConfigurationRequest {
  ab_test_percentage?: number;
  applies_to: string;
  config: DunningConfig;
  description?: string;
  is_ab_test?: boolean;
  name: string;
  priority?: number;
  target_rules?: Record<string, any>;
}

/** Update dunning configuration input (spec: UpdateDunningConfigurationRequest). */
export interface UpdateDunningConfigurationRequest {
  ab_test_percentage?: number;
  applies_to?: string;
  config?: DunningConfig;
  description?: string;
  is_ab_test?: boolean;
  name?: string;
  priority?: number;
  status?: string;
  target_rules?: Record<string, any>;
}

/** Update dunning campaign input (spec: UpdateDunningCampaignRequest). */
export interface UpdateDunningCampaignRequest {
  reason?: string;
  status: string;
}

/** Trigger manual attempt input (spec: TriggerManualAttemptRequest). */
export interface TriggerManualAttemptRequest {
  payment_method_id?: string;
}

/** Activate payment update token input (spec: ActivatePaymentTokenRequest). */
export interface ActivatePaymentTokenRequest {
  token_id: string;
}

/** Verify payment update token input (spec: VerifyPaymentTokenRequest). */
export interface VerifyPaymentTokenRequest {
  token_id: string;
}

/** Create payment update token input (spec: CreatePaymentTokenRequest). */
export interface CreatePaymentTokenRequest {
  admin_notes?: string;
  admin_reason?: string;
  allowed_actions?: Record<string, boolean>;
  expiry_hours?: number;
  max_uses?: number;
}

/** Payment update token (spec: PaymentUpdateTokenResponse). */
export interface PaymentUpdateTokenResponse {
  admin_generated: boolean;
  allowed_actions: Record<string, boolean>;
  created_at: string;
  created_by: string;
  customer_id: string;
  dunning_campaign_id: string;
  expires_at: string;
  last_used_at: string;
  max_uses: number;
  status: string;
  subscription_id: string;
  token_id: string;
  used_count: number;
}
