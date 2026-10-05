import { useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useActions } from '../actions'
import { useLayout } from '../LayoutContext'
import { commandNamed, commonPrefix, completionsFor, didYouMean, ghostFor, parseInput } from '../commandLine'
import type { Candidate } from '../actions'
import { devFlags, errorMessage } from '../api'
import { readStored, writeStored } from '../storage'
import { Keycap } from './Keycap'

type Message = { tone: 'ok' | 'error'; text: string } | null
type Popup = { start: number; items: Candidate[]; cursor: number } | null

const HISTORY_LIMIT = 100

export function CommandLine() {
  const { project, focusRequest, focus } = useAgentos()
  const actions = useActions()
  const { mode } = useLayout()
  const input = useRef<HTMLInputElement>(null)
  const [value, setValue] = useState(devFlags.cmd ?? '')
  const [message, setMessage] = useState<Message>(null)
  const [popup, setPopup] = useState<Popup>(null)
  const [busy, setBusy] = useState(false)
  const historyAt = useRef(-1)
  const draft = useRef('')

  const historyKey = `agentos.history.${project?.name ?? ''}`
  const historyRef = useRef<string[]>([])

  useEffect(() => {
    historyRef.current = readStored<string[]>(historyKey, [])
  }, [historyKey])

  useEffect(() => {
    if (devFlags.cmd !== undefined) input.current?.focus()
  }, [])

  useEffect(() => {
    if (focusRequest.target !== 'command') return
    input.current?.focus()
    input.current?.select()
  }, [focusRequest])

  const edit = (next: string) => {
    setValue(next)
    setPopup(null)
    setMessage(null)
    historyAt.current = -1
  }

  const remember = (line: string) => {
    if (historyRef.current[0] === line) return
    const next = [line, ...historyRef.current].slice(0, HISTORY_LIMIT)
    historyRef.current = next
    writeStored(historyKey, next)
  }

  const execute = async () => {
    const line = value.trim()
    if (!line || busy) return
    remember(line)
    historyAt.current = -1
    const { name, args } = parseInput(line)
    const action = commandNamed(actions, name)
    if (!action) {
      setMessage({ tone: 'error', text: `Unknown command "${name}".${didYouMean(actions, name)} Type help for the list.` })
      return
    }
    setBusy(true)
    try {
      const result = await action.run(args, 'command')
      setValue('')
      setMessage(result ? { tone: 'ok', text: result } : null)
    } catch (err) {
      setMessage({ tone: 'error', text: `${action.command?.syntax}: ${errorMessage(err)}` })
    } finally {
      setBusy(false)
    }
  }

  const complete = (backwards: boolean) => {
    if (popup) {
      const step = backwards ? -1 : 1
      setPopup({ ...popup, cursor: (popup.cursor + step + popup.items.length) % popup.items.length })
      return
    }
    const { start, typed, items } = completionsFor(value, actions)
    if (items.length === 0) {
      setMessage({ tone: 'error', text: 'Nothing to complete here.' })
      return
    }
    if (items.length === 1) {
      edit(`${value.slice(0, start)}${items[0].value} `)
      return
    }
    const prefix = commonPrefix(items.map((i) => i.value))
    if (prefix.length > typed.length) setValue(`${value.slice(0, start)}${prefix}`)
    setPopup({ start, items, cursor: 0 })
  }

  const accept = (candidate: Candidate, start: number) => {
    edit(`${value.slice(0, start)}${candidate.value} `)
    input.current?.focus()
  }

  const walkHistory = (delta: number) => {
    const list = historyRef.current
    if (historyAt.current === -1) draft.current = value
    const next = Math.min(Math.max(historyAt.current + delta, -1), list.length - 1)
    if (next === historyAt.current) return
    historyAt.current = next
    setValue(next === -1 ? draft.current : list[next])
    setMessage(null)
  }

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Tab') {
      e.preventDefault()
      complete(e.shiftKey)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (popup) accept(popup.items[popup.cursor], popup.start)
      else void execute()
    } else if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
      e.preventDefault()
      const delta = e.key === 'ArrowUp' ? 1 : -1
      if (popup) setPopup({ ...popup, cursor: (popup.cursor - delta + popup.items.length) % popup.items.length })
      else walkHistory(delta)
    } else if (e.key === 'Escape') {
      e.stopPropagation()
      if (popup) setPopup(null)
      else {
        input.current?.blur()
        focus('terminal')
      }
    }
  }

  const ghost = ghostFor(value, actions)
  const typedAction = commandNamed(actions, parseInput(value).name)

  return (
    <div data-panel="command" className="flex-none border-t border-line bg-surface">
      <div className="relative flex h-10 items-center gap-2 px-4">
        <span className="mono max-w-[8rem] flex-none truncate text-small text-accent">{project?.name ?? ''} ›</span>
        <div className="relative h-full min-w-0 flex-1">
          <div aria-hidden="true" className="mono pointer-events-none absolute inset-0 flex items-center overflow-hidden text-body whitespace-pre">
            <span className="invisible">{value}</span>
            <span className="text-dim">{ghost}</span>
          </div>
          <input
            ref={input}
            value={value}
            spellCheck={false}
            autoComplete="off"
            aria-label="Command line"
            placeholder={mode === 'narrow' ? 'Command, e.g. issue 394' : 'Type a command: new, issue 394, project, note, filter, stats, help'}
            className="mono relative h-full w-full bg-transparent text-body outline-none placeholder:text-dim"
            onChange={(e) => edit(e.target.value)}
            onKeyDown={onKeyDown}
          />
        </div>
        {mode !== 'narrow' && <Keycap>⌘S</Keycap>}
        {popup && (
          <div
            className="fade-in absolute bottom-10 left-4 max-h-60 w-[min(36rem,calc(100%-2rem))] overflow-y-auto rounded-md border border-line-strong bg-raised py-1 shadow-2xl"
            style={{ zIndex: 'var(--z-popup)' }}
            role="listbox"
          >
            {popup.items.map((item, i) => (
              <button
                key={item.value}
                role="option"
                aria-selected={i === popup.cursor}
                className="row h-7 items-center gap-3 px-3"
                data-selected={i === popup.cursor}
                onMouseDown={(e) => {
                  e.preventDefault()
                  accept(item, popup.start)
                }}
              >
                <span className="mono text-small text-accent">{item.value}</span>
                <span className="truncate text-small">{item.label === item.value ? '' : item.label}</span>
                <span className="mono ml-auto truncate text-label text-dim">{item.detail}</span>
              </button>
            ))}
          </div>
        )}
      </div>
      <div className="flex h-6 items-center gap-3 px-4 text-small short:h-5" role="status">
        {message ? (
          <span style={{ color: message.tone === 'error' ? 'var(--danger)' : 'var(--finished)' }}>
            <span className="label mr-2" style={{ color: 'inherit' }}>
              {message.tone === 'error' ? 'Error' : 'Done'}
            </span>
            {message.text}
          </span>
        ) : (
          <span className="truncate text-dim">
            {typedAction?.command
              ? `${typedAction.command.syntax}: ${typedAction.command.summary}`
              : 'Tab completes, up and down walk the history, Esc returns to the terminal.'}
          </span>
        )}
      </div>
    </div>
  )
}
