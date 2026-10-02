import { useEffect, useRef } from 'react'

type InlineInputProps = {
  initial?: string
  placeholder?: string
  blur: 'submit' | 'cancel'
  className?: string
  onSubmit(value: string): void
  onCancel(): void
}

export function InlineInput({ initial = '', placeholder, blur, className = '', onSubmit, onCancel }: InlineInputProps) {
  const input = useRef<HTMLInputElement>(null)
  const done = useRef(false)

  useEffect(() => {
    input.current?.focus()
    input.current?.select()
  }, [])

  const finish = (submit: boolean) => {
    if (done.current) return
    done.current = true
    if (submit) onSubmit(input.current?.value ?? '')
    else onCancel()
  }

  return (
    <input
      ref={input}
      defaultValue={initial}
      placeholder={placeholder}
      spellCheck={false}
      className={`field ${className}`}
      onKeyDown={(e) => {
        if (e.key === 'Enter') finish(true)
        if (e.key === 'Escape') finish(false)
      }}
      onBlur={() => finish(blur === 'submit')}
    />
  )
}
