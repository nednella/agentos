import { useAgentos } from '../AgentosContext'
import { api } from '../api'

type NoteTextProps = { text: string }

const INLINE = /(`[^`\n]+`)|(https?:\/\/[^\s)]+)/g

export function NoteText({ text }: NoteTextProps) {
  const { report } = useAgentos()
  const parts: React.ReactNode[] = []
  let last = 0
  for (const match of text.matchAll(INLINE)) {
    const at = match.index ?? 0
    if (at > last) parts.push(text.slice(last, at))
    const [token, code] = match
    if (code) {
      parts.push(
        <code key={at} className="mono rounded-sm px-1 text-[0.92em]" style={{ background: 'rgba(0,0,0,0.3)' }}>
          {token.slice(1, -1)}
        </code>,
      )
    } else {
      parts.push(
        <span
          key={at}
          role="link"
          tabIndex={0}
          className="link cursor-pointer break-all"
          onClick={(e) => {
            e.stopPropagation()
            report(() => api.openURL(token))
          }}
          onKeyDown={(e) => {
            if (e.key !== 'Enter') return
            e.stopPropagation()
            report(() => api.openURL(token))
          }}
        >
          {token}
        </span>,
      )
    }
    last = at + token.length
  }
  if (last < text.length) parts.push(text.slice(last))
  return <>{parts}</>
}
