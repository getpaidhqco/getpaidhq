import type { BaseClient } from '@/client/base-client'
import type { PaginatedResponse } from '@/types'

export interface Payment {
  id: string
  customerId: string
  invoiceId?: string
  amount: number
  currency: string
  status: 'pending' | 'succeeded' | 'failed' | 'refunded' | 'partially_refunded'
  paymentMethodId: string
  paymentMethodType: 'card' | 'bank_account' | 'wallet'
  paymentMethodDetails?: Record<string, any>
  refundedAmount?: number
  description?: string
  metadata?: Record<string, string>
  createdAt: string
  updatedAt: string
}

export interface CreatePaymentRequest {
  customerId: string
  amount: number
  currency: string
  paymentMethodId: string
  description?: string
  invoiceId?: string
  metadata?: Record<string, string>
}

export interface RefundPaymentRequest {
  amount?: number
  reason?: 'requested_by_customer' | 'duplicate' | 'fraudulent'
  metadata?: Record<string, string>
}

export class PaymentsResource {
  constructor(private client: BaseClient) {}

  /**
   * Create a new payment
   */
  async create(data: CreatePaymentRequest): Promise<Payment> {
    return this.client.post<Payment>('/api/payments', data)
  }

  /**
   * Retrieve a payment by ID
   */
  async find(id: string): Promise<Payment> {
    return this.client.get<Payment>(`/api/payments/${id}`)
  }

  /**
   * List payments with pagination
   */
  async list(params?: {
    limit?: number
    offset?: number
    customerId?: string
    invoiceId?: string
    status?: 'pending' | 'succeeded' | 'failed' | 'refunded' | 'partially_refunded'
  }): Promise<PaginatedResponse<Payment>> {
    const query = new URLSearchParams()
    if (params?.limit) query.set('limit', params.limit.toString())
    if (params?.offset) query.set('offset', params.offset.toString())
    if (params?.customerId) query.set('customer_id', params.customerId)
    if (params?.invoiceId) query.set('invoice_id', params.invoiceId)
    if (params?.status) query.set('status', params.status)

    const queryString = query.toString()
    const url = `/api/payments${queryString ? `?${queryString}` : ''}`

    return this.client.get<PaginatedResponse<Payment>>(url)
  }

  /**
   * Refund a payment (full or partial)
   */
  async refund(id: string, data?: RefundPaymentRequest): Promise<Payment> {
    return this.client.post<Payment>(`/api/payments/${id}/refund`, data || {})
  }

  /**
   * Capture a previously authorized payment
   */
  async capture(id: string, options?: {
    amount?: number
  }): Promise<Payment> {
    return this.client.post<Payment>(`/api/payments/${id}/capture`, options || {})
  }

  /**
   * Cancel a pending payment
   */
  async cancel(id: string): Promise<Payment> {
    return this.client.post<Payment>(`/api/payments/${id}/cancel`, {})
  }
}