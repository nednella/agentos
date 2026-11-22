import { useAgentos } from '../AgentosContext'
import type { Agentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import type { Layout } from '../LayoutContext'
import { noteActions } from './notes'
import { navigateActions } from './navigate'
import { queueActions } from './queue'
import { sessionActions } from './sessions'
import { shellActions } from './shell'
import type { Action } from './types'
import { viewActions } from './views'

export type { Action, ActionGroup, Shortcut } from './types'
export { formatShortcut, LIST_KEYS, matchShortcut } from './shortcut'

export function useActions(): Action[] {
  return actionsFor(useAgentos(), useLayout())
}

export function actionsFor(a: Agentos, layout: Layout): Action[] {
  const context = { a, layout, current: a.sessions.find((s) => s.id === a.selectedId) }
  return [
    ...sessionActions(context),
    ...navigateActions(context),
    ...queueActions(context),
    ...noteActions(context),
    ...viewActions(context),
    ...shellActions(context),
  ]
}
