import { HttpClient } from '../utils/http-client';
import { AddItemRequest, CartResponse, RemoveItemRequest } from '../types';

export class CartsResource {
  private readonly resourcePath = '/api/carts';

  constructor(private httpClient: HttpClient) {}

  /** Add an item to a cart (POST /api/carts/{id}/add). */
  async addItem(cartId: string, data: AddItemRequest): Promise<CartResponse> {
    return this.httpClient.post<CartResponse>(`${this.resourcePath}/${cartId}/add`, data);
  }

  /** Remove an item from a cart (POST /api/carts/{id}/remove). */
  async removeItem(cartId: string, data: RemoveItemRequest): Promise<CartResponse> {
    return this.httpClient.post<CartResponse>(`${this.resourcePath}/${cartId}/remove`, data);
  }
}
