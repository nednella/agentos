import { useEffect, useRef } from 'react'
import type { Ref, TextareaHTMLAttributes } from 'react'

type AutoTextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & { taRef?: Ref<HTMLTextAreaElement> }

const MAX_HEIGHT_REM = 16

function grow(el: HTMLTextAreaElement) {
  el.style.height = 'auto'
  const max = MAX_HEIGHT_REM * parseFloat(getComputedStyle(document.documentElement).fontSize)
  el.style.height = `${Math.min(el.scrollHeight + 2, max)}px`
}

export function AutoTextarea({ taRef, onInput, className = '', ...props }: AutoTextareaProps) {
  const inner = useRef<HTMLTextAreaElement | null>(null)

  useEffect(() => {
    if (inner.current) grow(inner.current)
  }, [props.value])

  return (
    <textarea
      {...props}
      ref={(el) => {
        inner.current = el
        if (typeof taRef === 'function') taRef(el)
        else if (taRef) (taRef as { current: HTMLTextAreaElement | null }).current = el
      }}
      rows={1}
      className={`w-full resize-none overflow-y-auto rounded-md border bg-transparent px-2.5 outline-none ${className}`}
      onInput={(e) => {
        grow(e.currentTarget)
        onInput?.(e)
      }}
    />
  )
}

export { grow as growTextarea }
