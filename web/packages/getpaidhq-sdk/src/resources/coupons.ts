import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import {
  CouponCodeResponse,
  CouponResponse,
  CreateCouponCodeInput,
  CreateCouponInput,
  DiscountResponse,
  ListResponse,
  PaginationParams,
  UpdateCouponCodeInput,
  UpdateCouponInput,
} from '../types';

export class CouponsResource {
  private readonly resourcePath = '/api/coupons';

  constructor(private httpClient: HttpClient) {}

  /** List coupons (GET /api/coupons). */
  async list(params?: PaginationParams): Promise<ListResponse<CouponResponse>> {
    return this.httpClient.get<ListResponse<CouponResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Create a coupon (POST /api/coupons). */
  async create(data: CreateCouponInput): Promise<CouponResponse> {
    return this.httpClient.post<CouponResponse>(this.resourcePath, data);
  }

  /** Get a coupon by id (GET /api/coupons/{id}). */
  async get(couponId: string): Promise<CouponResponse> {
    return this.httpClient.get<CouponResponse>(`${this.resourcePath}/${couponId}`);
  }

  /** Update a coupon (PATCH /api/coupons/{id}). */
  async update(couponId: string, data: UpdateCouponInput): Promise<CouponResponse> {
    return this.httpClient.patch<CouponResponse>(`${this.resourcePath}/${couponId}`, data);
  }

  /** Delete a coupon (DELETE /api/coupons/{id}). */
  async delete(couponId: string): Promise<CouponResponse> {
    return this.httpClient.delete<CouponResponse>(`${this.resourcePath}/${couponId}`);
  }

  /** List a coupon's codes (GET /api/coupons/{id}/codes). */
  async listCodes(couponId: string): Promise<CouponCodeResponse[]> {
    return this.httpClient.get<CouponCodeResponse[]>(`${this.resourcePath}/${couponId}/codes`);
  }

  /** Create a coupon code (POST /api/coupons/{id}/codes). */
  async createCode(couponId: string, data: CreateCouponCodeInput): Promise<CouponCodeResponse> {
    return this.httpClient.post<CouponCodeResponse>(
      `${this.resourcePath}/${couponId}/codes`,
      data,
    );
  }

  /** Update a coupon code (PATCH /api/coupon-codes/{id}). */
  async updateCode(couponCodeId: string, data: UpdateCouponCodeInput): Promise<CouponCodeResponse> {
    return this.httpClient.patch<CouponCodeResponse>(`/api/coupon-codes/${couponCodeId}`, data);
  }
}

export class DiscountsResource {
  private readonly resourcePath = '/api/discounts';

  constructor(private httpClient: HttpClient) {}

  /** Get a discount by id (GET /api/discounts/{id}). */
  async get(discountId: string): Promise<DiscountResponse> {
    return this.httpClient.get<DiscountResponse>(`${this.resourcePath}/${discountId}`);
  }
}
