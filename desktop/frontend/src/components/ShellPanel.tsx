import { ShellTerminal } from './ShellTerminal'

export function ShellPanel() {
  return (
    <section data-panel="shell" tabIndex={-1} className="panel relative h-full min-h-0 min-w-0 flex-1 bg-term">
      <ShellTerminal visible />
    </section>
  )
}
