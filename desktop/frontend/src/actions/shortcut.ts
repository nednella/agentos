import type { Shortcut } from './types'

const KEY_LABELS: Record<string, string> = { arrowleft: '←', arrowright: '→', arrowup: '↑', arrowdown: '↓' }

export function formatShortcut(shortcut: Shortcut): string {
  const key = shortcut.label ?? KEY_LABELS[shortcut.key] ?? shortcut.key.toUpperCase()
  return `⌘${shortcut.alt ? '⌥' : ''}${shortcut.shift ? '⇧' : ''}${key}`
}

export function matchShortcut(e: KeyboardEvent, shortcut: Shortcut): boolean {
  if (!e.metaKey || e.ctrlKey || e.altKey !== Boolean(shortcut.alt)) return false
  if (!shortcut.anyShift && e.shiftKey !== Boolean(shortcut.shift)) return false
  if (shortcut.code) return e.code === shortcut.code
  return [shortcut.key, ...(shortcut.aliases ?? [])].includes(e.key.toLowerCase())
}

export const LIST_KEYS: { keys: string; summary: string }[] = [
  { keys: 'W / S or ↑ / ↓', summary: 'Move the row cursor in a focused list' },
  { keys: 'Enter', summary: 'Act on the row: queue opens the issue, sessions opens, notes edits' },
  { keys: 'A / D or ← / →', summary: 'Switch Queue and Notes, or fold a lane' },
  { keys: 'Space', summary: 'Fold or unfold the lane under the cursor' },
  { keys: 'P / E in notes', summary: 'Pin or archive the note under the cursor' },
  { keys: 'Esc', summary: 'Close a panel or overlay. Inside the terminal it goes to the agent' },
]
