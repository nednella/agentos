import { useState } from 'react'
import { SegmentedControl } from './SegmentedControl'

const COMMAND = 'command'

type AgentControlProps = { agents: string[]; value: string; onChange(command: string): void }

// AgentControl picks a provider by name, or takes any command to run plain. A command that is a provider's name
// is that provider, so the segment shows it as one.
export function AgentControl({ agents, value, onChange }: AgentControlProps) {
  const [typing, setTyping] = useState(false)
  const isCommand = !agents.includes(value)
  const options = [...agents.map((a) => ({ value: a, label: a })), { value: COMMAND, label: 'Command' }]
  const showField = typing || isCommand

  const submit = (text: string) => {
    setTyping(false)
    const command = text.trim()
    if (command && command !== value) onChange(command)
  }

  return (
    <div className="flex flex-none flex-col items-end gap-2">
      <SegmentedControl
        label="Agent"
        options={options}
        value={isCommand ? COMMAND : value}
        onChange={(picked) => {
          if (picked === COMMAND) setTyping(true)
          else {
            setTyping(false)
            onChange(picked)
          }
        }}
      />
      {showField && (
        <input
          key={value}
          autoFocus={typing}
          defaultValue={isCommand ? value : ''}
          placeholder="e.g. codex"
          aria-label="Agent command"
          spellCheck={false}
          className="field mono h-7 w-48 text-small"
          onKeyDown={(e) => {
            if (e.key === 'Enter') e.currentTarget.blur()
            if (e.key === 'Escape') {
              e.currentTarget.value = isCommand ? value : ''
              e.currentTarget.blur()
            }
          }}
          onBlur={(e) => submit(e.currentTarget.value)}
        />
      )}
    </div>
  )
}
