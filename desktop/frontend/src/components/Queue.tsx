import { useEffect, useMemo, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import { filterIssues, isFiltering } from '../issueFilter'
import { readStored, writeStored } from '../storage'
import type { Issue } from '../types'
import { useListNav } from '../useListNav'
import { useScrollCursorIntoView } from '../useScrollCursorIntoView'
import { Collapse } from './Collapse'
import { IssueRow } from './IssueRow'
import { Notice } from './Notice'
import { QueueSectionHeader } from './QueueSectionHeader'
import { QueueSearch } from './QueueSearch'
import type { NavRef } from './Sidebar'

type Item = { kind: 'section'; name: string; count: number } | { kind: 'issue'; issue: Issue }

const itemKey = (item: Item) => (item.kind === 'section' ? `section:${item.name}` : `issue:${item.issue.number}`)

const CLOSED_KEY = 'agentos.queue.closed'

type QueueProps = { nav: NavRef }

export function Queue({ nav }: QueueProps) {
  const { project, issues, issuesDisabled, issueFilter, report } = useAgentos()
  if (project && !project.repo) {
    return <Notice title="No repo linked" hint={`${project.name} has no GitHub repo, so it has no queue.`} centered />
  }
  if (project && issuesDisabled) {
    const repo = (
      <button className="link" onClick={() => report(() => api.openURL(`https://github.com/${project.repo}`))}>
        {project.repo}
      </button>
    )
    return <Notice title="Issues are disabled" hint={<>{repo} has issues disabled, enable them to use the queue feature</>} centered />
  }
  const filtering = isFiltering(issueFilter)
  const shown = filtering ? filterIssues(issues, issueFilter) : issues

  return (
    <>
      <QueueSearch shown={filtering ? shown.length : null} />
      <QueueList nav={nav} shown={shown} filtering={filtering} />
    </>
  )
}

type QueueListProps = { nav: NavRef; shown: Issue[]; filtering: boolean }

function QueueList({ nav, shown, filtering }: QueueListProps) {
  const { issues, issuesLoading, setOverlay } = useAgentos()
  const [closed, setClosed] = useState<Set<string>>(() => new Set(readStored<string[]>(CLOSED_KEY, [])))
  const list = useRef<HTMLDivElement>(null)

  // The core lists issues section by section; a project with no sections has one nameless group.
  const sections = useMemo(() => {
    const groups: { name: string; issues: Issue[]; open: boolean }[] = []
    for (const issue of shown) {
      const group = groups.find((g) => g.name === issue.section)
      if (group) group.issues.push(issue)
      else groups.push({ name: issue.section, issues: [issue], open: filtering || !closed.has(issue.section) })
    }
    return groups
  }, [shown, filtering, closed])

  const items = useMemo<Item[]>(
    () =>
      sections.flatMap((section) => [
        ...(section.name ? [{ kind: 'section' as const, name: section.name, count: section.issues.length }] : []),
        ...(section.open ? section.issues.map((issue) => ({ kind: 'issue' as const, issue })) : []),
      ]),
    [sections],
  )

  const toggle = (name: string, open: boolean) =>
    setClosed((set) => {
      const next = new Set(set)
      if (open) next.delete(name)
      else next.add(name)
      writeStored(CLOSED_KEY, [...next])
      return next
    })

  const listNav = useListNav(items.map(itemKey), {
    onEnter(index) {
      const item = items[index]
      if (!item) return
      if (item.kind === 'section') {
        toggle(item.name, closed.has(item.name))
        return
      }
      setOverlay({ issue: item.issue.number })
    },
    onSpace(index) {
      const item = items[index]
      if (item?.kind !== 'section' || filtering) return false
      toggle(item.name, closed.has(item.name))
      return true
    },
    onLeft(index) {
      const item = items[index]
      if (item?.kind !== 'section' || filtering || closed.has(item.name)) return false
      toggle(item.name, false)
      return true
    },
    onRight(index) {
      const item = items[index]
      if (item?.kind !== 'section' || filtering || !closed.has(item.name)) return false
      toggle(item.name, true)
      return true
    },
  })

  useEffect(() => {
    nav.current = listNav.handle
    return () => {
      nav.current = null
    }
  }, [nav, listNav.handle])

  useScrollCursorIntoView(list, listNav.cursorKey)

  if (issuesLoading && issues.length === 0) return <Notice title="Loading issues" hint="Asking GitHub." centered />
  if (issues.length === 0) return <Notice title="No open issues" hint="This repo has nothing in the queue." centered />
  if (filtering && shown.length === 0) return <Notice title="No issues match" hint="Clear the search to see all of them." centered />

  return (
    <div ref={list} className="min-h-0 flex-1 overflow-y-auto pt-1.5 pb-2">
      {sections.map((section) => (
        <section key={section.name}>
          {section.name && (
            <QueueSectionHeader
              name={section.name}
              count={section.issues.length}
              open={section.open}
              cursor={listNav.cursorKey === `section:${section.name}`}
              locked={filtering}
              onToggle={() => toggle(section.name, closed.has(section.name))}
            />
          )}
          <Collapse open={section.open}>
            {section.issues.map((issue) => (
              <IssueRow key={issue.number} issue={issue} cursor={section.open && listNav.cursorKey === `issue:${issue.number}`} />
            ))}
          </Collapse>
        </section>
      ))}
    </div>
  )
}
