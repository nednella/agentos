import { DigestView } from './components/DigestView'
import { Divider } from './components/Divider'
import { CommandLine } from './components/CommandLine'
import { MobileTabs } from './components/MobileTabs'
import { Palette } from './components/Palette'
import { ProjectPanel } from './components/ProjectPanel'
import { SessionsPanel } from './components/SessionsPanel'
import { ShortcutsSheet } from './components/ShortcutsSheet'
import { Sidebar } from './components/Sidebar'
import { Stats } from './components/Stats'
import { Toasts } from './components/Toasts'
import { TopBar } from './components/TopBar'
import { Viewport } from './components/Viewport'
import { useLayout } from './LayoutContext'
import { useShortcuts } from './useShortcuts'

export function App() {
  useShortcuts()
  const { mode, sidebarOpen, sessionsOpen, mobilePanel, statsOpen, digestOpen } = useLayout()
  const narrow = mode === 'narrow'
  const showCentre = !narrow || mobilePanel === 'session'

  return (
    <div className="flex h-full flex-col bg-app" data-mode={mode}>
      <TopBar />
      <main className={`relative flex min-h-0 min-w-0 flex-1 ${narrow ? 'p-2' : 'p-3'}`}>
        <Sidebar />
        {mode === 'wide' && sidebarOpen && <Divider panel="sidebar" />}
        {!narrow && !(mode === 'wide' && sidebarOpen) && <span className="w-3 flex-none" />}
        <div className={`min-h-0 min-w-0 flex-1 ${showCentre ? '' : 'hidden'}`}>
          <div className={statsOpen || digestOpen ? 'hidden' : 'h-full'}>
            <Viewport />
          </div>
          {statsOpen && <Stats />}
          {digestOpen && <DigestView />}
        </div>
        {!narrow && sessionsOpen && <Divider panel="sessions" />}
        {!narrow && !sessionsOpen && <span className="w-3 flex-none" />}
        <SessionsPanel />
      </main>
      {narrow && <MobileTabs />}
      <CommandLine />
      <Palette />
      <ProjectPanel />
      <ShortcutsSheet />
      <Toasts />
    </div>
  )
}
