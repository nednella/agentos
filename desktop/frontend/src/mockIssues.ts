import type { Issue } from './types'

type Seed = [number, Issue['lane'], Issue['type'], string, string, string[]]

const day = 86_400_000

const livedocumentSeeds: Seed[] = [
  [454, 'ready', 'refactor', 'Split the document entry into edit and view', 'nednella', ['ned']],
  [444, 'ready', 'refactor', 'Restructure the embed route and providers', 'nednella', ['ned']],
  [326, 'ready', 'bug', 'sendBeacon drops the last view duration', 'mariam-k', []],
  [412, 'ready', 'chore', 'Sweep the locale keys for the billing pages', 'tomasz', ['tomasz']],
  [389, 'ready', 'feature', 'Stripe webhook retries with backoff', 'nednella', []],
  [401, 'ready', 'bug', 'Password prompt flashes before the document loads', 'mariam-k', ['mariam-k']],
  [405, 'plan', 'feature', 'Per-link expiry with a grace period', 'dev-bot', []],
  [398, 'plan', 'feature', 'Lead capture form on the viewer', 'tomasz', ['nednella']],
  [372, 'plan', 'refactor', 'One plan catalogue for every billing surface', 'nednella', []],
  [361, 'plan', 'bug', 'Embed ignores the allowlist on first paint', 'mariam-k', []],
  [418, 'you', 'chore', 'Decide the retention window for viewer events', 'tomasz', ['nednella']],
  [415, 'you', 'feature', 'Pick the default sort for the library', 'mariam-k', ['nednella']],
  [377, 'you', 'bug', 'Duplicate contacts after a Google import', 'dev-bot', []],
  [430, 'inbox', 'bug', 'Thumbnail aspect ratio jumps on resize', 'dev-bot', []],
  [431, 'inbox', '', 'Link analytics export as CSV', 'tomasz', []],
  [433, 'inbox', 'bug', 'Toast stays after the page changes', 'mariam-k', []],
  [436, 'inbox', 'feature', 'Dark mode for the public viewer', 'tomasz', []],
  [437, 'inbox', 'chore', 'Drop the cobrowse remnants in api/', 'nednella', []],
  [438, 'inbox', 'bug', 'Rename on a template does not refresh the list', 'dev-bot', []],
  [350, 'idea', 'feature', 'Heat map of where viewers stop reading', 'tomasz', []],
  [351, 'idea', 'feature', 'Slack ping when a lead opens a link twice', 'mariam-k', []],
  [352, 'idea', '', 'Presenter mode with a laser pointer', 'nednella', []],
  [353, 'idea', 'feature', 'Auto-summary of a document for the link preview', 'tomasz', []],
  [354, 'idea', 'refactor', 'Move the PDF worker to its own service', 'nednella', []],
  [355, 'idea', '', 'Team leaderboard for most viewed link', 'mariam-k', []],
]

const agentosSeeds: Seed[] = [
  [12, 'ready', 'feature', 'Persist the last selected session per project', 'nednella', []],
  [11, 'ready', 'bug', 'Terminal loses focus after a rename', 'nednella', []],
  [9, 'plan', 'feature', 'Notes sync between machines', 'nednella', []],
  [7, 'inbox', 'chore', 'Bundle only the latin font subsets', 'dev-bot', []],
  [5, 'idea', 'feature', 'Voice note capture', 'nednella', []],
]

const seedsByRepo: Record<string, Seed[]> = {
  'upscopeio/livedocument': livedocumentSeeds,
  'nednella/agentos': agentosSeeds,
}

const labelFor = (lane: Issue['lane']) =>
  ({ ready: 'ready', plan: 'needs-plan', you: 'needs-human', idea: 'idea', inbox: '' })[lane]

export function buildIssues(repo: string): Issue[] {
  const now = Date.now()
  return (seedsByRepo[repo] ?? []).map(([number, lane, type, title, author, assignees], i) => {
    const label = labelFor(lane)
    return {
      number,
      title,
      type,
      lane,
      url: `https://github.com/${repo}/issues/${number}`,
      sessionId: '',
      author,
      assignees,
      labels: [label, type ? `type:${type}` : ''].filter(Boolean),
      createdAt: now - (i + 2) * day,
      updatedAt: now - (i % 5) * day,
    }
  })
}
