import { useEffect, useState } from 'react'

const NAME = 'agentos'
const BLINKS = 2
const LEAVE_MS = 2450 + BLINKS * 800
const FADE_MS = 600

export type SplashStage = 'playing' | 'leaving' | 'done'

type SplashProps = {
  stage: SplashStage
  onStage: (stage: SplashStage) => void
}

export function Splash({ stage, onStage }: SplashProps) {
  const [drawn, setDrawn] = useState(false)
  const [cursorOn, setCursorOn] = useState(false)
  const [typed, setTyped] = useState(0)

  useEffect(() => {
    const timers: ReturnType<typeof setTimeout>[] = []
    const at = (ms: number, fn: () => void) => timers.push(setTimeout(fn, ms))
    at(50, () => setDrawn(true))
    at(650, () => setCursorOn(true))
    for (let i = 1; i <= NAME.length; i++) at(850 + i * 150, () => setTyped(i))
    for (let b = 0; b < BLINKS; b++) {
      at(2450 + b * 800, () => setCursorOn(false))
      at(2850 + b * 800, () => setCursorOn(true))
    }
    at(LEAVE_MS, () => onStage('leaving'))
    at(LEAVE_MS + FADE_MS, () => onStage('done'))
    return () => timers.forEach(clearTimeout)
  }, [onStage])

  return (
    <div
      className={`splash ${drawn ? 'drawn' : ''} ${stage === 'leaving' ? 'leaving' : ''}`}
      role="status"
      aria-label="Starting agentos"
    >
      <div className="splash-prompt">
        <svg viewBox="258 348 224 328" aria-hidden="true">
          <polyline points="290,380 450,512 290,644" />
        </svg>
        <span className="splash-name">{NAME.slice(0, typed)}</span>
        <span className={`splash-cursor ${cursorOn ? 'on' : ''}`} />
      </div>
    </div>
  )
}
