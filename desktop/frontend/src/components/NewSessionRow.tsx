import { useAgentos } from '../AgentosContext'
import { Icon } from './Icon'
import { InlineInput } from './InlineInput'
import { Keycap } from './Keycap'

type NewSessionRowProps = { cursor: boolean; compact: boolean }

export function NewSessionRow({ cursor, compact }: NewSessionRowProps) {
  const { composing, setComposing, newSession, report } = useAgentos()

  if (composing) {
    return (
      <div className="flex-none border-t border-line p-3">
        <InlineInput
          placeholder="Title (optional)"
          blur="cancel"
          action="Start"
          icon={<Icon name="plus" />}
          onSubmit={(title) => {
            setComposing(false)
            report(() => newSession(title.trim()))
          }}
          onCancel={() => setComposing(false)}
        />
      </div>
    )
  }
  return (
    <button
      className="row h-11 flex-none items-center gap-2 border-t border-line px-4 text-soft"
      data-cursor={cursor}
      onClick={() => setComposing(true)}
    >
      <Icon name="plus" />
      <span className="text-body">New session</span>
      {!compact && (
        <span className="ml-auto">
          <Keycap>⌘N</Keycap>
        </span>
      )}
    </button>
  )
}
