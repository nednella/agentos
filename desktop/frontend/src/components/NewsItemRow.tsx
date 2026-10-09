import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import type { NewsItem } from '../types'

type NewsItemRowProps = { item: NewsItem }

export function NewsItemRow({ item }: NewsItemRowProps) {
  const { report } = useAgentos()

  return (
    <li className="flex flex-col gap-1.5 px-6 py-3">
      <button className="headline text-left text-body font-medium" title={item.url} onClick={() => report(() => api.openURL(item.url))}>
        {item.title}
      </button>
      {item.summary && <p className="line-clamp-2 text-small leading-relaxed text-soft">{item.summary}</p>}
      {item.readTime && <span className="text-small text-dim">{item.readTime}</span>}
    </li>
  )
}
