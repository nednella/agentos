import { useEffect, useRef } from 'react'

type AgentControlProps = { agents: string[]; value: string; onChange(command: string): void }

// A pick from the datalist is an input event that is not typing: a plain Event in WebKit, insertReplacementText in Chromium.
const isPick = (e: Event) => !(e instanceof InputEvent) || e.inputType === 'insertReplacementText'

// AgentControl takes a provider's name, offered as a suggestion, or any command to run plain.
export function AgentControl({ agents, value, onChange }: AgentControlProps) {
  const submit = (text: string) => {
    const command = text.trim()
    if (command && command !== value) onChange(command)
  }

  // A panel that closes with the field focused removes it with no blur.
  const editing = useRef<HTMLInputElement | null>(null)
  const submitOnClose = useRef(submit)
  submitOnClose.current = submit
  useEffect(
    () => () => {
      if (editing.current) submitOnClose.current(editing.current.value)
    },
    [],
  )

  return (
    <>
      <input
        key={value}
        defaultValue={value}
        list="agent-names"
        aria-label="Default inference provider"
        spellCheck={false}
        className="field mono h-7 w-48 flex-none text-small"
        onChange={(e) => {
          if (isPick(e.nativeEvent) && agents.includes(e.currentTarget.value)) submit(e.currentTarget.value)
        }}
        onKeyDown={(e) => {
          if (e.key === 'Enter') e.currentTarget.blur()
          if (e.key === 'Escape') {
            e.currentTarget.value = value
            e.currentTarget.blur()
          }
        }}
        onFocus={(e) => (editing.current = e.currentTarget)}
        onBlur={(e) => {
          editing.current = null
          submit(e.currentTarget.value)
        }}
      />
      <datalist id="agent-names">
        {agents.map((a) => (
          <option key={a} value={a} />
        ))}
      </datalist>
    </>
  )
}
