import { useAgentos } from '../AgentosContext'
import type { Session } from '../types'

type BrowserRecentCapturesProps = { session: Session }

const SHOWN = 3

export function BrowserRecentCaptures({ session }: BrowserRecentCapturesProps) {
  const { evidence, setSessionView } = useAgentos()
  const recent = (evidence[session.id] ?? [])
    .filter((item) => item.kind === 'image')
    .sort((a, b) => b.at - a.at)
    .slice(0, SHOWN)

  return (
    <section>
      <h4 className="label pb-2">Recent captures</h4>
      {recent.length === 0 ? (
        <p className="text-small text-dim">No captures yet</p>
      ) : (
        <ul className="grid grid-cols-3 gap-2">
          {recent.map((item) => (
            <li key={item.id} className="min-w-0">
              <button
                className="block aspect-[16/10] w-full overflow-hidden rounded-md border border-line bg-term"
                title={item.caption || 'Open in Evidence'}
                onClick={() => setSessionView(session.id, 'evidence')}
              >
                <img src={item.url} alt={item.caption || 'Capture'} className="h-full w-full object-cover" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
