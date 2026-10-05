import { useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import type { BrowserState } from '../types'
import { Icon } from './Icon'
import { InlineInput } from './InlineInput'

type BrowserToolbarProps = { id: string; state: BrowserState }

export function BrowserToolbar({ id, state }: BrowserToolbarProps) {
  const { report, pushToast } = useAgentos()
  const [draft, setDraft] = useState<string | null>(null)
  const [capturing, setCapturing] = useState(false)
  const nav = (action: 'back' | 'forward' | 'reload' | 'stop') => report(() => api.browserNav(id, action))
  const iconButton = 'btn btn-ghost h-8 w-8 justify-center px-0'

  return (
    <div className="flex flex-none flex-wrap items-center gap-1.5 border-b border-line bg-surface px-2 py-1.5">
      <button className={iconButton} disabled={!state.canGoBack} title="Back" aria-label="Back" onClick={() => nav('back')}>
        <Icon name="back" />
      </button>
      <button className={iconButton} disabled={!state.canGoForward} title="Forward" aria-label="Forward" onClick={() => nav('forward')}>
        <Icon name="forward" />
      </button>
      <button
        className={iconButton}
        title={state.loading ? 'Stop' : 'Reload'}
        aria-label={state.loading ? 'Stop' : 'Reload'}
        onClick={() => nav(state.loading ? 'stop' : 'reload')}
      >
        <Icon name={state.loading ? 'stop' : 'reload'} />
      </button>
      <input
        value={draft ?? state.url}
        spellCheck={false}
        aria-label="Address"
        className="field mono min-w-[8rem] flex-1 text-small"
        onFocus={(e) => e.currentTarget.select()}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => setDraft(null)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.stopPropagation()
            setDraft(null)
            e.currentTarget.blur()
          }
          if (e.key !== 'Enter' || !draft?.trim()) return
          const url = draft.trim()
          setDraft(null)
          e.currentTarget.blur()
          report(() => api.browserGoto(id, url))
        }}
      />
      {state.loading && <span className="spinner" role="status" aria-label="Loading" />}
      {state.title && !capturing && (
        <span className="hidden max-w-[12rem] truncate text-small text-dim lg:block" title={state.title}>
          {state.title}
        </span>
      )}
      {capturing ? (
        <InlineInput
          placeholder="Caption (optional), Enter to capture"
          blur="cancel"
          className="w-64 max-w-full"
          onSubmit={(caption) => {
            setCapturing(false)
            report(async () => {
              await api.browserScreenshot(id, caption.trim())
              pushToast({ tone: 'info', text: 'Captured to Evidence' })
            })
          }}
          onCancel={() => setCapturing(false)}
        />
      ) : (
        <button className="btn" title="Capture the page as evidence" onClick={() => setCapturing(true)}>
          <Icon name="camera" />
          <span className="hidden sm:inline">Capture</span>
        </button>
      )}
      <button className={iconButton} title="Close this browser tab" aria-label="Close browser tab" onClick={() => report(() => api.browserClose(id))}>
        <Icon name="close" />
      </button>
    </div>
  )
}
