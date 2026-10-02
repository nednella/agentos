import { useEffect, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { ConfirmRow } from './ConfirmRow'

type KillButtonProps = { id: string }

export function KillButton({ id }: KillButtonProps) {
  const { killSession, report } = useAgentos()
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    if (!confirming) return
    const timer = setTimeout(() => setConfirming(false), 5000)
    return () => clearTimeout(timer)
  }, [confirming])

  useEffect(() => setConfirming(false), [id])

  if (!confirming) {
    return (
      <button className="btn btn-ghost" onClick={() => setConfirming(true)} title="Stop this session">
        Stop
      </button>
    )
  }
  return <ConfirmRow inline danger message="Stop this session?" confirmLabel="Stop" onCancel={() => setConfirming(false)} onConfirm={() => report(() => killSession(id))} />
}
