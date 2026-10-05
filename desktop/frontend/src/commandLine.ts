import type { Action, Candidate } from './actions'

export type Completion = { start: number; typed: string; items: Candidate[] }

const split = (value: string) => value.trimStart().split(/\s+/)

export function commandNamed(actions: Action[], name: string): Action | undefined {
  const lower = name.toLowerCase()
  return actions.find((a) => a.command?.name === lower || a.command?.aliases?.includes(lower))
}

export function parseInput(value: string): { name: string; args: string[] } {
  const [name = '', ...args] = split(value.trim())
  return { name, args }
}

export function completionsFor(value: string, actions: Action[]): Completion {
  const parts = split(value)
  const endsWithSpace = /\s$/.test(value)
  if (parts.length === 1 && !endsWithSpace) {
    const typed = parts[0].toLowerCase()
    const items = actions
      .filter((a) => a.command?.name.startsWith(typed))
      .map((a) => ({ value: a.command!.name, label: a.command!.syntax, detail: a.command!.summary }))
    return { start: value.length - parts[0].length, typed, items }
  }
  const args = endsWithSpace ? [...parts.slice(1), ''] : parts.slice(1)
  const typed = args[args.length - 1]
  const items = commandNamed(actions, parts[0])?.command?.complete?.(args) ?? []
  return { start: value.length - typed.length, typed, items }
}

export function commonPrefix(values: string[]): string {
  return values.reduce((prefix, v) => {
    let i = 0
    while (i < prefix.length && i < v.length && prefix[i].toLowerCase() === v[i].toLowerCase()) i++
    return prefix.slice(0, i)
  })
}

export function ghostFor(value: string, actions: Action[]): string {
  if (!value.trim() || /\s\S/.test(value.trimStart())) return ''
  const typed = value.trim().toLowerCase()
  const exact = commandNamed(actions, typed)
  const matches = actions.filter((a) => a.command?.name.startsWith(typed))
  const action = exact ?? (matches.length === 1 ? matches[0] : undefined)
  if (!action?.command) return ''
  const { name, syntax } = action.command
  const rest = syntax.slice(name.length)
  if (/\s$/.test(value)) return rest.trimStart()
  return name.slice(typed.length) + rest
}

export function didYouMean(actions: Action[], name: string): string {
  const close = actions.filter((a) => a.command && (a.command.name.startsWith(name.slice(0, 2)) || name.startsWith(a.command.name.slice(0, 2))))
  return close.length ? ` Did you mean ${close.slice(0, 3).map((a) => a.command!.name).join(', ')}?` : ''
}
