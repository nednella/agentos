import type { Issue, IssueDetail } from './types'

type Lane = 'ready' | 'plan' | 'you' | 'idea' | 'inbox'
type Seed = [number, Lane, Issue['type'], string, string, string[], string[]?]

const day = 86_400_000

const livedocumentSeeds: Seed[] = [
  [454, 'ready', 'refactor', 'Split the document entry into edit and view', 'nednella', ['ned']],
  [444, 'ready', 'refactor', 'Restructure the embed route and providers', 'nednella', ['ned']],
  [326, 'ready', 'bug', 'sendBeacon drops the last view duration', 'mariam-k', [], ['model:haiku']],
  [412, 'ready', 'chore', 'Sweep the locale keys for the billing pages', 'tomasz', ['tomasz']],
  [389, 'ready', 'feature', 'Stripe webhook retries with backoff', 'nednella', []],
  [401, 'ready', 'bug', 'Password prompt flashes before the document loads', 'mariam-k', ['mariam-k'], ['effort:high']],
  [405, 'plan', 'feature', 'Per-link expiry with a grace period', 'dev-bot', []],
  [398, 'plan', 'feature', 'Lead capture form on the viewer', 'tomasz', ['nednella']],
  [372, 'plan', 'refactor', 'One plan catalogue for every billing surface', 'nednella', []],
  [361, 'plan', 'bug', 'Embed ignores the allowlist on first paint', 'mariam-k', [], ['model:opus', 'effort:max']],
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

const labelFor = (lane: Lane) => ({ ready: 'ready', plan: 'needs-plan', you: 'needs-human', idea: 'idea', inbox: '' })[lane]

// The queue sections of the mock project; an issue no section takes goes in Other, as in the core.
const sections: Record<Lane, { name: string; actions: string[] }> = {
  inbox: { name: 'Inbox', actions: ['Plan', 'Investigate', 'Work'] },
  ready: { name: 'Ready', actions: ['Work'] },
  plan: { name: 'Needs plan', actions: ['Investigate'] },
  you: { name: 'Other', actions: ['Start'] },
  idea: { name: 'Other', actions: ['Start'] },
}
const sectionOrder = ['Inbox', 'Ready', 'Needs plan', 'Other']

export function buildIssues(repo: string): Issue[] {
  const now = Date.now()
  const issues = (seedsByRepo[repo] ?? []).map(([number, lane, type, title, author, assignees, extra = []], i) => {
    const label = labelFor(lane)
    return {
      number,
      title,
      type,
      section: sections[lane].name,
      actions: sections[lane].actions,
      url: `https://github.com/${repo}/issues/${number}`,
      sessionId: '',
      author,
      assignees,
      labels: [label, type ? `type:${type}` : '', ...extra].filter(Boolean),
      createdAt: now - (i + 2) * day,
      updatedAt: now - (i % 5) * day,
    }
  })
  return issues.sort((a, b) => sectionOrder.indexOf(a.section) - sectionOrder.indexOf(b.section))
}

const html = (parts: string[]) => parts.join('\n')

const bodies: Record<number, string> = {
  454: html([
    '<h2 dir="auto">Description</h2>',
    '<p dir="auto">The document entry mixes the edit and view paths. Split it so each path loads only what it needs.</p>',
    '<ul dir="auto"><li>Move the viewer into <code>document/view.tsx</code></li><li class="task-list-item"><input type="checkbox" disabled> Keep the URL the same</li><li class="task-list-item"><input type="checkbox" checked disabled> Measure the bundle before and after</li></ul>',
    '<p dir="auto">See <a href="https://github.com/upscopeio/livedocument/issues/444">#444</a> for the embed side of this.</p>',
    '<div class="highlight highlight-source-ts"><pre>export const entry = lazy(() =&gt; import(\'./view\'))</pre></div>',
  ]),
  12: html([
    '<h2 dir="auto">Description</h2>',
    '<p dir="auto">Coming back to a project should land on the session I left, not the first one in the list.</p>',
    '<blockquote><p dir="auto">The app already remembers the last project; do the same per project for the session.</p></blockquote>',
  ]),
}

export function buildIssueDetail(number: number): IssueDetail {
  const now = Date.now()
  const comments = number === 454 ? [{ author: 'mariam-k', createdAt: now - 2 * day, bodyHTML: '<p dir="auto">The embed route depends on this, so land it first.</p>' }] : []
  return { number, bodyHTML: bodies[number] ?? '', comments }
}
