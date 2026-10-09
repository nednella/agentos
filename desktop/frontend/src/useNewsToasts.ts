import { useEffect, useRef } from 'react'
import { useAgentos } from './AgentosContext'
import { useLayout } from './LayoutContext'
import type { NewsPage } from './LayoutContext'

export function useNewsToasts() {
  const { news, digest, pushToast, dismissToast } = useAgentos()
  const { newsOpen, newsPage, openNews } = useLayout()
  const latest = useRef({ newsOpen, newsPage, pushToast, dismissToast, openNews })
  latest.current = { newsOpen, newsPage, pushToast, dismissToast, openNews }
  const previousNews = useRef(news)
  const previousDigest = useRef(digest)

  useEffect(() => {
    const before = previousNews.current
    previousNews.current = news
    if (!before?.running || !news || news.running) return
    const seen = Math.max(0, ...before.issues.map((i) => i.fetchedAt))
    const fresh = news.issues.filter((i) => i.fetchedAt > seen).reduce((total, i) => total + i.items.length, 0)
    if (fresh > 0) announce(latest.current, 'tldr', `TLDR Dev: ${fresh} new ${fresh === 1 ? 'story' : 'stories'}`)
  }, [news])

  useEffect(() => {
    const before = previousDigest.current
    previousDigest.current = digest
    if (!before?.running || !digest || digest.running || before.project !== digest.project) return
    const seen = Math.max(0, ...before.items.map((i) => i.at))
    const fresh = digest.items.filter((i) => !i.noteId && i.at > seen).length
    if (fresh > 0) announce(latest.current, 'digest', `Weekly digest: ${fresh} new ${fresh === 1 ? 'item' : 'items'}`)
  }, [digest])
}

type Announcer = {
  newsOpen: boolean
  newsPage: NewsPage
  pushToast: ReturnType<typeof useAgentos>['pushToast']
  dismissToast(key: number): void
  openNews(page: NewsPage): void
}

function announce({ newsOpen, newsPage, pushToast, dismissToast, openNews }: Announcer, page: NewsPage, text: string) {
  if (newsOpen && newsPage === page) return
  const key = pushToast({
    tone: 'info',
    text,
    action: {
      label: 'Open',
      run() {
        openNews(page)
        dismissToast(key)
      },
    },
  })
}
