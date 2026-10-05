import { useEffect } from 'react'
import type { RefObject } from 'react'

export function useScrollCursorIntoView(list: RefObject<HTMLElement>, cursor: unknown) {
  useEffect(() => {
    list.current?.querySelector('[data-cursor="true"]')?.scrollIntoView({ block: 'nearest' })
  }, [list, cursor])
}
