/**
 * Utility for handling currency amounts and formatting
 */

export class AmountFormatter {
  /**
   * Convert dollars to cents
   */
  dollarsTocents(dollars: number): number {
    return Math.round(dollars * 100)
  }

  /**
   * Convert cents to dollars
   */
  centsToDollars(cents: number): number {
    return cents / 100
  }

  /**
   * Format amount as currency string
   */
  formatCurrency(cents: number, currency = 'USD'): string {
    const amount = this.centsToDollars(cents)
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency
    }).format(amount)
  }

  /**
   * Parse currency string to cents
   */
  parseCurrency(currencyString: string): number {
    const number = parseFloat(currencyString.replace(/[^\d.-]/g, ''))
    return this.dollarsTocents(number)
  }
}
