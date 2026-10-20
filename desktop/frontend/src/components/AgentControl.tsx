type AgentControlProps = { agents: string[]; value: string; onChange(command: string): void }

// AgentControl takes a provider's name, offered as a suggestion, or any command to run plain.
export function AgentControl({ agents, value, onChange }: AgentControlProps) {
  const submit = (text: string) => {
    const command = text.trim()
    if (command && command !== value) onChange(command)
  }

  return (
    <>
      <input
        key={value}
        defaultValue={value}
        list="agent-names"
        aria-label="Default inference provider"
        spellCheck={false}
        className="field mono h-7 w-48 flex-none text-small"
        onKeyDown={(e) => {
          if (e.key === 'Enter') e.currentTarget.blur()
          if (e.key === 'Escape') {
            e.currentTarget.value = value
            e.currentTarget.blur()
          }
        }}
        onBlur={(e) => submit(e.currentTarget.value)}
      />
      <datalist id="agent-names">
        {agents.map((a) => (
          <option key={a} value={a} />
        ))}
      </datalist>
    </>
  )
}
