import type { BaseClient } from '@/client/base-client'
import type { PaginatedResponse } from '@/types'

export interface Invoice {
  id: string
  customerId: string
  subscriptionId?: string
  status: 'draft' | 'open' | 'paid' | 'void' | 'uncollectible'
  currency: string
  amount: number
  amountPaid: number
  amountRemaining: number
  description?: string
  dueDate?: string
  invoiceDate: string
  invoiceNumber: string
  metadata?: Record<string, string>
  lineItems?: InvoiceLineItem[]
  createdAt: string
  updatedAt: string
}

export interface InvoiceLineItem {
  id: string
  invoiceId: string
  description: string
  quantity: number
  unitAmount: number
  amount: number
  currency: string
  taxable: boolean
  taxRate?: number
  taxAmount?: number
  metadata?: Record<string, string>
}

export interface CreateInvoiceRequest {
  customerId: string
  currency: string
  description?: string
  dueDate?: string
  lineItems: {
    description: string
    quantity: number
    unitAmount: number
    taxable?: boolean
    taxRate?: number
    metadata?: Record<string, string>
  }[]
  metadata?: Record<string, string>
}

export class InvoicesResource {
  constructor(private client: BaseClient) {}

  /**
   * Create a new invoice
   */
  async create(data: CreateInvoiceRequest): Promise<Invoice> {
    return this.client.post<Invoice>('/api/invoices', data)
  }

  /**
   * Retrieve an invoice by ID
   */
  async find(id: string): Promise<Invoice> {
    return this.client.get<Invoice>(`/api/invoices/${id}`)
  }

  /**
   * List invoices with pagination
   */
  async list(params?: {
    limit?: number
    offset?: number
    customerId?: string
    status?: 'draft' | 'open' | 'paid' | 'void' | 'uncollectible'
  }): Promise<PaginatedResponse<Invoice>> {
    const query = new URLSearchParams()
    if (params?.limit) query.set('limit', params.limit.toString())
    if (params?.offset) query.set('offset', params.offset.toString())
    if (params?.customerId) query.set('customer_id', params.customerId)
    if (params?.status) query.set('status', params.status)

    const queryString = query.toString()
    const url = `/api/invoices${queryString ? `?${queryString}` : ''}`

    return this.client.get<PaginatedResponse<Invoice>>(url)
  }

  /**
   * Finalize a draft invoice
   */
  async finalize(id: string): Promise<Invoice> {
    return this.client.post<Invoice>(`/api/invoices/${id}/finalize`, {})
  }

  /**
   * Pay an invoice
   */
  async pay(id: string, options?: {
    paymentMethodId?: string
  }): Promise<Invoice> {
    return this.client.post<Invoice>(`/api/invoices/${id}/pay`, options)
  }

  /**
   * Void an invoice
   */
  async void(id: string): Promise<Invoice> {
    return this.client.post<Invoice>(`/api/invoices/${id}/void`, {})
  }

  /**
   * Send an invoice by email
   */
  async send(id: string): Promise<{ success: boolean }> {
    return this.client.post<{ success: boolean }>(`/api/invoices/${id}/send`, {})
  }

  /**
   * Get a PDF download URL for an invoice
   */
  async getPdfUrl(id: string): Promise<{ url: string }> {
    return this.client.get<{ url: string }>(`/api/invoices/${id}/pdf`)
  }
}