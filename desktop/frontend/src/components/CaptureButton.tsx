import { useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import { Icon } from './Icon'
import { InlineInput } from './InlineInput'

type CaptureButtonProps = { id: string }

export function CaptureButton({ id }: CaptureButtonProps) {
  const { report, pushToast } = useAgentos()
  const [capturing, setCapturing] = useState(false)

  if (capturing) {
    return (
      <InlineInput
        placeholder="Caption (optional), Enter to capture"
        blur="cancel"
        className="w-64 max-w-full"
        onSubmit={(caption) => {
          setCapturing(false)
          report(async () => {
            await api.browserScreenshot(id, caption.trim())
            pushToast({ tone: 'info', text: 'Captured to Evidence' })
          })
        }}
        onCancel={() => setCapturing(false)}
      />
    )
  }
  return (
    <button className="btn" title="Capture the page as evidence" onClick={() => setCapturing(true)}>
      <Icon name="camera" />
      <span className="hidden sm:inline">Capture</span>
    </button>
  )
}
