import { useSyncExternalStore } from 'react'

const SECOND = 1000
const MINUTE = 60 * SECOND
const DAY = 24 * 60 * MINUTE
const TWO_DAYS = 2 * DAY

const listeners = new Set<() => void>()
let now = Date.now()

setInterval(() => {
  now = Date.now()
  listeners.forEach((listener) => listener())
}, 1000)

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

// Renders again only when `read` gives a new value, so text that reads the clock renders when the text changes.
export function useClock<T extends string | number>(read: (now: number) => T): T {
  return useSyncExternalStore(subscribe, () => read(now))
}

export function duration(ms: number): string {
  const seconds = Math.max(0, Math.floor(ms / 1000))
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours >= 48) return `${Math.floor(hours / 24)}d`
  const rest = minutes % 60
  return rest ? `${Math.floor(minutes / 60)}h ${rest}m` : `${Math.floor(minutes / 60)}h`
}

export function ago(at: number, current: number): string {
  if (!at) return '–'
  if (current - at < 5000) return 'now'
  return `${duration(current - at)} ago`
}

export function until(at: number, current: number): string {
  return at <= current ? 'now' : `in ${duration(at - current)}`
}

// The clock time `duration` shows for a stretch that started at `from`, so a row renders only when that text moves.
export function shownNow(current: number, from: number): number {
  const elapsed = Math.max(0, current - from)
  const step = elapsed < MINUTE ? SECOND : elapsed < TWO_DAYS ? MINUTE : DAY
  return from + elapsed - (elapsed % step)
}
