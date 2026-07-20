/** Create webhook subscription input (spec: CreateWebhookSubscriptionRequest). */
export interface CreateWebhookSubscriptionRequest {
  events: string[];
  secret?: string;
  url: string;
}
