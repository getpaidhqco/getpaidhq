/**
 * Utility for verifying webhook signatures
 */

import { createHmac } from 'crypto'

export class WebhookVerifier {
  /**
   * Verify webhook signature
   */
  verify(payload: string, signature: string, secret: string): boolean {
    const expectedSignature = this.generateSignature(payload, secret)
    return this.secureCompare(signature, expectedSignature)
  }

  /**
   * Generate HMAC signature for payload
   */
  private generateSignature(payload: string, secret: string): string {
    return createHmac('sha256', secret)
      .update(payload, 'utf8')
      .digest('hex')
  }

  /**
   * Secure string comparison to prevent timing attacks
   */
  private secureCompare(a: string, b: string): boolean {
    if (a.length !== b.length) return false
    
    let result = 0
    for (let i = 0; i < a.length; i++) {
      result |= a.charCodeAt(i) ^ b.charCodeAt(i)
    }
    
    return result === 0
  }
}
