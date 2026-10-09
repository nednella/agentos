import type { Issue, IssueDetail } from './types'

type Lane = 'ready' | 'plan' | 'you' | 'idea' | 'inbox'
type Seed = [number, Lane, Issue['type'], string, string, string[], string[]?]

const day = 86_400_000

const storefrontSeeds: Seed[] = [
  [454, 'ready', 'refactor', 'Split the product page into server and client parts', 'nednella', ['ned']],
  [444, 'ready', 'refactor', 'Restructure the checkout route and providers', 'nednella', ['ned']],
  [326, 'ready', 'bug', 'Cart total is a cent off on large orders', 'alex-r', [], ['model:haiku']],
  [412, 'ready', 'chore', 'Sweep the locale keys for the account pages', 'sam-p', ['sam-p']],
  [389, 'ready', 'feature', 'Payment webhook retries with backoff', 'nednella', []],
  [401, 'ready', 'bug', 'Sign-in prompt flashes before the cart loads', 'alex-r', ['alex-r'], ['effort:high']],
  [405, 'plan', 'feature', 'Discount codes with an expiry date', 'dev-bot', []],
  [398, 'plan', 'feature', 'Wishlist on the product page', 'sam-p', ['nednella']],
  [372, 'plan', 'refactor', 'One price table for every checkout surface', 'nednella', []],
  [361, 'plan', 'bug', 'Shipping zones ignore the postcode on first load', 'alex-r', [], ['model:opus', 'effort:max']],
  [418, 'you', 'chore', 'Decide how long to keep abandoned carts', 'sam-p', ['nednella']],
  [415, 'you', 'feature', 'Pick the default sort for the catalogue', 'alex-r', ['nednella']],
  [377, 'you', 'bug', 'Duplicate customers after a CSV import', 'dev-bot', []],
  [430, 'inbox', 'bug', 'Thumbnail aspect ratio jumps on resize', 'dev-bot', []],
  [431, 'inbox', '', 'Order export as CSV', 'sam-p', []],
  [433, 'inbox', 'bug', 'Toast stays after the page changes', 'alex-r', []],
  [436, 'inbox', 'feature', 'Dark mode for the shop', 'sam-p', []],
  [437, 'inbox', 'chore', 'Drop the legacy coupon code in api/', 'nednella', []],
  [438, 'inbox', 'bug', 'Renaming a category does not refresh the list', 'dev-bot', []],
  [350, 'idea', 'feature', 'Heat map of where shoppers stop scrolling', 'sam-p', []],
  [351, 'idea', 'feature', 'Slack ping when a large order lands', 'alex-r', []],
  [352, 'idea', '', 'Gift wrap option at checkout', 'nednella', []],
  [353, 'idea', 'feature', 'Draft product descriptions from the photos', 'sam-p', []],
  [354, 'idea', 'refactor', 'Move the image resizer to its own service', 'nednella', []],
  [355, 'idea', '', 'Team leaderboard for best selling product', 'alex-r', []],
]

const agentosSeeds: Seed[] = [
  [12, 'ready', 'feature', 'Persist the last selected session per project', 'nednella', []],
  [11, 'ready', 'bug', 'Terminal loses focus after a rename', 'nednella', []],
  [9, 'plan', 'feature', 'Notes sync between machines', 'nednella', []],
  [7, 'inbox', 'chore', 'Bundle only the latin font subsets', 'dev-bot', []],
  [5, 'idea', 'feature', 'Voice note capture', 'nednella', []],
]

const seedsByRepo: Record<string, Seed[]> = {
  'acme/storefront': storefrontSeeds,
  'nednella/agentos': agentosSeeds,
}

const labelFor = (lane: Lane) => ({ ready: 'ready', plan: 'needs-plan', you: 'needs-human', idea: 'idea', inbox: '' })[lane]

// The queue sections of the mock project; an issue no section takes goes last with no section, as in the core.
const sections: Record<Lane, { name: string; actions: string[] }> = {
  inbox: { name: 'Inbox', actions: ['Plan', 'Investigate', 'Work'] },
  ready: { name: 'Ready', actions: ['Work'] },
  plan: { name: 'Needs plan', actions: ['Investigate'] },
  you: { name: '', actions: ['Start'] },
  idea: { name: 'Ideas', actions: ['Start'] },
}
const sectionOrder = ['Inbox', 'Ready', 'Needs plan', 'Ideas', '']

// Ready is refused both ways, as if the core only lets an issue out of the triage sections.
const moveTargets: Record<string, string[]> = {
  Inbox: ['Needs plan', 'Ideas'],
  'Needs plan': ['Inbox', 'Ideas'],
  Ideas: ['Inbox', 'Needs plan'],
  '': ['Inbox', 'Needs plan', 'Ideas'],
}

export const sortBySection = (issues: Issue[]) => issues.sort((a, b) => sectionOrder.indexOf(a.section) - sectionOrder.indexOf(b.section))

export const movesFrom = (section: string) => moveTargets[section] ?? []

export function moveIssue(issue: Issue, section: string): Issue {
  const lane = (Object.keys(sections) as Lane[]).find((l) => sections[l].name === section)
  if (!movesFrom(issue.section).includes(section) || !lane) throw `issue #${issue.number} cannot move to ${section}`
  const laneLabels = Object.keys(sections).map((l) => labelFor(l as Lane))
  const labels = [...issue.labels.filter((l) => !laneLabels.includes(l)), labelFor(lane)].filter(Boolean)
  return { ...issue, section, actions: sections[lane].actions, moves: movesFrom(section), labels }
}

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
      moves: movesFrom(sections[lane].name),
      url: `https://github.com/${repo}/issues/${number}`,
      sessionId: '',
      author,
      assignees,
      labels: [label, type ? `type:${type}` : '', ...extra].filter(Boolean),
      createdAt: now - (i + 2) * day,
      updatedAt: now - (i % 5) * day,
    }
  })
  return sortBySection(issues)
}

const html = (parts: string[]) => parts.join('\n')

const bodies: Record<number, string> = {
  454: html([
    '<h2 dir="auto">Description</h2>',
    '<p dir="auto">The product page mixes server and client code. Split it so each part loads only what it needs.</p>',
    '<ul dir="auto"><li>Move the gallery into <code>product/gallery.tsx</code></li><li class="task-list-item"><input type="checkbox" disabled> Keep the URL the same</li><li class="task-list-item"><input type="checkbox" checked disabled> Measure the bundle before and after</li></ul>',
    '<p dir="auto">See <a href="https://github.com/acme/storefront/issues/444">#444</a> for the checkout side of this.</p>',
    '<div class="highlight highlight-source-ts"><pre>export const Gallery = lazy(() =&gt; import(\'./gallery\'))</pre></div>',
  ]),
  12: html([
    '<h2 dir="auto">Description</h2>',
    '<p dir="auto">Coming back to a project should land on the session I left, not the first one in the list.</p>',
    '<blockquote><p dir="auto">The app already remembers the last project; do the same per project for the session.</p></blockquote>',
  ]),
}

export function buildIssueDetail(number: number): IssueDetail {
  const now = Date.now()
  const comments = number === 454 ? [{ author: 'alex-r', createdAt: now - 2 * day, bodyHTML: '<p dir="auto">The checkout route depends on this, so land it first.</p>' }] : []
  return { number, bodyHTML: bodies[number] ?? '', comments }
}
