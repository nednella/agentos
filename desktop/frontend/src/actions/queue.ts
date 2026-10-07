import { isFiltering } from '../issueFilter'
import type { Action, ActionContext } from './types'

export const queueActions = ({ a, layout }: ActionContext): Action[] => [
  {
    id: 'start-issue',
    label: 'Start issue',
    group: 'Queue',
    palette: false,
    async run(args) {
      const numbers = args.map((x) => Number(x.replace(/^#/, '')))
      if (numbers.length === 0 || numbers.some((n) => !Number.isInteger(n))) throw 'Give issue numbers: issue 394 393'
      for (const [i, number] of numbers.entries()) {
        try {
          await a.startIssue(number, '', i < numbers.length - 1)
        } catch (err) {
          throw `Issue #${number}: ${typeof err === 'string' ? err : 'could not start'}`
        }
      }
      return `Started ${numbers.map((n) => `#${n}`).join(', ')}`
    },
  },
  {
    id: 'show-queue',
    label: 'Search the queue',
    group: 'Queue',
    uiCommand: 'queue',
    run() {
      layout.showSidebarTab('queue')
      a.focus('queue-filter')
    },
  },
  {
    id: 'filter-queue',
    label: 'Filter the queue…',
    group: 'Queue',
    palette: { prompt: 'Filter, e.g. @alex-r label:idea -type:bug', initial: a.issueFilter },
    uiCommand: 'filter',
    run(args) {
      layout.showSidebarTab('queue')
      a.setIssueFilter(args.join(' '))
      return isFiltering(args.join(' ')) ? 'Filter set' : 'Filter cleared'
    },
  },
  {
    id: 'refresh-issues',
    label: 'Refresh issues and PRs',
    group: 'Queue',
    shortcut: { key: 'r' },
    uiCommand: 'refresh',
    async run() {
      await Promise.all([a.refreshIssues(), a.refreshPRs()])
      return 'Issues and pull requests refreshed'
    },
  },
]
