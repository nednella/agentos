import { useAgentos } from '../AgentosContext'
import { ShellList } from './ShellList'
import { ShellTerminal } from './ShellTerminal'

export function ShellPanel() {
  const { shellIds } = useAgentos()
  return (
    <section data-panel="shell" tabIndex={-1} className="panel flex h-full min-h-0 min-w-0 flex-1 bg-term">
      <div className="relative min-w-0 flex-1">
        <ShellTerminal visible />
      </div>
      {shellIds.length > 0 && <ShellList />}
    </section>
  )
}
