import { useMemo, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { readStored, writeStored } from '../storage'
import type { Session } from '../types'
import { EvidenceCard } from './EvidenceCard'
import { EvidenceCompare } from './EvidenceCompare'
import { EvidenceViewer } from './EvidenceViewer'
import { Icon } from './Icon'

type EvidenceTabProps = { session: Session }

const ORDER_KEY = 'agentos.evidenceNewest'

export function EvidenceTab({ session }: EvidenceTabProps) {
  const { evidence } = useAgentos()
  const [newestFirst, setNewestFirst] = useState(() => readStored(ORDER_KEY, false))
  const [comparing, setComparing] = useState(false)
  const [picked, setPicked] = useState<string[]>([])
  const [viewing, setViewing] = useState<string | null>(null)
  const [showCompare, setShowCompare] = useState(false)

  const items = useMemo(() => {
    const list = [...(evidence[session.id] ?? [])].sort((a, b) => a.at - b.at)
    return newestFirst ? list.reverse() : list
  }, [evidence, session.id, newestFirst])
  const images = items.filter((i) => i.kind === 'image')
  const [left, right] = picked.map((id) => items.find((i) => i.id === id))

  const toggle = (id: string) => setPicked((list) => (list.includes(id) ? list.filter((x) => x !== id) : [...list.slice(-1), id]))

  if (items.length === 0) {
    return (
      <div className="absolute inset-0 grid place-items-center p-6">
        <div className="flex max-w-sm flex-col items-center gap-2 text-center">
          <Icon name="image" size={28} />
          <h3 className="text-title font-semibold">Nothing to show yet</h3>
          <p className="text-soft">Screenshots, before and after pairs and short notes from this session appear here. The agent adds them with</p>
          <code className="mono rounded-sm border border-line-strong px-2 py-1 text-small">agentos show image.png --caption "…"</code>
          <p className="text-small text-dim">You can capture the browser yourself from its toolbar.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="absolute inset-0 flex flex-col">
      <div className="flex flex-none flex-wrap items-center gap-2 border-b border-line bg-surface px-3 py-2">
        <span className="mono text-small text-dim">{items.length} items</span>
        <span className="ml-auto" />
        {comparing && (
          <button className="btn btn-accent" disabled={picked.length !== 2} onClick={() => setShowCompare(true)}>
            Compare {picked.length} of 2
          </button>
        )}
        <button
          className="btn"
          aria-pressed={comparing}
          disabled={images.length < 2}
          title="Pick two images to see them side by side"
          onClick={() => {
            setComparing(!comparing)
            setPicked([])
          }}
        >
          <Icon name="compare" /> {comparing ? 'Done' : 'Compare'}
        </button>
        <button
          className="btn"
          onClick={() => {
            writeStored(ORDER_KEY, !newestFirst)
            setNewestFirst(!newestFirst)
          }}
        >
          {newestFirst ? 'Newest first' : 'Oldest first'}
        </button>
      </div>
      <div className="grid min-h-0 flex-1 grid-cols-[repeat(auto-fill,minmax(13rem,1fr))] content-start gap-3 overflow-y-auto p-3">
        {items.map((item) => (
          <EvidenceCard
            key={item.id}
            item={item}
            selectable={comparing && item.kind === 'image'}
            selected={picked.includes(item.id)}
            onOpen={() => setViewing(item.id)}
            onToggle={() => toggle(item.id)}
          />
        ))}
      </div>
      {viewing && <EvidenceViewer sessionId={session.id} items={images} startId={viewing} onClose={() => setViewing(null)} />}
      {showCompare && left && right && <EvidenceCompare left={left} right={right} onClose={() => setShowCompare(false)} />}
    </div>
  )
}
