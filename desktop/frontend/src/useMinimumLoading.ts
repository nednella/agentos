import { useEffect, useRef, useState } from 'react'

// Long enough for the loader to type its three dots, so a fast answer does not flash it.
const MINIMUM_MS = 1000

export function useMinimumLoading(loading: boolean): boolean {
  const [holding, setHolding] = useState(loading)
  const startedAt = useRef<number | null>(loading ? Date.now() : null)

  useEffect(() => {
    if (loading) {
      startedAt.current ??= Date.now()
      setHolding(true)
      return
    }
    if (startedAt.current === null) return
    const timer = setTimeout(() => setHolding(false), MINIMUM_MS - (Date.now() - startedAt.current))
    startedAt.current = null
    return () => clearTimeout(timer)
  }, [loading])

  return loading || holding
}
