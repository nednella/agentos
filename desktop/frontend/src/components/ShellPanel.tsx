import { ShellTabs } from './ShellTabs'
import { ShellTerminal } from './ShellTerminal'

export function ShellPanel() {
  return (
    <section data-panel="shell" tabIndex={-1} className="panel flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-term">
      <ShellTabs />
      <div className="relative min-h-0 flex-1">
        <ShellTerminal visible />
      </div>
    </section>
  )
}
