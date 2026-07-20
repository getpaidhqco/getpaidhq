/**
 * Serialize a params object into a query string (prefixed with `?` when non-empty).
 * Undefined/null values are skipped; arrays are expanded as repeated `key[]=` pairs.
 */
export function buildQueryString(params?: Record<string, any>): string {
  if (!params) return '';

  const query = Object.entries(params)
    .filter(([, value]) => value !== undefined && value !== null)
    .map(([key, value]) => {
      if (Array.isArray(value)) {
        return value.map((v) => `${key}[]=${encodeURIComponent(String(v))}`).join('&');
      }
      return `${key}=${encodeURIComponent(String(value))}`;
    })
    .join('&');

  return query ? `?${query}` : '';
}
