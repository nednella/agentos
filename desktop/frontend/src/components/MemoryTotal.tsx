import { useAgentos } from '../AgentosContext'
import { gigabytes, memoryNote } from '../memory'

export function MemoryTotal() {
  const { memory } = useAgentos()
  if (!memory || memory.total === 0) return null
  const machine = memory.machine > 0 ? ` of ${Math.round(memory.machine / 2 ** 30)} GB` : ''
  const parts = [`agentos ${gigabytes(memory.app)}`, `Browsers ${gigabytes(memory.browsers)}`, `Sessions and shells ${gigabytes(memory.sessions)}`]
  return (
    <span className="mono flex-none text-small whitespace-nowrap text-dim" title={`${parts.join('\n')}\n\n${memoryNote}`}>
      {gigabytes(memory.total)}
      {machine}
    </span>
  )
}
