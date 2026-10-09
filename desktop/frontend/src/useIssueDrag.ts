import { useRef, useState } from 'react'
import { useAgentos } from './AgentosContext'
import { api } from './api'
import type { Issue } from './types'

export type IssueDrag = {
  dragging: Issue | null
  start(issue: Issue): void
  end(): void
  drop(section: string): void
  place(list: Issue[]): Issue[]
}

// Where the queue shows an issue until the core's own list agrees: `settled` is the list that was
// current when the call finished, so the next list to arrive replaces the override.
type Placement = { section: string; settled: Issue[] | null }

export function useIssueDrag(issues: Issue[]): IssueDrag {
  const { report, pushToast, dismissToast } = useAgentos()
  const [dragging, setDragging] = useState<Issue | null>(null)
  const [placements, setPlacements] = useState<Map<number, Placement>>(new Map())
  const latest = useRef(issues)
  latest.current = issues

  const setPlacement = (number: number, placement: Placement | null) =>
    setPlacements((map) => {
      const next = new Map(map)
      if (placement) next.set(number, placement)
      else next.delete(number)
      return next
    })

  async function relocate(number: number, section: string, call: () => Promise<void>) {
    setPlacement(number, { section, settled: null })
    try {
      await call()
    } catch (err) {
      setPlacement(number, null)
      throw err
    }
    setPlacement(number, { section, settled: latest.current })
  }

  async function move(issue: Issue, section: string) {
    await relocate(issue.number, section, () => api.moveIssue(issue.number, section))
    const key = pushToast({
      tone: 'info',
      text: `#${issue.number} moved to ${section}`,
      action: {
        label: 'Undo',
        run() {
          dismissToast(key)
          report(() => relocate(issue.number, issue.section, () => api.undoMove(issue.number)))
        },
      },
    })
  }

  return {
    dragging,
    start: setDragging,
    end: () => setDragging(null),
    drop(section) {
      if (!dragging?.moves.includes(section)) return
      const issue = dragging
      setDragging(null)
      report(() => move(issue, section))
    },
    place(list) {
      const pending = list.filter((issue) => {
        const placement = placements.get(issue.number)
        return placement && placement.section !== issue.section && (!placement.settled || placement.settled === latest.current)
      })
      if (pending.length === 0) return list
      const moved = list.map((issue) => (pending.includes(issue) ? { ...issue, section: placements.get(issue.number)!.section } : issue))
      const rank = ({ section }: Issue) => {
        if (!section) return list.length + 1
        const at = list.findIndex((i) => i.section === section)
        return at < 0 ? list.length : at
      }
      return moved.map((issue, at) => ({ issue, at })).sort((a, b) => rank(a.issue) - rank(b.issue) || a.at - b.at).map((x) => x.issue)
    },
  }
}
