type NoticeProps = { title: string; hint: string }

export function Notice({ title, hint }: NoticeProps) {
  return (
    <div className="flex flex-col gap-1 px-4 py-8 text-center">
      <p className="text-body font-medium">{title}</p>
      <p className="text-small text-dim">{hint}</p>
    </div>
  )
}
