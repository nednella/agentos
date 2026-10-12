import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import type { NewsItem } from '../types'

type NewsItemRowProps = { item: NewsItem; unread: boolean }

export function NewsItemRow({ item, unread }: NewsItemRowProps) {
  const { report, markNewsRead } = useAgentos()

  return (
    <li className="relative flex flex-col gap-1.5 px-6 py-3">
      {unread && <span className="dot absolute top-[1.2rem] left-2.5" style={{ ['--c' as string]: 'var(--accent)' }} aria-label="New" />}
      <button
        className="headline text-left text-body font-medium"
        title={item.url}
        onClick={() => {
          markNewsRead(item.id)
          report(() => api.openURL(item.url))
        }}
      >
        {item.title}
      </button>
      {item.summary && <p className="line-clamp-2 text-small leading-relaxed text-soft">{item.summary}</p>}
      {item.readTime && <span className="text-small text-dim">{item.readTime}</span>}
    </li>
  )
}
