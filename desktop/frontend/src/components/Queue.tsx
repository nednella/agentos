import { useEffect, useMemo, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { filterIssues, isFiltering } from '../issueFilter'
import { LANES } from '../lanes'
import type { Issue, Lane } from '../types'
import { useListNav } from '../useListNav'
import { Collapse } from './Collapse'
import { Icon } from './Icon'
import { IssueRow } from './IssueRow'
import { Notice } from './Notice'
import { QueueLaneHeader } from './QueueLaneHeader'
import { QueueSearch } from './QueueSearch'
import type { NavRef } from './Sidebar'

type Item = { kind: 'lane'; lane: Lane; count: number } | { kind: 'issue'; issue: Issue }

type QueueProps = { nav: NavRef }

export function Queue({ nav }: QueueProps) {
  const { project, issues, issuesLoading, issueFilter, refreshIssues, report } = useAgentos()
  if (project && !project.repo) {
    return <Notice title="No repo linked" hint={`${project.name} has no GitHub repo, so it has no queue.`} />
  }
  const filtering = isFiltering(issueFilter)
  const shown = filtering ? filterIssues(issues, issueFilter) : issues

  return (
    <>
      <div className="flex h-9 flex-none items-center gap-2 px-4">
        <span className="mono text-small text-dim">{filtering ? `${shown.length} of ${issues.length}` : `${issues.length} issues`}</span>
        <button
          className="btn btn-ghost ml-auto h-6 w-6 justify-center px-0"
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
      <QueueSearch />
      <QueueList nav={nav} shown={shown} filtering={filtering} />
    </>
  )
}

type QueueListProps = { nav: NavRef; shown: Issue[]; filtering: boolean }

function QueueList({ nav, shown, filtering }: QueueListProps) {
  const { issues, issuesLoading, select, startIssue, focus, report } = useAgentos()
  const [closed, setClosed] = useState<Set<Lane>>(() => new Set<Lane>(['idea']))
  const list = useRef<HTMLDivElement>(null)

  const sections = useMemo(
    () =>
      LANES.flatMap(({ id }) => {
        const inLane = shown.filter((i) => i.lane === id)
        if (filtering && inLane.length === 0) return []
        return [{ lane: id, issues: inLane, open: filtering || !closed.has(id) }]
      }),
    [shown, filtering, closed],
  )

  const items = useMemo<Item[]>(
    () =>
      sections.flatMap((section) => [
        { kind: 'lane' as const, lane: section.lane, count: section.issues.length },
        ...(section.open ? section.issues.map((issue) => ({ kind: 'issue' as const, issue })) : []),
      ]),
    [sections],
  )

  const toggle = (lane: Lane, open: boolean) =>
    setClosed((set) => {
      const next = new Set(set)
      if (open) next.delete(lane)
      else next.add(lane)
      return next
    })

  const listNav = useListNav(items.length, {
    onEnter(index) {
      const item = items[index]
      if (!item) return
      if (item.kind === 'lane') {
        toggle(item.lane, closed.has(item.lane))
        return
      }
      const { issue } = item
      if (issue.sessionId) {
        select(issue.sessionId)
        focus('terminal')
        return
      }
      report(() => startIssue(issue.number))
    },
    onLeft(index) {
      const item = items[index]
      if (item?.kind !== 'lane' || filtering || closed.has(item.lane)) return false
      toggle(item.lane, false)
      return true
    },
    onRight(index) {
      const item = items[index]
      if (item?.kind !== 'lane' || filtering || !closed.has(item.lane)) return false
      toggle(item.lane, true)
      return true
    },
  })

  useEffect(() => {
    nav.current = listNav.handle
    return () => {
      nav.current = null
    }
  }, [nav, listNav.handle])

  useEffect(() => {
    list.current?.querySelector('[data-cursor="true"]')?.scrollIntoView({ block: 'nearest' })
  }, [listNav.cursor])

  if (issuesLoading && issues.length === 0) return <Notice title="Loading issues" hint="Asking GitHub." />
  if (issues.length === 0) return <Notice title="No open issues" hint="This repo has nothing in the queue." />
  if (filtering && shown.length === 0) return <Notice title="No issues match" hint="Clear the search to see all of them." />

  let index = 0
  return (
    <div ref={list} className="min-h-0 flex-1 overflow-y-auto pb-2">
      {sections.map((section) => {
        const headerAt = index++
        const first = index
        if (section.open) index += section.issues.length
        return (
          <section key={section.lane}>
            <QueueLaneHeader
              lane={section.lane}
              count={section.issues.length}
              open={section.open}
              cursor={listNav.cursor === headerAt}
              locked={filtering}
              onToggle={() => toggle(section.lane, closed.has(section.lane))}
            />
            <Collapse open={section.open}>
              {section.issues.map((issue, i) => (
                <IssueRow key={issue.number} issue={issue} cursor={section.open && listNav.cursor === first + i} />
              ))}
            </Collapse>
          </section>
        )
      })}
    </div>
  )
}
