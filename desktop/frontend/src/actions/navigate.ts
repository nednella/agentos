import { api } from '../api'
import type { Action, ActionContext } from './types'

export const navigateActions = ({ a, layout }: ActionContext): Action[] => [
  {
    id: 'switch-project',
    label: 'Switch project…',
    group: 'Projects',
    shortcut: { key: 'p' },
    run: () => a.setOverlay('projects'),
  },
  {
    id: 'open-repo',
    label: 'Open repository on GitHub',
    group: 'Projects',
    async run() {
      if (!a.project?.repo) throw 'This project has no GitHub repo'
      await api.openURL(`https://github.com/${a.project.repo}`)
    },
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
    id: 'set-up-project',
    label: 'Set up project',
    group: 'Projects',
    async run() {
      await a.setUpProject()
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
    id: 'toggle-sessions',
    label: 'Toggle sessions panel',
    group: 'Navigate',
    shortcut: { key: 'b', code: 'KeyB', alt: true },
    run: layout.toggleSessions,
  },
  {
    id: 'panel-sidebar',
    label: 'Left panel: queue and notes',
    group: 'Navigate',
    shortcut: { key: 'a', unlessTyping: true },
    run() {
      layout.showSidebarTab(a.sidebarTab)
      a.focus(a.sidebarTab === 'queue' ? 'queue-filter' : 'sidebar')
    },
  },
  {
    id: 'panel-terminal',
    label: 'Middle panel: the active session',
    group: 'Navigate',
    run() {
      layout.closeCentre()
      layout.focusPanel('terminal')
    },
  },
  {
    id: 'panel-sessions',
    label: 'Right panel: sessions',
    group: 'Navigate',
    shortcut: { key: 'd' },
    run: () => layout.focusPanel('sessions'),
  },
  {
    id: 'previous-session',
    label: 'Previous session',
    group: 'Navigate',
    shortcut: { key: '[' },
    run: () => a.stepSession(-1),
  },
  {
    id: 'next-session',
    label: 'Next session',
    group: 'Navigate',
    shortcut: { key: ']' },
    run: () => a.stepSession(1),
  },
  {
    id: 'palette',
    label: 'Command palette',
    group: 'Navigate',
    shortcut: { key: 'k' },
    palette: false,
    run: () => a.setOverlay(a.overlay === 'palette' ? null : 'palette'),
  },
]
