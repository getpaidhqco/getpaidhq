import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import axios from 'axios'

export interface ClientConfig {
  apiKey: string
  baseURL?: string
  timeout?: number
  retries?: number
  debug?: boolean
}

export interface RequestOptions extends Omit<AxiosRequestConfig, 'url' | 'method'> {
  retries?: number
}

export class BaseClient {
  private readonly axios: AxiosInstance
  private readonly config: Required<ClientConfig>

  constructor(config: ClientConfig) {
    this.config = {
      baseURL: 'https://api.getpaidhq.com',
      timeout: 30000,
      retries: 3,
      debug: false,
      ...config
    }

    this.axios = axios.create({
      baseURL: this.config.baseURL,
      timeout: this.config.timeout,
      headers: {
        'Authorization': `Bearer ${this.config.apiKey}`,
        'Content-Type': 'application/json',
        'User-Agent': `@getpaidhq/sdk/${process.env.npm_package_version || '0.1.0'}`
      }
    })

    this.setupInterceptors()
  }

  private setupInterceptors() {
    // Request interceptor for debugging
    this.axios.interceptors.request.use(
      (config) => {
        if (this.config.debug) {
          console.log('→', config.method?.toUpperCase(), config.url, config.data)
        }
        return config
      },
      (error) => Promise.reject(error)
    )

    // Response interceptor for debugging and error handling
    this.axios.interceptors.response.use(
      (response) => {
        if (this.config.debug) {
          console.log('←', response.status, response.config.url)
        }
        return response
      },
      (error) => {
        if (this.config.debug) {
          console.error('✗', error.response?.status, error.config?.url, error.response?.data)
        }
        return Promise.reject(this.enhanceError(error))
      }
    )
  }

  private enhanceError(error: any): GetPaidHQError {
    if (error.response) {
      return new GetPaidHQError(
        error.response.data?.message || error.message,
        error.response.status,
        error.response.data?.code,
        error.response.data
      )
    }
    return new GetPaidHQError(error.message)
  }

  protected async request<T = any>(
    method: string,
    url: string,
    options: RequestOptions = {}
  ): Promise<T> {
    const { retries = this.config.retries, ...axiosOptions } = options

    for (let attempt = 0; attempt <= retries; attempt++) {
      try {
        const response: AxiosResponse<T> = await this.axios.request({
          method,
          url,
          ...axiosOptions
        })
        return response.data
      } catch (error) {
        if (attempt === retries || !this.shouldRetry(error)) {
          throw error
        }
        await this.delay(Math.pow(2, attempt) * 1000) // Exponential backoff
      }
    }

    throw new Error('Max retries exceeded')
  }

  private shouldRetry(error: any): boolean {
    if (!error.response) return true // Network errors
    const status = error.response.status
    return status >= 500 || status === 429 // Server errors and rate limits
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
  }

  // HTTP method helpers
  protected get<T>(url: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('GET', url, options)
  }

  protected post<T>(url: string, data?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>('POST', url, { ...options, data })
  }

  protected put<T>(url: string, data?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>('PUT', url, { ...options, data })
  }

  protected patch<T>(url: string, data?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>('PATCH', url, { ...options, data })
  }

  protected delete<T>(url: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('DELETE', url, options)
  }
}

export class GetPaidHQError extends Error {
  constructor(
    message: string,
    public readonly status?: number,
    public readonly code?: string,
    public readonly details?: any
  ) {
    super(message)
    this.name = 'GetPaidHQError'
  }
}