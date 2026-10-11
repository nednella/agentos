import { useEffect, useRef, useState, type CSSProperties } from 'react'

const NAME = 'agentos'
const PHASES = ['drawn', 'selected', 'cleared', 'undrawn', 'closing']
const LEAVE_MS = 4150
const FADE_MS = 600

export type SplashStage = 'playing' | 'leaving' | 'done'

type SplashProps = {
  stage: SplashStage
  onStage: (stage: SplashStage) => void
}

export function Splash({ stage, onStage }: SplashProps) {
  const [phase, setPhase] = useState(0)
  const [cursorOn, setCursorOn] = useState(false)
  const [typed, setTyped] = useState(0)
  const [glide, setGlide] = useState(0)
  const nameRef = useRef<HTMLSpanElement>(null)

  useEffect(() => {
    const timers: ReturnType<typeof setTimeout>[] = []
    const at = (ms: number, fn: () => void) => timers.push(setTimeout(fn, ms))
    at(50, () => setPhase(1))
    at(650, () => setCursorOn(true))
    for (let i = 1; i <= NAME.length; i++) at(850 + i * 130, () => setTyped(i))
    at(2150, () => setCursorOn(false))
    at(2550, () => setCursorOn(true))
    at(2850, () => setPhase(2))
    at(3200, () => {
      setGlide(nameRef.current?.getBoundingClientRect().width ?? 0)
      setTyped(0)
      setPhase(3)
    })
    at(3400, () => setPhase(4))
    at(3900, () => setPhase(5))
    at(LEAVE_MS, () => onStage('leaving'))
    at(LEAVE_MS + FADE_MS, () => onStage('done'))
    return () => timers.forEach(clearTimeout)
  }, [onStage])

  return (
    <div
      className={`splash ${PHASES.slice(0, phase).join(' ')} ${stage === 'leaving' ? 'leaving' : ''}`}
      role="status"
      aria-label="Starting agentos"
    >
      <div className="splash-prompt">
        <svg viewBox="258 348 224 328" aria-hidden="true">
          <polyline points="290,380 450,512 290,644" />
        </svg>
        <span className="splash-name" ref={nameRef}>
          <span>{NAME.slice(0, typed)}</span>
        </span>
        <span
          className={`splash-cursor ${cursorOn ? 'on' : ''}`}
          style={{ '--glide': `${glide}px` } as CSSProperties}
        />
      </div>
    </div>
  )
}
