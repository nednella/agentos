type OfflineSplashProps = {
  retrying: boolean
  onRetry: () => void
}

export function OfflineSplash({ retrying, onRetry }: OfflineSplashProps) {
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
        <div className="flex h-[1.875rem] items-center">
          {retrying ? (
            <span className="spinner" role="status" aria-label="Checking the network" />
          ) : (
            <button className="btn" autoFocus onClick={onRetry}>
              Try now
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
