import { useState } from 'react'
import { useAgentos } from './AgentosContext'
import { readStored, writeStored } from './storage'

export function useProjectStored<T>(name: string, fallback: T): [T, (value: T) => void] {
  const { project } = useAgentos()
  const key = `agentos.${name}.${project?.name ?? ''}`
  const [stored, setStored] = useState(() => ({ key, value: readStored(key, fallback) }))

  let current = stored
  if (stored.key !== key) {
    current = { key, value: readStored(key, fallback) }
    setStored(current)
  }

  const store = (value: T) => {
    writeStored(key, value)
    setStored({ key, value })
  }

  return [current.value, store]
}
