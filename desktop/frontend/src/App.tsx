import { Divider } from './components/Divider'
import { MobileTabs } from './components/MobileTabs'
import { ProjectPanel } from './components/ProjectPanel'
import { Sidebar } from './components/Sidebar'
import { TopBar } from './components/TopBar'
import { useLayout } from './LayoutContext'
import { useShortcuts } from './useShortcuts'

export function App() {
  useShortcuts()
  const { mode, sidebarOpen, mobilePanel } = useLayout()
  const narrow = mode === 'narrow'
  const showCentre = !narrow || mobilePanel === 'session'

  return (
    <div className="flex h-full flex-col bg-app" data-mode={mode}>
      <TopBar />
      <main className={`relative flex min-h-0 min-w-0 flex-1 ${narrow ? 'p-2' : 'p-3'}`}>
        <Sidebar />
        {!narrow && sidebarOpen && <Divider panel="sidebar" />}
        {!narrow && !sidebarOpen && <span className="w-3 flex-none" />}
        <div className={`min-h-0 min-w-0 flex-1 ${showCentre ? '' : 'hidden'}`} />
      </main>
      {narrow && <MobileTabs />}
      <ProjectPanel />
    </div>
  )
}
