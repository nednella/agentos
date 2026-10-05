import { useEffect, useMemo, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { activeValue, applyCompletion, completionAt, facet, TYPES, withToken } from '../issueFilter'
import { FacetMenu } from './FacetMenu'
import { Icon } from './Icon'

type QueueSearchProps = { shown: number | null }

export function QueueSearch({ shown }: QueueSearchProps) {
  const { issues, issuesLoading, issueFilter, setIssueFilter, focusRequest, refreshIssues, report } = useAgentos()
  const { returnToTerminal } = useLayout()
  const input = useRef<HTMLInputElement>(null)
  const [cursor, setCursor] = useState(0)
  const [dismissed, setDismissed] = useState(false)

  useEffect(() => {
    if (focusRequest.target !== 'queue-filter') return
    input.current?.focus()
    input.current?.select()
  }, [focusRequest])

  const completion = useMemo(() => (dismissed ? null : completionAt(issueFilter, issues)), [dismissed, issueFilter, issues])
  const authors = useMemo(() => facet(issues, (i) => [i.author]), [issues])
  const types = useMemo(
    () => TYPES.map((value) => ({ value, count: issues.filter((i) => i.type === value).length })).filter((f) => f.count > 0),
    [issues],
  )

  const update = (query: string) => {
    setIssueFilter(query)
    setCursor(0)
    setDismissed(false)
  }

  const accept = (value: string) => {
    if (!completion) return
    update(applyCompletion(issueFilter, completion, value))
  }

  return (
    <div className="flex flex-col gap-1.5 border-b border-line px-3 pt-2.5 pb-2">
      <div className="flex items-center gap-1.5">
        <div className="relative min-w-0 flex-1">
          <span className="pointer-events-none absolute inset-y-0 left-2.5 flex items-center text-dim">
            <Icon name="search" size={13} />
          </span>
          <input
            ref={input}
            value={issueFilter}
            spellCheck={false}
            placeholder="Search: text, @author, label:, type:, is:running"
            aria-label="Search issues"
            className="field w-full pr-7 pl-8 text-small"
            onChange={(e) => update(e.target.value)}
            onKeyDown={(e) => {
              if (completion && (e.key === 'Tab' || e.key === 'Enter')) {
                e.preventDefault()
                accept(completion.values[cursor].value)
              } else if (completion && e.key === 'ArrowDown') {
                e.preventDefault()
                setCursor((c) => Math.min(c + 1, completion.values.length - 1))
              } else if (completion && e.key === 'ArrowUp') {
                e.preventDefault()
                setCursor((c) => Math.max(c - 1, 0))
              } else if (e.key === 'Escape') {
                if (completion) setDismissed(true)
                else returnToTerminal()
                e.stopPropagation()
              }
            }}
          />
          {issueFilter && (
            <button
              className="btn btn-ghost absolute top-0.5 right-0.5 h-6 w-6 justify-center px-0"
              aria-label="Clear search"
              title="Clear"
              onClick={() => {
                update('')
                input.current?.focus()
              }}
            >
              <Icon name="close" size={12} />
            </button>
          )}
          {completion && (
            <div
              className="fade-in absolute top-8 right-0 left-0 max-h-52 overflow-y-auto rounded-md border border-line-strong bg-raised py-1"
              style={{ zIndex: 'var(--z-popup)' }}
              role="listbox"
            >
              {completion.values.map((v, i) => (
                <button
                  key={v.value}
                  role="option"
                  aria-selected={i === cursor}
                  className="row h-7 items-center gap-3 px-3"
                  data-selected={i === cursor}
                  onMouseDown={(e) => {
                    e.preventDefault()
                    accept(v.value)
                  }}
                >
                  <span className="truncate">{v.value}</span>
                  <span className="mono ml-auto text-small text-dim">{v.count}</span>
                </button>
              ))}
            </div>
          )}
        </div>
        <button
          className="btn btn-ghost h-[1.875rem] w-[1.875rem] flex-none justify-center px-0"
          title="Refresh issues and PRs (⌘R)"
          aria-label="Refresh issues"
          disabled={issuesLoading}
          onClick={() => report(refreshIssues)}
        >
          <span className={issuesLoading ? 'inline-flex animate-spin' : 'inline-flex'}>
            <Icon name="refresh" size={13} />
          </span>
        </button>
      </div>
      <div className="flex items-center gap-1">
        <FacetMenu label="Author" facets={authors} active={activeValue(issueFilter, 'author')} onPick={(v) => update(withToken(issueFilter, 'author', v))} />
        <FacetMenu label="Type" facets={types} active={activeValue(issueFilter, 'type')} onPick={(v) => update(withToken(issueFilter, 'type', v))} />
        {shown !== null && <span className="mono ml-auto pr-1 text-small text-dim">{`${shown} of ${issues.length}`}</span>}
      </div>
    </div>
  )
}
