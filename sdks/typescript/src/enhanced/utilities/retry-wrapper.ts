/**
 * Utility for adding retry logic to API calls
 */

export interface RetryOptions {
  maxRetries: number
  debug: boolean
  baseDelay?: number
  maxDelay?: number
}

export class RetryWrapper {
  constructor(private options: RetryOptions) {}

  /**
   * Wrap a function with retry logic
   */
  wrap<T extends (...args: any[]) => Promise<any>>(fn: T): T {
    return (async (...args: any[]) => {
      let lastError: Error
      
      for (let attempt = 0; attempt <= this.options.maxRetries; attempt++) {
        try {
          return await fn(...args)
        } catch (error) {
          lastError = error as Error
          
          if (attempt === this.options.maxRetries) {
            throw lastError
          }

          if (!this.shouldRetry(error)) {
            throw lastError
          }

          const delay = this.calculateDelay(attempt)
          
          if (this.options.debug) {
            console.log(`Attempt ${attempt + 1} failed, retrying in ${delay}ms:`, error)
          }
          
          await this.sleep(delay)
        }
      }
      
      throw lastError!
    }) as T
  }

  private shouldRetry(error: any): boolean {
    // Don't retry client errors (4xx)
    if (error.response?.status >= 400 && error.response?.status < 500) {
      return false
    }
    
    // Retry server errors (5xx) and network errors
    return true
  }

  private calculateDelay(attempt: number): number {
    const baseDelay = this.options.baseDelay || 1000
    const maxDelay = this.options.maxDelay || 30000
    
    // Exponential backoff with jitter
    const exponentialDelay = baseDelay * Math.pow(2, attempt)
    const jitter = Math.random() * 0.1 * exponentialDelay
    
    return Math.min(exponentialDelay + jitter, maxDelay)
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
  }
}
