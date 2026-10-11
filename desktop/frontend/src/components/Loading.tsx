export function Loading() {
  return (
    <div role="status" aria-label="Loading" className="flex h-full flex-1 items-center justify-center px-4 py-16">
      <span className="prompt-loader" aria-hidden="true">
        <svg viewBox="258 348 224 328">
          <polyline points="290,380 450,512 290,644" />
        </svg>
        <span className="prompt-loader-line">
          <i />
          <i />
          <i />
        </span>
        <span className="prompt-loader-cursor" />
      </span>
    </div>
  )
}
