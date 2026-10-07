import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import type { BrowserState, Session } from '../types'
import { BrowserToolbar } from './BrowserToolbar'

type BrowserHeadedProps = { session: Session; state: BrowserState }

export function BrowserHeaded({ session, state }: BrowserHeadedProps) {
  const { report } = useAgentos()

  return (
    <div className="absolute inset-0 flex flex-col">
      <BrowserToolbar id={session.id} state={state} />
      <div className="grid min-h-0 flex-1 place-items-center overflow-y-auto p-6">
        <div className="flex w-full max-w-md flex-col items-center gap-3 text-center">
          <h3 className="max-w-full truncate text-title font-semibold">{state.title || 'Untitled page'}</h3>
          <p className="mono max-w-full truncate text-small text-dim">{state.url}</p>
          <button className="btn btn-accent" onClick={() => report(() => api.browserShow(session.id))}>
            Show window
          </button>
          <p className="text-soft">This page is open in its own browser window.</p>
        </div>
      </div>
    </div>
  )
}
