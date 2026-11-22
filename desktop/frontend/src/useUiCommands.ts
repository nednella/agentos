import { useEffect, useRef } from 'react'
import { useAgentos, useLatestAgentos } from './AgentosContext'
import { actionsFor } from './actions'
import { on } from './api'
import { useLayout } from './LayoutContext'

export function useUiCommands() {
  const { report } = useAgentos('report')
  const latestAgentos = useLatestAgentos()
  const layout = useLayout()
  const latestLayout = useRef(layout)
  latestLayout.current = layout

  useEffect(
    () =>
      on('ui:command', ({ name, args }) =>
        report(() => {
          const action = actionsFor(latestAgentos(), latestLayout.current).find((a) => a.uiCommand === name)
          if (!action) throw `agentos ${name}: the app has no such command`
          return action.run(args)
        }),
      ),
    [report, latestAgentos],
  )
}
