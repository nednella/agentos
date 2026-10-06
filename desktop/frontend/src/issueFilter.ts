import type { Issue, IssueType } from './types'

export type Field = 'author' | 'assignee' | 'label' | 'type' | 'section' | 'is'

export type Token = { field: Field; value: string; negate: boolean }
export type Word = { text: string; negate: boolean }
export type ParsedQuery = { words: Word[]; tokens: Token[] }

export const TYPES: IssueType[] = ['bug', 'feature', 'refactor', 'chore']

// A section's name as it is typed after section:, which has no room for a space.
export const sectionSlug = (name: string) => name.toLowerCase().replace(/\s+/g, '-')

const TOKEN = /^(author|assignee|label|type|section|is):(.*)$/i

export function parseQuery(query: string): ParsedQuery {
  const parsed: ParsedQuery = { words: [], tokens: [] }
  for (const part of query.split(/\s+/).filter(Boolean)) {
    const negate = part.startsWith('-') && part.length > 1
    const body = negate ? part.slice(1) : part
    const match = body.startsWith('@') ? ['', 'author', body.slice(1)] : TOKEN.exec(body)
    if (!match) {
      parsed.words.push({ text: body.replace(/^#/, '').toLowerCase(), negate })
      continue
    }
    const value = match[2].toLowerCase()
    if (value) parsed.tokens.push({ field: match[1].toLowerCase() as Field, value, negate })
  }
  return parsed
}

export function isFiltering(query: string): boolean {
  const { words, tokens } = parseQuery(query)
  return words.length + tokens.length > 0
}

function tokenMatches(issue: Issue, { field, value }: Token): boolean {
  if (field === 'author') return issue.author.toLowerCase().startsWith(value)
  if (field === 'assignee') {
    if (value === 'none') return issue.assignees.length === 0
    return issue.assignees.some((a) => a.toLowerCase().startsWith(value))
  }
  if (field === 'label') return issue.labels.some((l) => l.toLowerCase().startsWith(value))
  if (field === 'type') return issue.type === value
  if (field === 'section') return sectionSlug(issue.section) === value
  return value === 'running' && issue.sessionId !== ''
}

export function matchIssue(issue: Issue, parsed: ParsedQuery): boolean {
  const haystack = `${issue.number} ${issue.title.toLowerCase()}`
  return (
    parsed.words.every((w) => haystack.includes(w.text) !== w.negate) &&
    parsed.tokens.every((t) => tokenMatches(issue, t) !== t.negate)
  )
}

export function filterIssues(issues: Issue[], query: string): Issue[] {
  const parsed = parseQuery(query)
  return issues.filter((issue) => matchIssue(issue, parsed))
}

export type Facet = { value: string; count: number }

export function facet(issues: Issue[], pick: (issue: Issue) => string[]): Facet[] {
  const counts = new Map<string, number>()
  for (const issue of issues) for (const value of new Set(pick(issue))) counts.set(value, (counts.get(value) ?? 0) + 1)
  return [...counts].map(([value, count]) => ({ value, count })).sort((a, b) => b.count - a.count || a.value.localeCompare(b.value))
}

export type Completion = { start: number; prefix: string; values: Facet[] }

function valuesFor(field: string, issues: Issue[]): Facet[] {
  if (field === 'author') return facet(issues, (i) => [i.author])
  if (field === 'assignee') return [{ value: 'none', count: issues.filter((i) => i.assignees.length === 0).length }, ...facet(issues, (i) => i.assignees)]
  if (field === 'label') return facet(issues, (i) => i.labels)
  if (field === 'type') return TYPES.map((value) => ({ value, count: issues.filter((i) => i.type === value).length }))
  if (field === 'section') return facet(issues, (i) => [sectionSlug(i.section)])
  return [{ value: 'running', count: issues.filter((i) => i.sessionId).length }]
}

export function completionAt(query: string, issues: Issue[]): Completion | null {
  if (/\s$/.test(query)) return null
  const start = query.search(/\S+$/)
  if (start < 0) return null
  const raw = query.slice(start)
  const negate = raw.startsWith('-') ? '-' : ''
  const body = raw.slice(negate.length)
  const match = body.startsWith('@') ? ['', 'author', body.slice(1)] : TOKEN.exec(body)
  if (!match) return null
  const field = match[1].toLowerCase()
  const typed = match[2].toLowerCase()
  const prefix = `${negate}${body.startsWith('@') ? '@' : `${field}:`}`
  const values = valuesFor(field, issues).filter((v) => v.value.toLowerCase().startsWith(typed) && v.value.toLowerCase() !== typed)
  return values.length ? { start, prefix, values } : null
}

export function applyCompletion(query: string, completion: Completion, value: string): string {
  return `${query.slice(0, completion.start)}${completion.prefix}${value} `
}

export function withToken(query: string, field: 'author' | 'type', value: string | null): string {
  const kept = query
    .split(/\s+/)
    .filter(Boolean)
    .filter((part) => !(field === 'author' ? /^(@|author:)/i.test(part) : /^type:/i.test(part)))
  if (value) kept.push(field === 'author' ? `@${value}` : `type:${value}`)
  return kept.join(' ') + (kept.length ? ' ' : '')
}

export function activeValue(query: string, field: 'author' | 'type'): string | null {
  const token = parseQuery(query).tokens.find((t) => t.field === field && !t.negate)
  return token?.value ?? null
}
