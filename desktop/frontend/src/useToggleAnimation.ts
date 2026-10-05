import { useEffect, useState } from 'react'

const SETTLE_MS = 220

export function useToggleAnimation(open: boolean): boolean {
  const [settled, setSettled] = useState(open)

  useEffect(() => {
    if (settled === open) return
    const timer = setTimeout(() => setSettled(open), SETTLE_MS)
    return () => clearTimeout(timer)
  }, [open, settled])

  return settled !== open
}
