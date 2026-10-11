export function tokenCount(n: number): string {
  if (n < 1000) return String(n)
  if (n < 1_000_000) return `${(n / 1000).toFixed(n < 10_000 ? 1 : 0)}k`
  return `${(n / 1_000_000).toFixed(n < 10_000_000 ? 2 : 1)}M`
}

export function dollars(cost: number): string {
  if (cost > 0 && cost < 0.01) return '<$0.01'
  return `$${cost.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

export const priceNote = (asOf: string) => `Estimated at Anthropic API list prices of ${asOf}`
