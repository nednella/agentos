import { useEffect, useRef } from 'react'
import type { RefObject } from 'react'
import { useAgentos } from './AgentosContext'
import type { FocusTarget } from './AgentosContext'

export function useFocusRequest(target: FocusTarget, ref: RefObject<HTMLElement>, ready = true) {
  const { focusRequest } = useAgentos()
  const handled = useRef(focusRequest.n)

  useEffect(() => {
    if (!ready || focusRequest.n === handled.current) return
    handled.current = focusRequest.n
    if (focusRequest.target === target) ref.current?.focus()
  }, [focusRequest, ready, ref, target])
}
