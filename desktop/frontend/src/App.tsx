import { DigestView } from './components/DigestView'
import { NewsIndex } from './components/NewsIndex'
import { NewsView } from './components/NewsView'
import { Divider } from './components/Divider'
import { IssueDialog } from './components/IssueDialog'
import { MobileTabs } from './components/MobileTabs'
import { Palette } from './components/Palette'
import { PatchNotes } from './components/PatchNotes'
import { ProjectPanel } from './components/ProjectPanel'
import { SessionsPanel } from './components/SessionsPanel'
import { SettingsPanel } from './components/SettingsPanel'
import { ShortcutsSheet } from './components/ShortcutsSheet'
import { ShellPanel } from './components/ShellPanel'
import { ShellStrip } from './components/ShellStrip'
import { Sidebar } from './components/Sidebar'
import { Stats } from './components/Stats'
import { Toasts } from './components/Toasts'
import { TopBar } from './components/TopBar'
import { Viewport } from './components/Viewport'
import { useLayout } from './LayoutContext'
import { useNewsToasts } from './useNewsToasts'
import { useShortcuts } from './useShortcuts'
import { useUiCommands } from './useUiCommands'

export function App() {
  useShortcuts()
  useUiCommands()
  useNewsToasts()
  const { mode, sidebarOpen, sessionsOpen, mobilePanel, statsOpen, newsOpen, newsPage } = useLayout()
  const narrow = mode === 'narrow'
  const showCentre = !narrow || mobilePanel === 'session'

  return (
    <div className="flex h-full flex-col bg-app" data-mode={mode}>
      <TopBar />
      <main className={`relative flex min-h-0 min-w-0 flex-1 ${narrow ? 'p-2' : 'p-3'}`}>
        <Sidebar />
        {!narrow && sidebarOpen && <Divider panel="sidebar" />}
        {!narrow && !sidebarOpen && <span className="w-3 flex-none" />}
        <div className={`min-h-0 min-w-0 flex-1 ${showCentre ? '' : 'hidden'}`}>
          <div className={statsOpen || newsOpen ? 'hidden' : 'h-full'}>
            <Viewport />
          </div>
          {statsOpen && <Stats />}
          {newsOpen && newsPage === 'index' && <NewsIndex />}
          {newsOpen && newsPage === 'tldr' && <NewsView />}
          {newsOpen && newsPage === 'digest' && <DigestView />}
        </div>
        {!narrow && sessionsOpen && <Divider panel="sessions" />}
        {!narrow && !sessionsOpen && <span className="w-3 flex-none" />}
        <SessionsPanel />
        {narrow && mobilePanel === 'shell' && <ShellPanel />}
      </main>
      {narrow && <MobileTabs />}
      {!narrow && <ShellStrip />}
      <Palette />
      <ProjectPanel />
      <IssueDialog />
      <ShortcutsSheet />
      <SettingsPanel />
      <PatchNotes />
      <Toasts />
    </div>
  )
}
