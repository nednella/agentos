import { duration, useClock } from '../time'

type ElapsedProps = { since: number }

export function Elapsed({ since }: ElapsedProps) {
  return <>{useClock((now) => duration(now - since))}</>
}
