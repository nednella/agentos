import { useAgentos } from '../AgentosContext'
import { PaletteDialog } from './PaletteDialog'

export function Palette() {
  const { overlay } = useAgentos()
  if (overlay !== 'palette') return null
  return <PaletteDialog />
}
