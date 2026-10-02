import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import type { DigestItem } from '../types'

type DigestItemRowProps = { item: DigestItem }

export function DigestItemRow({ item }: DigestItemRowProps) {
  const { digestToNote, dismissDigestItem, report } = useAgentos()
  const saved = item.noteId !== ''

  return (
    <li className="flex flex-col gap-1 px-4 py-3">
      <div className="flex items-baseline gap-2">
        <span className="flex-none rounded-sm border border-line-strong px-1.5 text-label text-soft">{item.source}</span>
        <button className="link min-w-0 truncate text-left text-body font-medium" title={item.url} onClick={() => report(() => api.openURL(item.url))}>
          {item.title}
        </button>
      </div>
      <p className="text-small text-soft">{item.why}</p>
      <div className="mt-1 flex items-center gap-2">
        {saved ? (
          <span className="text-small text-finished">Saved as a note</span>
        ) : (
          <button className="btn h-7" onClick={() => report(() => digestToNote(item.id))}>
            Save as note
          </button>
        )}
        <button className="btn btn-ghost h-7" onClick={() => report(() => dismissDigestItem(item.id))}>
          Dismiss
        </button>
      </div>
    </li>
  )
}
