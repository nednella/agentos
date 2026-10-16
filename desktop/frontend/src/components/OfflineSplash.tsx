type OfflineSplashProps = {
  checking: boolean
  nextCheckIn: number
  onRetry: () => void
}

export function OfflineSplash({ checking, nextCheckIn, onRetry }: OfflineSplashProps) {
  return (
    <div className="splash offline" role="alertdialog" aria-modal="true" aria-label="Network unavailable">
      <div className="flex flex-col items-center gap-6 px-4 text-center">
        <div className="offline-prompt" aria-hidden="true">
          <svg viewBox="258 348 224 328">
            <polyline points="290,380 450,512 290,644" />
          </svg>
          <span>offline</span>
          <span className="offline-cursor" />
        </div>
        <p className="max-w-[46ch] text-soft">No network. Your sessions keep running on this Mac and pick up again when it comes back.</p>
        <div className="flex flex-col items-center gap-3">
          <span className="flex items-center gap-2 text-small text-dim" aria-live="polite">
            <span className="dot animate-[dot-pulse_1.6s_ease-in-out_infinite] bg-idle" aria-hidden="true" />
            {checking ? 'Checking…' : `Checking again in ${nextCheckIn}s`}
          </span>
          <button className="btn" autoFocus disabled={checking} onClick={onRetry}>
            Try now
          </button>
        </div>
      </div>
    </div>
  )
}
