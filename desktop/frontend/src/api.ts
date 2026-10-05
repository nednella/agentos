import { createMock } from './mock'
import type { BrowserInput, BrowserState, Cleanup, Digest, EventMap, Evidence, Issue, Note, Session, Snapshot, Stats, WaitKind } from './types'

type Backend = {
  Snapshot(): Promise<Snapshot>
  NewSession(title: string, prefill: string): Promise<Session>
  KillSession(id: string): Promise<void>
  DismissSession(id: string): Promise<void>
  RenameSession(id: string, title: string): Promise<void>
  SwitchProject(name: string): Promise<Snapshot>
  AddProject(): Promise<Snapshot>
  AddProjectDir(dir: string): Promise<Snapshot>
  RemoveProject(name: string): Promise<Snapshot>
  AddNote(text: string): Promise<Note>
  UpdateNote(id: string, text: string): Promise<Note>
  SetNotePinned(id: string, pinned: boolean): Promise<Note>
  SetNoteArchived(id: string, archived: boolean): Promise<Note>
  AddNoteImage(id: string, base64: string, mime: string): Promise<Note>
  RemoveNoteImage(id: string, url: string): Promise<Note>
  DeleteNote(id: string): Promise<void>
  NoteToIssue(id: string): Promise<Note>
  NoteToSession(id: string): Promise<Session>
  Issues(refresh: boolean): Promise<Issue[]>
  StartIssue(number: number): Promise<Session>
  TermOpen(id: string, cols: number, rows: number): Promise<void>
  TermWrite(id: string, data: string): Promise<void>
  TermResize(id: string, cols: number, rows: number): Promise<void>
  TermClose(id: string): Promise<void>
  OpenURL(url: string): Promise<void>
  Stats(days: number): Promise<Stats>
  RefreshPRs(): Promise<void>
  AckPR(id: string): Promise<void>
  TypeInto(id: string, text: string): Promise<void>
  Cleanup(id: string, force: boolean): Promise<void>
  Cleanups(): Promise<Cleanup[]>
  BrowserOpen(id: string, url: string): Promise<BrowserState>
  BrowserGoto(id: string, url: string): Promise<void>
  BrowserNav(id: string, action: 'back' | 'forward' | 'reload' | 'stop'): Promise<void>
  BrowserInput(id: string, input: BrowserInput): Promise<void>
  BrowserResize(id: string, width: number, height: number): Promise<void>
  BrowserView(id: string, visible: boolean): Promise<void>
  BrowserState(id: string): Promise<BrowserState>
  BrowserScreenshot(id: string, caption: string): Promise<Evidence>
  BrowserClose(id: string): Promise<void>
  Evidence(id: string): Promise<Evidence[]>
  DeleteEvidence(id: string, evidenceId: string): Promise<void>
  HarnessCheck(): Promise<Session>
  Digest(): Promise<Digest>
  RunDigest(): Promise<void>
  DigestToNote(itemId: string): Promise<Note>
  DismissDigestItem(itemId: string): Promise<void>
}

type Unsubscribe = () => void

type Runtime = {
  EventsOn(name: string, handler: (payload: never) => void): Unsubscribe | void
}

declare global {
  interface Window {
    go?: { main: { App: Backend } }
    runtime?: Runtime
  }
}

export type DevFlags = { overlay?: string; tab?: string; cmd?: string; view?: string }

const mock = window.go ? null : createMock(new URLSearchParams(location.search))
const backend: Backend = window.go?.main.App ?? mock!.backend

export const devFlags: DevFlags = mock?.flags ?? {}

