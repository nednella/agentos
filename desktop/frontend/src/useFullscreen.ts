import { useEffect, useState } from 'react'

// Wails sends no event when the window enters or leaves full screen, but the window resizes both times.
export function useFullscreen(): boolean {
  const [fullscreen, setFullscreen] = useState(false)
  useEffect(() => {
    const check = () => window.runtime?.WindowIsFullscreen?.().then(setFullscreen)
    check()
    window.addEventListener('resize', check)
    return () => window.removeEventListener('resize', check)
  }, [])
  return fullscreen
}
