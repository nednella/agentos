import { useState } from 'react'
import { useAgentos } from '../AgentosContext'
import type { Session } from '../types'

type BrowserEmptyProps = { session: Session; error: string }

export function BrowserEmpty({ session, error }: BrowserEmptyProps) {
  const { openBrowser, report } = useAgentos()
  const [url, setUrl] = useState('')

  return (
    <div className="absolute inset-0 grid place-items-center overflow-y-auto p-6">
      <form
        className="flex w-full max-w-md flex-col gap-3 text-center"
        onSubmit={(e) => {
          e.preventDefault()
          report(() => openBrowser(session.id, url.trim()))
        }}
      >
        <h3 className="text-title font-semibold">No page open</h3>
        <p className="text-soft">The agent can drive this browser, and so can you. It stays logged in across this project.</p>
        <div className="flex gap-2">
          <input
            value={url}
            autoFocus
            spellCheck={false}
            aria-label="Address"
            placeholder="Address, or leave empty for the project page"
            className="field min-w-0 flex-1"
            onChange={(e) => setUrl(e.target.value)}
          />
          <button type="submit" className="btn btn-accent">
            Open
          </button>
        </div>
        {error && <p className="text-small text-danger">{error}</p>}
      </form>
    </div>
  )
}
