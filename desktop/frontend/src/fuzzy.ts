export function fuzzyScore(query: string, text: string): number {
  if (!query) return 1
  const q = query.toLowerCase()
  const t = text.toLowerCase()
  const at = t.indexOf(q)
  if (at >= 0) return 100 - at
  let score = 0
  let from = 0
  for (const ch of q) {
    const found = t.indexOf(ch, from)
    if (found < 0) return 0
    score += found === from ? 3 : 1
    from = found + 1
  }
  return score
}
