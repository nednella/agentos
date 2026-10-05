import { useCallback, useEffect, useState } from 'react'
import type { KeyboardEvent } from 'react'

export type ListNav = { cursor: number; setCursor(index: number): void; handle(e: KeyboardEvent): boolean }

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

export function isEditingText(target: EventTarget | null): boolean {
  return isTyping(target) && (target as HTMLInputElement | HTMLTextAreaElement).value !== ''
}

export function useListNav(count: number, handlers: ListNavHandlers): ListNav {
  const [cursor, setCursor] = useState(0)

  useEffect(() => {
    setCursor((c) => Math.min(c, Math.max(count - 1, 0)))
  }, [count])

  const handle = useCallback(
    (e: KeyboardEvent): boolean => {
      if (e.metaKey || e.ctrlKey || e.altKey || isTyping(e.target)) return false
      if (UP.has(e.key)) setCursor((c) => Math.max(c - 1, 0))
      else if (DOWN.has(e.key)) setCursor((c) => Math.min(c + 1, count - 1))
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
    [count, cursor, handlers],
  )

  return { cursor, setCursor, handle }
}
