const ESC = '\x1b['
const reset = `${ESC}0m`
const dim = (s: string) => `${ESC}2m${s}${reset}`
const bold = (s: string) => `${ESC}1m${s}${reset}`
const fg = (r: number, g: number, b: number, s: string) => `${ESC}38;2;${r};${g};${b}m${s}${reset}`
const rowBg = (r: number, g: number, b: number, s: string) => `${ESC}48;2;${r};${g};${b}m${s}${ESC}K${reset}`

const orange = (s: string) => fg(255, 150, 90, s)
const cyan = (s: string) => fg(80, 210, 255, s)
const green = (s: string) => fg(70, 225, 150, s)
const amber = (s: string) => fg(255, 190, 60, s)

const visibleLength = (s: string) => s.replace(/\x1b\[[0-9;]*m/g, '').length

function box(color: (s: string) => string, width: number, rows: string[]): string {
  const rule = '─'.repeat(width + 2)
  const body = rows.map((row) => `${color('│')} ${row}${' '.repeat(width - visibleLength(row))} ${color('│')}`)
  return [color(`╭${rule}╮`), ...body, color(`╰${rule}╯`), ''].join('\r\n') + '\r\n'
}

export function banner(cwd: string): string {
  return box(orange, 52, [`${orange('✻')} ${bold('Welcome to Claude Code')}`, `  ${dim(`cwd: ${cwd}`)}`])
}

export function promptLine(text: string): string {
  return `${cyan('>')} ${text}\r\n\r\n`
}

export function diffBlock(file: string): string {
  const context = (n: number, s: string) => dim(`    ${String(n).padStart(3)}   ${s}`)
  const removed = (n: number, s: string) => rowBg(70, 24, 36, `${fg(255, 120, 130, `    ${String(n).padStart(3)} - ${s}`)}`)
  const added = (n: number, s: string) => rowBg(20, 62, 46, `${fg(110, 240, 170, `    ${String(n).padStart(3)} + ${s}`)}`)
  return [
    `${green('●')} ${bold('Update')}(${file})`,
    `  ${dim('⎿')}  Updated ${file} with 3 additions and 2 removals`,
    context(41, "import { useParams } from 'react-router-dom'"),
    context(42, ''),
    removed(43, 'const link = useLinkContext()'),
    removed(44, 'const doc = link.document'),
    added(43, 'const { document: doc, link } = useDocumentLink()'),
    added(44, 'if (!doc) return <LiveDocumentLoadError />'),
    added(45, 'const canEdit = link.access === "edit"'),
    context(46, ''),
    context(47, 'return ('),
    '',
  ].join('\r\n') + '\r\n'
}

export const paragraphs = [
  'The entry file now answers with edit or view access, so the viewer no longer needs its own layout wrapper. I moved the reader into the document folder and renamed the context to match the link it carries.',
  'Next I will run the type checker across the frontend and confirm that no import still points at the old partials folder. If anything is left over it will show up as a missing module.',
  'Two components still read the old context. I am switching them over now, then I will sweep the tests that mocked the previous shape.',
  'The beacon fires on pagehide, not unload, because Safari drops unload handlers once the page is in the back-forward cache. That keeps the last view duration intact.',
]

export const toolLines = [
  'Read(app-frontend/src/js/pages/[slug]/Index.tsx)',
  'Search(pattern: "documentContext", path: "app-frontend")',
  'Bash(npx tsc --noEmit)',
  'Update(app-frontend/src/js/pages/[slug]/Layout.tsx)',
  'Read(app-frontend/src/js/pages/[slug]/edit/Index.tsx)',
]

export function toolCall(index: number): string {
  return `${green('●')} ${bold(toolLines[index % toolLines.length])}\r\n  ${dim('⎿  done')}\r\n\r\n`
}

export function permissionPrompt(command: string): string {
  return box(amber, 58, [
    bold('Bash command'),
    `  ${command}`,
    'Do you want to proceed?',
    cyan('❯ 1. Yes'),
    "  2. Yes, and don't ask again this session",
    '  3. No, and tell Claude what to do differently',
  ])
}

export function finishedNote(seconds: number): string {
  return `${green('✻')} ${dim(`Cooked for ${Math.floor(seconds / 60)}m ${seconds % 60}s`)}\r\n\r\n${cyan('>')} `
}
