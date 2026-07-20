import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import {
  CreateProductRequest,
  UpdateProductRequest,
  ProductResponse,
  CreateVariantRequest,
  UpdateVariantRequest,
  VariantResponse,
  CreatePriceRequest,
  PriceResponse,
  ListResponse,
  PaginationParams,
} from '../types';

export class ProductsResource {
  private readonly resourcePath = '/api/products';

  constructor(private httpClient: HttpClient) {}

  /** List products (GET /api/products). */
  async list(params?: PaginationParams): Promise<ListResponse<ProductResponse>> {
    return this.httpClient.get<ListResponse<ProductResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Create a product (POST /api/products). */
  async create(data: CreateProductRequest): Promise<ProductResponse> {
    return this.httpClient.post<ProductResponse>(this.resourcePath, data);
  }

  /** Get a product by id (GET /api/products/{id}). */
  async get(productId: string): Promise<ProductResponse> {
    return this.httpClient.get<ProductResponse>(`${this.resourcePath}/${productId}`);
  }

  /** Update a product (PATCH /api/products/{id}). */
  async update(productId: string, data: UpdateProductRequest): Promise<ProductResponse> {
    return this.httpClient.patch<ProductResponse>(`${this.resourcePath}/${productId}`, data);
  }

  /** Delete a product (DELETE /api/products/{id}). */
  async delete(productId: string): Promise<void> {
    return this.httpClient.delete(`${this.resourcePath}/${productId}`);
  }

  /** Archive a product (POST /api/products/{id}/archive). */
  async archive(productId: string): Promise<ProductResponse> {
    return this.httpClient.post<ProductResponse>(`${this.resourcePath}/${productId}/archive`, {});
  }

  /** Unarchive a product (POST /api/products/{id}/unarchive). */
  async unarchive(productId: string): Promise<ProductResponse> {
    return this.httpClient.post<ProductResponse>(`${this.resourcePath}/${productId}/unarchive`, {});
  }

  /** List a product's variants (GET /api/products/{id}/variants). */
  async listVariants(productId: string): Promise<ListResponse<VariantResponse>> {
    return this.httpClient.get<ListResponse<VariantResponse>>(
      `${this.resourcePath}/${productId}/variants`,
    );
  }

  /** Create a variant under a product (POST /api/products/{id}/variants). */
  async createVariant(productId: string, data: CreateVariantRequest): Promise<VariantResponse> {
    return this.httpClient.post<VariantResponse>(
      `${this.resourcePath}/${productId}/variants`,
      data,
    );
  }
}

export class VariantsResource {
  private readonly resourcePath = '/api/variants';

  constructor(private httpClient: HttpClient) {}

  /** Get a variant by id (GET /api/variants/{variantId}). */
  async get(variantId: string): Promise<VariantResponse> {
    return this.httpClient.get<VariantResponse>(`${this.resourcePath}/${variantId}`);
  }

  /** Update a variant (PUT /api/variants/{variantId}). */
  async update(variantId: string, data: UpdateVariantRequest): Promise<VariantResponse> {
    return this.httpClient.put<VariantResponse>(`${this.resourcePath}/${variantId}`, data);
  }

  /** Delete a variant (DELETE /api/variants/{variantId}). */
  async delete(variantId: string): Promise<void> {
    return this.httpClient.delete(`${this.resourcePath}/${variantId}`);
  }

  /** List a variant's prices (GET /api/variants/{variantId}/prices). */
  async listPrices(variantId: string): Promise<ListResponse<PriceResponse>> {
    return this.httpClient.get<ListResponse<PriceResponse>>(
      `${this.resourcePath}/${variantId}/prices`,
    );
  }
}

export class PricesResource {
  private readonly resourcePath = '/api/prices';

  constructor(private httpClient: HttpClient) {}

  /** Create a price (POST /api/prices). */
  async create(data: CreatePriceRequest): Promise<PriceResponse> {
    return this.httpClient.post<PriceResponse>(this.resourcePath, data);
  }

  /** Get a price by id (GET /api/prices/{priceId}). */
  async get(priceId: string): Promise<PriceResponse> {
    return this.httpClient.get<PriceResponse>(`${this.resourcePath}/${priceId}`);
  }

  /** Update a price (PATCH /api/prices/{priceId}). */
  async update(priceId: string, data: CreatePriceRequest): Promise<PriceResponse> {
    return this.httpClient.patch<PriceResponse>(`${this.resourcePath}/${priceId}`, data);
  }

  /** Delete a price (DELETE /api/prices/{priceId}). */
  async delete(priceId: string): Promise<void> {
    return this.httpClient.delete(`${this.resourcePath}/${priceId}`);
  }
}
