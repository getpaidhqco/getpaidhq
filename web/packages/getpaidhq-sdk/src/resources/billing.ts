import { HttpClient } from '../utils/http-client';
import { ReminderConfigDTO } from '../types';

export class BillingResource {
  private readonly resourcePath = '/api/billing';

  constructor(private httpClient: HttpClient) {}

  /** Get the invoice reminder configuration (GET /api/billing/reminder-config). */
  async getReminderConfig(): Promise<ReminderConfigDTO> {
    return this.httpClient.get<ReminderConfigDTO>(`${this.resourcePath}/reminder-config`);
  }

  /** Update the invoice reminder configuration (PUT /api/billing/reminder-config). */
  async updateReminderConfig(data: ReminderConfigDTO): Promise<ReminderConfigDTO> {
    return this.httpClient.put<ReminderConfigDTO>(`${this.resourcePath}/reminder-config`, data);
  }
}
