export const formatCurrency = (currency: string, amount: number = 0) => {
  return new Intl.NumberFormat(getBrowserLocale() || 'en', {
    style: 'currency', currency
  }).format(amount / 100);
};
export const getBrowserLocale = () => {
  if (typeof navigator === 'undefined') return 'en';
  return navigator.language || (navigator as any).userLanguage;
};


export const centsToCurrency = (cents: number): number => {
  return cents / 100;
};

export const currencyToCents = (amount: number): number => {
  return Math.round(amount * 100);
};
