import type { ReactNode } from 'react'

type CollapseProps = { open: boolean; children: ReactNode }

export function Collapse({ open, children }: CollapseProps) {
  const closed = open ? {} : { inert: '' }
  return (
    <div className="fold" data-open={open}>
      <div {...closed}>{children}</div>
    </div>
  )
}
