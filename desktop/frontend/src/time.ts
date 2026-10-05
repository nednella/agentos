import { useSyncExternalStore } from 'react'

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

export function useNow(): number {
  return useSyncExternalStore(subscribe, () => now)
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
