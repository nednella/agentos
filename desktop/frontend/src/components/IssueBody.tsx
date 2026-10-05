import type { MouseEvent } from 'react'
import { useAgentos } from '../AgentosContext'
import { api } from '../api'

type IssueBodyProps = { html: string }

// GitHub renders and sanitizes this HTML for its own pages; the app shows it as it is.
export function IssueBody({ html }: IssueBodyProps) {
  const { report } = useAgentos()

  const onClick = (e: MouseEvent<HTMLDivElement>) => {
    const link = (e.target as HTMLElement).closest('a')
    if (!link) return
    e.preventDefault()
    if (/^https?:\/\//.test(link.href)) report(() => api.openURL(link.href))
  }

  return <div className="markdown text-body" onClick={onClick} dangerouslySetInnerHTML={{ __html: html }} />
}
