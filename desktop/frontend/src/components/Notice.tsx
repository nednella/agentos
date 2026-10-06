import type { ReactNode } from 'react'

type NoticeProps = { title: string; hint: ReactNode; centered?: boolean }

export function Notice({ title, hint, centered = false }: NoticeProps) {
  return (
    <div className={`flex flex-col gap-1 px-4 py-8 text-center ${centered ? 'flex-1 justify-center' : ''}`}>
      <p className="text-body font-medium">{title}</p>
      <p className="text-small text-dim">{hint}</p>
    </div>
  )
}
