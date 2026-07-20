import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import {
  DunningListResponse,
  DunningCampaignResponse,
  DunningAttemptResponse,
  DunningConfigurationResponse,
  UpdateDunningCampaignRequest,
  TriggerManualAttemptRequest,
  CreateDunningConfigurationRequest,
  UpdateDunningConfigurationRequest,
  VerifyPaymentTokenRequest,
  ActivatePaymentTokenRequest,
  CreatePaymentTokenRequest,
  PaymentUpdateTokenResponse,
  PaginationParams,
} from '../types';

export class DunningResource {
  constructor(private httpClient: HttpClient) {}

  // Campaigns

  /** List dunning campaigns (GET /api/dunning/campaigns). */
  async listCampaigns(
    params?: PaginationParams,
  ): Promise<DunningListResponse<DunningCampaignResponse>> {
    return this.httpClient.get<DunningListResponse<DunningCampaignResponse>>(
      `/api/dunning/campaigns${buildQueryString(params)}`,
    );
  }

  /** Get a dunning campaign (GET /api/dunning/campaigns/{id}). */
  async getCampaign(campaignId: string): Promise<DunningCampaignResponse> {
    return this.httpClient.get<DunningCampaignResponse>(`/api/dunning/campaigns/${campaignId}`);
  }

  /** Update a dunning campaign (PATCH /api/dunning/campaigns/{id}). */
  async updateCampaign(
    campaignId: string,
    data: UpdateDunningCampaignRequest,
  ): Promise<DunningCampaignResponse> {
    return this.httpClient.patch<DunningCampaignResponse>(
      `/api/dunning/campaigns/${campaignId}`,
      data,
    );
  }

  /** List a campaign's attempts (GET /api/dunning/campaigns/{id}/attempts). */
  async listCampaignAttempts(
    campaignId: string,
  ): Promise<DunningListResponse<DunningAttemptResponse>> {
    return this.httpClient.get<DunningListResponse<DunningAttemptResponse>>(
      `/api/dunning/campaigns/${campaignId}/attempts`,
    );
  }

  /** Trigger a manual dunning attempt (POST /api/dunning/campaigns/{id}/attempts). */
  async triggerManualAttempt(
    campaignId: string,
    data?: TriggerManualAttemptRequest,
  ): Promise<DunningAttemptResponse> {
    return this.httpClient.post<DunningAttemptResponse>(
      `/api/dunning/campaigns/${campaignId}/attempts`,
      data ?? {},
    );
  }

  /** List a campaign's communications (GET /api/dunning/campaigns/{id}/communications). */
  async listCampaignCommunications(campaignId: string): Promise<DunningListResponse> {
    return this.httpClient.get<DunningListResponse>(
      `/api/dunning/campaigns/${campaignId}/communications`,
    );
  }

  // Configurations

  /** List dunning configurations (GET /api/dunning/configurations). */
  async listConfigurations(): Promise<DunningListResponse<DunningConfigurationResponse>> {
    return this.httpClient.get<DunningListResponse<DunningConfigurationResponse>>(
      '/api/dunning/configurations',
    );
  }

  /** Create a dunning configuration (POST /api/dunning/configurations). */
  async createConfiguration(
    data: CreateDunningConfigurationRequest,
  ): Promise<DunningConfigurationResponse> {
    return this.httpClient.post<DunningConfigurationResponse>('/api/dunning/configurations', data);
  }

  /** Get a dunning configuration (GET /api/dunning/configurations/{id}). */
  async getConfiguration(configId: string): Promise<DunningConfigurationResponse> {
    return this.httpClient.get<DunningConfigurationResponse>(
      `/api/dunning/configurations/${configId}`,
    );
  }

  /** Update a dunning configuration (PATCH /api/dunning/configurations/{id}). */
  async updateConfiguration(
    configId: string,
    data: UpdateDunningConfigurationRequest,
  ): Promise<DunningConfigurationResponse> {
    return this.httpClient.patch<DunningConfigurationResponse>(
      `/api/dunning/configurations/${configId}`,
      data,
    );
  }

  // Payment update tokens

  /** Verify a payment update token (POST /api/payment-tokens/verify). */
  async verifyPaymentToken(data: VerifyPaymentTokenRequest): Promise<PaymentUpdateTokenResponse> {
    return this.httpClient.post<PaymentUpdateTokenResponse>('/api/payment-tokens/verify', data);
  }

  /** Activate a payment update token (POST /api/payment-tokens/activate). */
  async activatePaymentToken(
    data: ActivatePaymentTokenRequest,
  ): Promise<PaymentUpdateTokenResponse> {
    return this.httpClient.post<PaymentUpdateTokenResponse>('/api/payment-tokens/activate', data);
  }

  /**
   * Generate an admin payment update token for a subscription
   * (POST /api/admin/subscriptions/{id}/payment-tokens).
   */
  async createSubscriptionPaymentToken(
    subscriptionId: string,
    data?: CreatePaymentTokenRequest,
  ): Promise<PaymentUpdateTokenResponse> {
    return this.httpClient.post<PaymentUpdateTokenResponse>(
      `/api/admin/subscriptions/${subscriptionId}/payment-tokens`,
      data ?? {},
    );
  }
}
