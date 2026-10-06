import type { Action, ActionContext } from './types'

export const viewActions = ({ a, layout, current }: ActionContext): Action[] => [
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
  {
    id: 'settings',
    label: 'Settings',
    group: 'App',
    shortcut: { key: ',' },
    run: () => a.setOverlay('settings'),
  },
  {
    id: 'stats',
    label: 'Stats: what interrupts you',
    group: 'App',
    shortcut: { key: 's', shift: true },
    uiCommand: 'stats',
    run: layout.toggleStats,
  },
  {
    id: 'show-terminal',
    label: 'Show terminal',
    group: 'Sessions',
    uiCommand: 'term',
    run() {
      if (!current) throw 'No session selected'
      layout.closeCentre()
      a.setSessionView(current.id, 'terminal')
    },
  },
  {
    id: 'show-browser',
    label: 'Show browser',
    group: 'Sessions',
    uiCommand: 'browser',
    async run(args) {
      if (!current) throw 'No session selected'
      layout.closeCentre()
      if (args.length === 0) {
        a.setSessionView(current.id, 'browser')
        return
      }
      await a.openBrowser(current.id, args.join(' '))
      return `Opened ${args.join(' ')}`
    },
  },
  {
    id: 'show-evidence',
    label: 'Show evidence',
    group: 'Sessions',
    uiCommand: 'evidence',
    run() {
      if (!current) throw 'No session selected'
      layout.closeCentre()
      a.setSessionView(current.id, 'evidence')
    },
  },
  {
    id: 'view-previous',
    label: 'Previous view (terminal, browser, evidence)',
    group: 'Navigate',
    shortcut: { key: '[', code: 'BracketLeft', shift: true, label: '[' },
    palette: false,
    run: () => a.cycleSessionView(-1),
  },
  {
    id: 'view-next',
    label: 'Next view (terminal, browser, evidence)',
    group: 'Navigate',
    shortcut: { key: ']', code: 'BracketRight', shift: true, label: ']' },
    palette: false,
    run: () => a.cycleSessionView(1),
  },
  {
    id: 'digest',
    label: 'Weekly digest',
    group: 'App',
    shortcut: { key: 'd', shift: true },
    uiCommand: 'digest',
    run: layout.toggleDigest,
  },
  {
    id: 'shortcuts',
    label: 'Keyboard shortcuts',
    group: 'App',
    shortcut: { key: '/' },
    run: () => a.setOverlay('shortcuts'),
  },
]
