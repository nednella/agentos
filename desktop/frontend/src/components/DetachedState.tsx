import { Masthead } from './Masthead'

export function DetachedState() {
  return (
    <div className="grid h-full place-items-center">
      <div className="flex max-w-sm flex-col items-center gap-2 text-center">
        <Masthead />
      </div>
    </div>
  )
}