export const api = {
  snapshot: () => backend.Snapshot(),
  newSession: (title: string, prefill = '') => backend.NewSession(title, prefill),
  killSession: (id: string) => backend.KillSession(id),
  dismissSession: (id: string) => backend.DismissSession(id),
  renameSession: (id: string, title: string) => backend.RenameSession(id, title),
  switchProject: (name: string) => backend.SwitchProject(name),
  addProject: () => backend.AddProject(),
  addProjectDir: (dir: string) => backend.AddProjectDir(dir),
  removeProject: (name: string) => backend.RemoveProject(name),
  addNote: (text: string) => backend.AddNote(text),
  updateNote: (id: string, text: string) => backend.UpdateNote(id, text),
  setNotePinned: (id: string, pinned: boolean) => backend.SetNotePinned(id, pinned),
  setNoteArchived: (id: string, archived: boolean) => backend.SetNoteArchived(id, archived),
  addNoteImage: (id: string, base64: string, mime: string) => backend.AddNoteImage(id, base64, mime),
  removeNoteImage: (id: string, url: string) => backend.RemoveNoteImage(id, url),
  deleteNote: (id: string) => backend.DeleteNote(id),
  noteToIssue: (id: string) => backend.NoteToIssue(id),
  noteToSession: (id: string) => backend.NoteToSession(id),
  issues: (refresh: boolean) => backend.Issues(refresh),
  startIssue: (number: number) => backend.StartIssue(number),
  termOpen: (id: string, cols: number, rows: number) => backend.TermOpen(id, cols, rows),
  // A terminal can close (project switch, session end) while keys or a resize
  // are still in flight; that is not an error worth surfacing.
  termWrite: (id: string, data: string) => backend.TermWrite(id, data).catch(() => {}),
  termResize: (id: string, cols: number, rows: number) => backend.TermResize(id, cols, rows).catch(() => {}),
  termClose: (id: string) => backend.TermClose(id),
  openURL: (url: string) => backend.OpenURL(url),
  stats: (days: number) => backend.Stats(days).then(readIdleKinds),
  refreshPRs: () => backend.RefreshPRs(),
  ackPR: (id: string) => backend.AckPR(id),
  typeInto: (id: string, text: string) => backend.TypeInto(id, text),
  cleanup: (id: string, force: boolean) => backend.Cleanup(id, force),
  cleanups: () => backend.Cleanups(),
  browserOpen: (id: string, url: string) => backend.BrowserOpen(id, url),
  browserGoto: (id: string, url: string) => backend.BrowserGoto(id, url),
  browserNav: (id: string, action: 'back' | 'forward' | 'reload' | 'stop') => backend.BrowserNav(id, action),
  // Input and resize arrive in bursts and can outlive the tab; losing one is harmless.
  browserInput: (id: string, input: BrowserInput) => backend.BrowserInput(id, input).catch(() => {}),
  browserResize: (id: string, width: number, height: number) => backend.BrowserResize(id, width, height).catch(() => {}),
  browserView: (id: string, visible: boolean) => backend.BrowserView(id, visible).catch(() => {}),
  browserState: (id: string) => backend.BrowserState(id),
  browserScreenshot: (id: string, caption: string) => backend.BrowserScreenshot(id, caption),
  browserClose: (id: string) => backend.BrowserClose(id),
  evidence: (id: string) => backend.Evidence(id),
  deleteEvidence: (id: string, evidenceId: string) => backend.DeleteEvidence(id, evidenceId),
  harnessCheck: () => backend.HarnessCheck(),
  digest: () => backend.Digest(),
  runDigest: () => backend.RunDigest(),
  digestToNote: (itemId: string) => backend.DigestToNote(itemId),
  dismissDigestItem: (itemId: string) => backend.DismissDigestItem(itemId),
}

// The Go side used to call an idle wait "finished"; accept both.
type LegacyWaitKind = WaitKind | 'finished'

function readIdleKinds(stats: Stats): Stats {
  const idle = (kind: LegacyWaitKind): WaitKind => (kind === 'finished' ? 'idle' : kind)
  return {
    ...stats,
    byCause: stats.byCause.map((c) => ({ ...c, kind: idle(c.kind) })),
    recent: stats.recent.map((w) => ({ ...w, kind: idle(w.kind) })),
  }
}

export function on<K extends keyof EventMap>(
  event: K,
  handler: (payload: EventMap[K]) => void,
): Unsubscribe {
  if (mock) return mock.on(event, handler)
  const off = window.runtime?.EventsOn(event, handler as (payload: never) => void)
  return off ?? (() => undefined)
}

export function errorMessage(err: unknown): string {
  if (typeof err === 'string') return err
  if (err instanceof Error) return err.message
  return 'Something went wrong'
}
