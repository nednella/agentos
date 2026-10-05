import { useEffect, useRef } from 'react'
import { useAgentos } from './AgentosContext'
import { useActions } from './actions'
import { on } from './api'

export function useUiCommands() {
  const { report } = useAgentos()
  const actions = useActions()
  const latest = useRef(actions)
  latest.current = actions

  useEffect(
    () =>
      on('ui:command', ({ name, args }) =>
        report(() => {
          const action = latest.current.find((a) => a.uiCommand === name)
          if (!action) throw `agentos ${name}: the app has no such command`
          return action.run(args)
        }),
      ),
    [report],
  )
}
