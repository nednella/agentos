import { useAgentos } from '../AgentosContext'
import { Icon } from './Icon'
import { InlineInput } from './InlineInput'
import { Keycap } from './Keycap'

type NewSessionRowProps = { cursor: boolean; compact: boolean }

export function NewSessionRow({ cursor, compact }: NewSessionRowProps) {
  const { composing, setComposing, newSession, newChat, report } = useAgentos()

  if (composing) {
    const chat = composing === 'chat'
    return (
      <div className="flex-none border-t border-line p-3">
        <InlineInput
          key={composing}
          placeholder={chat ? 'Chat title (optional)' : 'Session title (optional)'}
          blur="cancel"
          action="Start"
          icon={<Icon name="plus" />}
          onSubmit={(text) => {
            setComposing(null)
            const title = text.trim()
            report(() => (chat ? newChat(title) : newSession(title)))
          }}
          onCancel={() => setComposing(null)}
        />
      </div>
    )
  }
  return (
    <div className="flex h-11 flex-none border-t border-line">
      <button className="row flex-1 items-center gap-2 px-4 text-soft" title="Ask a question about the project; the chat cannot change files" onClick={() => setComposing('chat')}>
        <Icon name="plus" />
        <span className="text-body">Chat</span>
        {!compact && (
          <span className="ml-auto">
            <Keycap>⌘M</Keycap>
          </span>
        )}
      </button>
      <span className="w-px flex-none bg-line" />
      <button className="row flex-1 items-center gap-2 px-4 text-soft" data-cursor={cursor} onClick={() => setComposing('session')}>
        <Icon name="plus" />
        <span className="text-body">Session</span>
        {!compact && (
          <span className="ml-auto">
            <Keycap>⌘N</Keycap>
          </span>
        )}
      </button>
    </div>
  )
}
