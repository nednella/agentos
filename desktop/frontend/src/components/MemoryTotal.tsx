import { useAgentos } from '../AgentosContext'
import { gigabytes, memoryNote } from '../memory'

export function MemoryTotal() {
  const { memory } = useAgentos()
  if (!memory || memory.total === 0) return null
  const machine = memory.machine > 0 ? `\nof ${Math.round(memory.machine / 2 ** 30)} GB` : ''
  const parts = [`agentos ${gigabytes(memory.app)}`, `Browsers ${gigabytes(memory.browsers)}`, `Sessions and shells ${gigabytes(memory.sessions)}`]
  return (
    <>
      <span className="mono flex-none cursor-default pr-3 pl-1 text-small whitespace-nowrap text-dim" title={`${parts.join('\n')}${machine}\n\n${memoryNote}`}>
        Memory <span className="text-soft">{gigabytes(memory.total)}</span>
      </span>
      <span className="mr-1 h-[18px] w-px flex-none bg-line-strong" />
    </>
  )
}
