import { useCallback, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'

export type ListNav = { cursor: number; cursorKey: string | null; setCursorKey(key: string): void; handle(e: KeyboardEvent): boolean }

type ListNavHandlers = {
  onEnter(index: number): void
  onLeft?(index: number): boolean
  onRight?(index: number): boolean
  onSpace?(index: number): boolean
}

const UP = new Set(['ArrowUp', 'w'])
const DOWN = new Set(['ArrowDown', 's'])
const LEFT = new Set(['ArrowLeft', 'a'])
const RIGHT = new Set(['ArrowRight', 'd'])

export function isTyping(target: EventTarget | null): boolean {
  return target instanceof HTMLElement && (target.matches('input, textarea, select') || target.isContentEditable)
}

// Inside xterm the app's own shortcuts must still work; every other text field keeps its keys.
export function isEditingText(target: EventTarget | null): boolean {
  return isTyping(target) && !(target as HTMLElement).closest('.xterm')
}

// The cursor follows the row it is on by key, so re-sorting the list does not move it.
export function useListNav(keys: string[], handlers: ListNavHandlers): ListNav {
  const [cursorKey, setCursorKey] = useState<string | null>(null)
  const lastIndex = useRef(0)
  const at = cursorKey === null ? -1 : keys.indexOf(cursorKey)
  const cursor = at >= 0 ? at : Math.max(0, Math.min(lastIndex.current, keys.length - 1))
  lastIndex.current = cursor

  const handle = useCallback(
    (e: KeyboardEvent): boolean => {
      if (e.metaKey || e.ctrlKey || e.altKey || isTyping(e.target)) return false
      if (UP.has(e.key)) setCursorKey(keys[Math.max(cursor - 1, 0)] ?? null)
      else if (DOWN.has(e.key)) setCursorKey(keys[Math.max(0, Math.min(cursor + 1, keys.length - 1))] ?? null)
      else if (e.key === 'Enter') {
        if (e.target !== e.currentTarget) return false
        handlers.onEnter(cursor)
      }
      else if (e.key === ' ') return handlers.onSpace?.(cursor) ?? false
      else if (LEFT.has(e.key)) return handlers.onLeft?.(cursor) ?? false
      else if (RIGHT.has(e.key)) return handlers.onRight?.(cursor) ?? false
      else return false
      return true
    },
    [keys, cursor, handlers],
  )

  return { cursor, cursorKey: keys[cursor] ?? null, setCursorKey, handle }
}
