import { useEffect, useRef } from 'react'
import type { RefObject } from 'react'
import { useAgentos } from './AgentosContext'
import type { FocusTarget } from './AgentosContext'

// Acts once on each focus request for `target`, while `ready`. A request made before the
// component mounted still counts, so a panel that opens because of it can take the focus.
export function useFocusRequest(target: FocusTarget, focusOn: RefObject<HTMLElement> | (() => void), ready = true) {
  const { focusRequest } = useAgentos()
  const handled = useRef(-1)

  useEffect(() => {
    if (!ready || focusRequest.n === handled.current) return
    handled.current = focusRequest.n
    if (focusRequest.target !== target) return
    if (typeof focusOn === 'function') focusOn()
    else focusOn.current?.focus()
  }, [focusRequest, ready, focusOn, target])
}
