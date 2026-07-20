import { HttpClient } from '../utils/http-client';
import { CreateGatewayRequest, GatewayResponse } from '../types';

export class GatewaysResource {
  private readonly resourcePath = '/api/gateways';

  constructor(private httpClient: HttpClient) {}

  /** Create a payment service provider gateway (POST /api/gateways). */
  async create(data: CreateGatewayRequest): Promise<GatewayResponse> {
    return this.httpClient.post<GatewayResponse>(this.resourcePath, data);
  }
}
