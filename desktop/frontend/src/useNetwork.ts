import { useCallback, useEffect, useState } from 'react'
import { on } from './api'

const CHECK_EVERY_S = 10
const CHECK_TIMEOUT_MS = 5000
// navigator.onLine only says the Mac has a link; a request shows the link reaches GitHub, which the queue and PRs need.
const CHECK_URL = 'https://api.github.com/'

async function reachable(): Promise<boolean> {
  if (!navigator.onLine) return false
  try {
    await fetch(CHECK_URL, { method: 'HEAD', mode: 'no-cors', cache: 'no-store', signal: AbortSignal.timeout(CHECK_TIMEOUT_MS) })
    return true
  } catch {
    return false
  }
}

export function useNetwork() {
  const [online, setOnline] = useState(navigator.onLine)
  const [checking, setChecking] = useState(false)
  const [nextCheckIn, setNextCheckIn] = useState(CHECK_EVERY_S)

  const check = useCallback(async () => {
    setChecking(true)
    const ok = await reachable()
    setChecking(false)
    setOnline(ok)
    setNextCheckIn(CHECK_EVERY_S)
  }, [])

  useEffect(() => {
    const goOffline = () => {
      setOnline(false)
      setNextCheckIn(CHECK_EVERY_S)
    }
    window.addEventListener('offline', goOffline)
    window.addEventListener('online', check)
    return () => {
      window.removeEventListener('offline', goOffline)
      window.removeEventListener('online', check)
    }
  }, [check])

  // The app sends network when a GitHub or git call fails: Wi-Fi with no internet sends no offline event.
  useEffect(() => {
    if (!online) return
    return on('network', () => void check())
  }, [online, check])

  useEffect(() => {
    if (online || checking) return
    if (nextCheckIn === 0) {
      void check()
      return
    }
    const timer = setTimeout(() => setNextCheckIn(nextCheckIn - 1), 1000)
    return () => clearTimeout(timer)
  }, [online, checking, nextCheckIn, check])

  return { online, checking, nextCheckIn, check }
}
