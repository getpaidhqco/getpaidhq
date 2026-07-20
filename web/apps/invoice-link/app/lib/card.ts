export type CardBrand =
  | "visa"
  | "mastercard"
  | "amex"
  | "discover"
  | "unknown";

export function digitsOnly(value: string): string {
  return value.replace(/\D/g, "");
}

export function detectBrand(digits: string): CardBrand {
  if (/^4/.test(digits)) return "visa";
  if (/^(5[1-5]|2(2[2-9]|[3-6]|7[01]|720))/.test(digits)) return "mastercard";
  if (/^3[47]/.test(digits)) return "amex";
  if (/^(6011|65)/.test(digits)) return "discover";
  return "unknown";
}

export function cardNumberLength(brand: CardBrand): number {
  return brand === "amex" ? 15 : 16;
}

export function formatCardNumber(digits: string, brand: CardBrand): string {
  const groups =
    brand === "amex" ? [4, 6, 5] : [4, 4, 4, 4];
  const parts: string[] = [];
  let rest = digits;
  for (const size of groups) {
    if (!rest) break;
    parts.push(rest.slice(0, size));
    rest = rest.slice(size);
  }
  return parts.join(" ");
}

export function luhnValid(digits: string): boolean {
  let sum = 0;
  let double = false;
  for (let i = digits.length - 1; i >= 0; i--) {
    let digit = Number(digits[i]);
    if (double) {
      digit *= 2;
      if (digit > 9) digit -= 9;
    }
    sum += digit;
    double = !double;
  }
  return sum % 10 === 0;
}

export function formatExpiry(value: string): string {
  const digits = digitsOnly(value).slice(0, 4);
  if (digits.length <= 2) return digits;
  return `${digits.slice(0, 2)}/${digits.slice(2)}`;
}

export function expiryValid(value: string): boolean {
  const match = /^(\d{2})\/(\d{2})$/.exec(value);
  if (!match) return false;
  const month = Number(match[1]);
  if (month < 1 || month > 12) return false;
  const year = 2000 + Number(match[2]);
  const now = new Date();
  return (
    year > now.getFullYear() ||
    (year === now.getFullYear() && month >= now.getMonth() + 1)
  );
}

export function cvcLength(brand: CardBrand): number {
  return brand === "amex" ? 4 : 3;
}
