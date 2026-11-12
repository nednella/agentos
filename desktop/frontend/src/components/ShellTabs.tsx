import { useCallback, useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { Icon } from './Icon'
import { ShellTab, tabRowButton } from './ShellTab'

export function ShellTabs() {
  const { project, shellIds, shellId, selectShell, newShell, closeShell, report } = useAgentos()
  const { mode, shellOpen, setShellOpen, focusPanel } = useLayout()
  const scroller = useRef<HTMLDivElement>(null)
  const [view, setView] = useState({ start: 0, end: 1 })
  const [projectShell = '', ...shells] = shellIds

  const measure = useCallback(() => {
    const { scrollLeft, clientWidth, scrollWidth } = scroller.current!
    setView({
      start: scrollLeft > 1 ? scrollLeft / scrollWidth : 0,
      end: scrollLeft + clientWidth < scrollWidth - 1 ? (scrollLeft + clientWidth) / scrollWidth : 1,
    })
  }, [])

  useEffect(() => {
    const el = scroller.current!
    const onWheel = (e: WheelEvent) => {
      if (Math.abs(e.deltaY) <= Math.abs(e.deltaX)) return
      e.preventDefault()
      el.scrollLeft += e.deltaY
    }
    const resize = new ResizeObserver(measure)
    resize.observe(el)
    el.addEventListener('wheel', onWheel, { passive: false })
    return () => {
      resize.disconnect()
      el.removeEventListener('wheel', onWheel)
    }
  }, [measure])

  useEffect(() => {
    scroller.current!.querySelector('[data-selected="true"]')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
    measure()
  }, [shellIds, shellId, measure])

  const pick = (id: string) => {
    selectShell(id)
    focusPanel('shell')
  }

  return (
    <div
      className={`shell-tabs group/tabs flex h-10 flex-none items-stretch border-b text-small ${shellOpen ? 'border-line' : 'border-transparent'}`}
      style={{ paddingRight: 'var(--corner-pad)' }}
    >
      <div role="tablist" aria-label="Shells" className="flex min-w-0 items-stretch">
        <ShellTab label={project?.name ?? ''} pinned selected={shellId === projectShell} onSelect={() => pick(projectShell)} />
        <div className="relative flex min-w-0">
          <div
            ref={scroller}
            className="shell-tabs-scroll flex min-w-0 items-stretch overflow-x-auto"
            data-hidden-left={view.start > 0}
            data-hidden-right={view.end < 1}
            onScroll={measure}
          >
            {shells.map((id, i) => (
              <ShellTab
                key={id}
                label={`Shell ${i + 2}`}
                selected={id === shellId}
                onSelect={() => pick(id)}
                onClose={() => report(() => closeShell(id))}
              />
            ))}
          </div>
          {(view.start > 0 || view.end < 1) && (
            <span
              className="pointer-events-none absolute -bottom-px h-px bg-dim transition-colors group-hover/tabs:bg-soft"
              style={{ left: `${view.start * 100}%`, width: `${(view.end - view.start) * 100}%` }}
            />
          )}
        </div>
      </div>
      <button
        className={`${tabRowButton} ml-1.5 self-center`}
        title="New shell"
        aria-label="New shell"
        onClick={() =>
          report(async () => {
            await newShell()
            focusPanel('shell')
          })
        }
      >
        <Icon name="add" size={14} />
      </button>
      {mode !== 'narrow' && (
        <button
          className={`${tabRowButton} ml-auto self-center`}
          title={shellOpen ? 'Collapse the shell (⌘S)' : 'Open the shell (⌘S)'}
          aria-label={shellOpen ? 'Collapse the shell' : 'Open the shell'}
          aria-expanded={shellOpen}
          onClick={() => setShellOpen(!shellOpen)}
        >
          <span className={`transition-transform duration-150 ${shellOpen ? '' : 'rotate-180'}`}>
            <Icon name="chevron-down" size={14} />
          </span>
        </button>
      )}
    </div>
  )
}
