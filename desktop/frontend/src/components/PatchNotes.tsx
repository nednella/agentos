import { useEffect, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import type { IssueType, Release } from '../types'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'
import { TypeMark } from './TypeMark'

const SECTION_TYPE: Record<string, IssueType> = { Features: 'feature', 'Bug Fixes': 'bug' }

const shortDate = (date: string) => (date ? new Date(`${date}T12:00:00`).toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) : '')

export function PatchNotes() {
  const { overlay, setOverlay, patchNotes } = useAgentos()
  if (overlay !== 'patch-notes' || patchNotes.length === 0) return null
  return <PatchNotesSheet releases={patchNotes} onClose={() => setOverlay(null)} />
}

type PatchNotesSheetProps = { releases: Release[]; onClose(): void }

function PatchNotesSheet({ releases, onClose }: PatchNotesSheetProps) {
  const { report } = useAgentos()
  const [index, setIndex] = useState(0)
  const release = releases[index]
  const several = releases.length > 1

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'ArrowUp') setIndex((i) => Math.max(i - 1, 0))
      else if (e.key === 'ArrowDown') setIndex((i) => Math.min(i + 1, releases.length - 1))
      else return
      e.preventDefault()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [releases.length])

  return (
    <Overlay align="center" size="wide" label="Patch notes" onClose={onClose}>
      <div className="flex flex-none items-center gap-3 border-b border-line px-4 py-2.5">
        <h2 className="flex-none text-title font-semibold">{several ? 'Patch notes' : `What's new in ${release.version}`}</h2>
        {several && <span className="min-w-0 flex-1 truncate text-small text-dim">{releases.length} releases since you last opened agentos</span>}
        <span className="flex-1" />
        <Keycap>Esc</Keycap>
      </div>
      <div className="flex h-[min(32rem,70vh)] min-h-0">
        {several && (
          <ul className="w-40 flex-none overflow-y-auto border-r border-line py-2" aria-label="Releases">
            {releases.map((r, i) => (
              <li key={r.version}>
                <button
                  aria-current={i === index}
                  className={`flex w-full items-baseline justify-between px-3.5 py-1.5 text-left ${i === index ? 'bg-hover text-ink shadow-[inset_2px_0_0_var(--accent)]' : 'text-soft hover:bg-hover'}`}
                  onClick={() => setIndex(i)}
                >
                  <span className="mono text-small">{r.version}</span>
                  <span className="text-label text-dim">{shortDate(r.date)}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="min-w-0 flex-1 overflow-y-auto px-5 py-3">
          {release.sections.map((section) => (
            <section key={section.title} className="pb-4">
              <h3 className="label pb-1">{section.title}</h3>
              <ul>
                {section.changes.map((change, i) => (
                  <li key={i} className="flex items-baseline gap-3 border-b border-line py-1.5 last:border-b-0">
                    <span className="flex-none self-center">
                      <TypeMark type={SECTION_TYPE[section.title] ?? 'chore'} />
                    </span>
                    <span className="mono w-16 flex-none truncate text-label text-dim">{change.scope}</span>
                    <span className="min-w-0 flex-1 text-body">{change.text}</span>
                    {change.url && (
                      <button className="link mono flex-none text-label" onClick={() => report(() => api.openURL(change.url))}>
                        #{change.issue}
                      </button>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      </div>
      {several && (
        <div className="flex flex-none items-center gap-2 border-t border-line px-4 py-2 text-small text-dim">
          <Keycap>↑ / ↓</Keycap> release
          <span className="flex-1" />
          <Keycap>Esc</Keycap> close
        </div>
      )}
    </Overlay>
  )
}
