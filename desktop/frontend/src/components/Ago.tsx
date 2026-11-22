import { ago, useClock } from '../time'

type AgoProps = { at: number }

export function Ago({ at }: AgoProps) {
  return <>{useClock((now) => ago(at, now))}</>
}
