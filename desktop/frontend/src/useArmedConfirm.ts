import { useState } from 'react'

export type ArmedConfirm<K> = { armed: K | null; arm(key?: K): void; disarm(): void }

export function useArmedConfirm<K = true>(): ArmedConfirm<K> {
  const [armed, setArmed] = useState<K | null>(null)
  return { armed, arm: (key) => setArmed(key ?? (true as K)), disarm: () => setArmed(null) }
}
