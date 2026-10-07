import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import { ago, useNow } from '../time'
import type { BrowserState } from '../types'
import { CaptureButton } from './CaptureButton'
import { Icon } from './Icon'

type BrowserStatusHeaderProps = { id: string; state: BrowserState }

export function BrowserStatusHeader({ id, state }: BrowserStatusHeaderProps) {
  const { report } = useAgentos()
  const now = useNow()

  return (
    <header className="flex flex-none flex-wrap items-center gap-x-3 gap-y-2 border-b border-line bg-surface px-4 py-3">
      <div className="min-w-0 flex-1 basis-48">
        <h3 className="truncate text-title font-semibold" title={state.title}>
          {state.title || 'Untitled page'}
        </h3>
        <p className="mt-0.5 flex items-center gap-2 text-small text-dim">
          <span className="mono min-w-0 truncate" title={state.url}>
            {state.url.replace(/^https?:\/\//, '')}
          </span>
          {state.loading ? (
            <span className="flex flex-none items-center gap-1.5">
              <span className="spinner" role="status" aria-label="Loading" />
              loading
            </span>
          ) : (
            <span className="flex-none">{state.loadedAt ? `· loaded ${ago(state.loadedAt, now)}` : ''}</span>
          )}
        </p>
      </div>
      <div className="flex flex-none items-center gap-1.5">
        <button className="btn" onClick={() => report(() => api.browserShow(id))}>
          <Icon name="external" />
          Show window
        </button>
        <CaptureButton id={id} />
        <button className="btn btn-ghost h-8 w-8 justify-center px-0" title="Close this browser window" aria-label="Close browser window" onClick={() => report(() => api.browserClose(id))}>
          <Icon name="close" />
        </button>
      </div>
    </header>
  )
}
