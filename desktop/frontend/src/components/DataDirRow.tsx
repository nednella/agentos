import { useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import type { DataDirChoice } from '../types'
import { Icon } from './Icon'

type DataDirRowProps = { dir: string; fixed: boolean }

type Busy = '' | 'copying' | 'relaunching'

export function DataDirRow({ dir, fixed }: DataDirRowProps) {
  const { report } = useAgentos()
  const [choice, setChoice] = useState<DataDirChoice | null>(null)
  const [busy, setBusy] = useState<Busy>('')
  const [target, setTarget] = useState('')

  const pick = () =>
    report(async () => {
      const picked = await api.pickDataDir()
      if (picked.dir) setChoice(picked)
    })

  const apply = (to: string, withData: boolean) =>
    report(async () => {
      setChoice(null)
      setTarget(to)
      setBusy(withData ? 'copying' : 'relaunching')
      try {
        await api.setDataDir(to, withData)
        setBusy('relaunching')
      } catch (err) {
        setBusy('')
        throw err
      }
    })

  const cancel = () => setChoice(null)
  const shown = choice?.dir ?? (busy ? target : dir)

  return (
    <div className="border-b border-line py-4 last:border-b-0">
      <p className="text-body font-medium">Data folder</p>
      <p className="mt-0.5 text-small text-dim">
        Where agentos keeps your notes, evidence, stats and digests. {fixed ? 'AGENTOS_DEV_DATA_DIR sets it, so it cannot change here.' : 'Changing it relaunches agentos.'}
      </p>
      <div className="mt-3 flex items-center gap-2">
        <div className="flex h-8 min-w-0 flex-1 items-center gap-2 rounded-md border border-line-strong bg-raised px-2.5 text-soft" title={shown}>
          <Icon name="folder" />
          {/* rtl puts the ellipsis at the start, so the end of a long path stays in view */}
          <span className="mono min-w-0 flex-1 truncate text-left text-small select-text" dir="rtl">
            <bdi>{shown}</bdi>
          </span>
        </div>
        <button className="btn h-8 w-28 flex-none justify-center px-3" disabled={fixed || busy !== '' || choice !== null} onClick={pick}>
          {busy === 'copying' ? 'Copying…' : busy === 'relaunching' ? 'Relaunching…' : 'Change…'}
        </button>
      </div>
      {choice && (
        <div
          role="alertdialog"
          aria-label="Change data folder"
          className="mt-3 flex items-center gap-3"
          onKeyDown={(e) => {
            if (e.key !== 'Escape') return
            e.stopPropagation()
            cancel()
          }}
        >
          <p className="min-w-0 flex-1 text-small text-soft">
            {choice.empty ? 'That folder is empty. Copy your current data into it?' : 'That folder already has data. agentos will use it as it is.'}
          </p>
          <span className="flex flex-none items-center gap-1.5">
            <button className="btn h-7 px-2.5" onClick={cancel}>
              Cancel
            </button>
            {choice.empty && (
              <button className="btn h-7 px-2.5" onClick={() => apply(choice.dir, false)}>
                Start empty
              </button>
            )}
            <button className="btn btn-accent h-7 px-2.5" autoFocus onClick={() => apply(choice.dir, choice.empty)}>
              {choice.empty ? 'Copy and relaunch' : 'Switch and relaunch'}
            </button>
          </span>
        </div>
      )}
    </div>
  )
}
