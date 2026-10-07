import { Icon } from './Icon'

type BrowserConsoleListProps = { lines: string[] }

export function BrowserConsoleList({ lines }: BrowserConsoleListProps) {
  return (
    <section>
      <h4 className="label pb-2">Console</h4>
      {lines.length === 0 ? (
        <p className="text-small text-dim">No console errors</p>
      ) : (
        <ul className="flex flex-col gap-1">
          {lines.map((line, i) => (
            <li key={i} className="flex items-baseline gap-2" title={line}>
              <span className="flex-none self-center text-danger">
                <Icon name="x" size={12} />
              </span>
              <span className="mono min-w-0 truncate text-small">{line}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
