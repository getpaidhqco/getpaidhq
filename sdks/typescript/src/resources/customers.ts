import type { BaseClient } from '@/client/base-client'
import type { PaginatedResponse } from '@/types'

export interface Customer {
  id: string
  email: string
  name?: string
  phone?: string
  metadata?: Record<string, string>
  createdAt: string
  updatedAt: string
}

export interface CreateCustomerRequest {
  email: string
  name?: string
  phone?: string
  metadata?: Record<string, string>
}

export interface UpdateCustomerRequest {
  email?: string
  name?: string
  phone?: string
  metadata?: Record<string, string>
}

export class CustomersResource {
  constructor(private client: BaseClient) {}

  /**
   * Create a new customer
   */
  async create(data: CreateCustomerRequest): Promise<Customer> {
    return this.client.post<Customer>('/api/customers', data)
  }

  /**
   * Retrieve a customer by ID
   */
  async find(id: string): Promise<Customer> {
    return this.client.get<Customer>(`/api/customers/${id}`)
  }

  /**
   * List customers with pagination
   */
  async list(params?: {
    limit?: number
    offset?: number
    email?: string
  }): Promise<PaginatedResponse<Customer>> {
    const query = new URLSearchParams()
    if (params?.limit) query.set('limit', params.limit.toString())
    if (params?.offset) query.set('offset', params.offset.toString())
    if (params?.email) query.set('email', params.email)

    const queryString = query.toString()
    const url = `/api/customers${queryString ? `?${queryString}` : ''}`

    return this.client.get<PaginatedResponse<Customer>>(url)
  }

  /**
   * Update a customer
   */
  async update(id: string, data: UpdateCustomerRequest): Promise<Customer> {
    return this.client.put<Customer>(`/api/customers/${id}`, data)
  }

  /**
   * Delete a customer
   */
  async delete(id: string): Promise<void> {
    return this.client.delete(`/api/customers/${id}`)
  }
}