import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './App'
import { AgentosProvider } from './AgentosContext'
import { LayoutProvider } from './LayoutContext'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AgentosProvider>
      <LayoutProvider>
        <App />
      </LayoutProvider>
    </AgentosProvider>
  </StrictMode>,
)
