import { useAgentos } from './AgentosContext'
import type { Overlay } from './AgentosContext'
import { useLayout } from './LayoutContext'

export type Shortcut = {
  key: string
  shift?: boolean
  alt?: boolean
  code?: string
  aliases?: string[]
  anyShift?: boolean
  unlessTyping?: boolean
  label?: string
}
export type ActionGroup = 'Navigate' | 'Projects' | 'App'

export type Action = {
  id: string
  label: string
  group: ActionGroup
  shortcut?: Shortcut
  run(args: string[]): string | void | Promise<string | void>
}

const KEY_LABELS: Record<string, string> = { arrowleft: '←', arrowright: '→', arrowup: '↑', arrowdown: '↓' }

export function formatShortcut(shortcut: Shortcut): string {
  const key = shortcut.label ?? KEY_LABELS[shortcut.key] ?? shortcut.key.toUpperCase()
  return `⌘${shortcut.alt ? '⌥' : ''}${shortcut.shift ? '⇧' : ''}${key}`
}

export function matchShortcut(e: KeyboardEvent, shortcut: Shortcut): boolean {
  if (!e.metaKey || e.ctrlKey || e.altKey !== Boolean(shortcut.alt)) return false
  if (!shortcut.anyShift && e.shiftKey !== Boolean(shortcut.shift)) return false
  if (shortcut.code) return e.code === shortcut.code
  return [shortcut.key, ...(shortcut.aliases ?? [])].includes(e.key.toLowerCase())
}



export function useActions(): Action[] {
  const a = useAgentos()
  const layout = useLayout()
  const openOverlay = (overlay: Overlay) => () => a.setOverlay(overlay)

  const actions: Action[] = [
    {
      id: 'switch-project',
      label: 'Switch project…',
      group: 'Projects',
      shortcut: { key: 'p' },
      run: openOverlay('projects'),
    },
    {
      id: 'add-project',
      label: 'Add project…',
      group: 'Projects',
      async run() {
        await a.addProject()
      },
    },
    {
      id: 'toggle-sidebar',
      label: 'Toggle queue and notes panel',
      group: 'Navigate',
      shortcut: { key: 'b' },
      run: layout.toggleSidebar,
    },
    {
      id: 'zoom-in',
      label: 'Larger text',
      group: 'App',
      shortcut: { key: '=', aliases: ['+'], anyShift: true, label: '+' },
      run: () => layout.zoom(1),
    },
    {
      id: 'zoom-out',
      label: 'Smaller text',
      group: 'App',
      shortcut: { key: '-', aliases: ['_'], anyShift: true },
      run: () => layout.zoom(-1),
    },
    {
      id: 'zoom-reset',
      label: 'Reset text size',
      group: 'App',
      shortcut: { key: '0' },
      run: () => layout.zoom(0),
    },
  ]

  return actions
}
