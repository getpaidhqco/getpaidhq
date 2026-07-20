import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import axiosRetry from 'axios-retry';
import { AuthManager } from '../auth/auth';
import { handleApiError } from '../errors/errors';

export interface HttpClientConfig {
  baseURL: string;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
  userAgent?: string;
}

export class HttpClient {
  private client: AxiosInstance;
  private authManager: AuthManager;

  constructor(config: HttpClientConfig, authManager: AuthManager) {
    this.authManager = authManager;

    this.client = axios.create({
      baseURL: config.baseURL,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Configure retry logic
    axiosRetry(this.client, {
      retries: config.retries || 3,
      retryDelay: (retryCount) => {
        const delay = config.retryDelay || 1000;
        return delay * Math.pow(2, retryCount - 1); // Exponential backoff
      },
      retryCondition: (error) => {
        // Retry on network errors
        return axiosRetry.isNetworkError(error);
      },
    });

    // Request interceptor for authentication
    this.client.interceptors.request.use(
      async (config) => {
        const authConfig = await this.authManager.applyAuth(config);
        return authConfig as any;
      },
      (error) => Promise.reject(error),
    );

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error) => handleApiError(error),
    );
  }

  async get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.get<T>(url, config);
    return response.data;
  }

  async post<T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.post<T>(url, data, config);
    return response.data;
  }

  async put<T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.put<T>(url, data, config);
    return response.data;
  }

  async patch<T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.patch<T>(url, data, config);
    return response.data;
  }

  async delete<T = void>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.delete<T>(url, config);
    return response.data;
  }

  getAxiosInstance(): AxiosInstance {
    return this.client;
  }
}
