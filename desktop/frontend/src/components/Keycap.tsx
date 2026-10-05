type KeycapProps = { children: string }

export function Keycap({ children }: KeycapProps) {
  return <kbd className="keycap">{children}</kbd>
}
