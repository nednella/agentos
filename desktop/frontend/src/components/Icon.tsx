type IconName =
  | 'play'
  | 'chevron'
  | 'refresh'
  | 'plus'
  | 'close'
  | 'external'
  | 'search'
  | 'folder'
  | 'check'
  | 'trash'
  | 'help'
  | 'panel-left'
  | 'panel-right'
  | 'chart'
  | 'pin'
  | 'archive'
  | 'broom'
  | 'next'
  | 'queue'
  | 'x'
  | 'image'
  | 'digest'
  | 'moon'
  | 'sun'
  | 'back'
  | 'forward'
  | 'reload'
  | 'stop'
  | 'camera'
  | 'compare'
  | 'issue'

type IconProps = { name: IconName; size?: number }

export function Icon({ name, size = 14 }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <IconShape name={name} />
    </svg>
  )
}

type IconShapeProps = { name: IconName }

function IconShape({ name }: IconShapeProps) {
  if (name === 'play') return <path d="M5 3.5v9l7-4.5Z" fill="currentColor" />
  if (name === 'chevron') return <path d="m5 6.5 3 3 3-3" />
  if (name === 'refresh') return <path d="M13 8a5 5 0 1 1-1.6-3.7M13 2.5v3h-3" />
  if (name === 'plus') return <path d="M8 3v10M3 8h10" />
  if (name === 'close') return <path d="m4 4 8 8M12 4l-8 8" />
  if (name === 'external') return <path d="M9 3h4v4M13 3 7.5 8.5M11 9.5V13H3V5h3.5" />
  if (name === 'search') return <path d="M7 12A5 5 0 1 0 7 2a5 5 0 0 0 0 10ZM11 11l3 3" />
  if (name === 'check') return <path d="m3.5 8.5 3 3 6-7" />
  if (name === 'trash') return <path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.5 8.5h6l.5-8.5" />
  if (name === 'panel-left') return <path d="M2.5 3h11v10h-11ZM6.5 3v10" />
  if (name === 'panel-right') return <path d="M2.5 3h11v10h-11ZM9.5 3v10" />
  if (name === 'moon') return <path d="M13 9.5A5.5 5.5 0 0 1 6.5 3a5.5 5.5 0 1 0 6.5 6.5Z" />
  if (name === 'sun') return <path d="M8 10.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5ZM8 1.5v1.5M8 13v1.5M1.5 8H3M13 8h1.5M3.4 3.4l1 1M11.6 11.6l1 1M3.4 12.6l1-1M11.6 4.4l1-1" />
  if (name === 'chart') return <path d="M3 13V8M7 13V3M11 13V6M14 13H2" />
  if (name === 'pin') return <path d="M9.5 2.5 13.5 6.5 11 7l-2 3-.5 3-5-5 3-.5 3-2ZM5.5 10.5 2.5 13.5" />
  if (name === 'archive') return <path d="M2 3.5h12v3H2ZM3 6.5V13h10V6.5M6.5 9h3" />
  if (name === 'broom') return <path d="M9.5 2.5 13 6M8 5l3 3-4.5 5.5H2.5V9.5ZM5 8l3 3" />
  if (name === 'next') return <path d="M3 3v10l6-5ZM11.5 3v10" />
  if (name === 'queue') return <path d="M2.5 4h11M2.5 8h11M2.5 12h7" />
  if (name === 'x') return <path d="m4.5 4.5 7 7M11.5 4.5l-7 7" />
  if (name === 'image') return <path d="M2.5 3h11v10h-11ZM2.5 11l3.5-3.5 3 3 2-2 2.5 2.5M10.5 6.2v.01" />
  if (name === 'digest') return <path d="M3 2.5h8.5v11H4.5a1.5 1.5 0 0 1-1.5-1.5ZM11.5 5H14v7a1.5 1.5 0 0 1-1.5 1.5M5.5 5.5h4M5.5 8h4M5.5 10.5h2.5" />
  if (name === 'back') return <path d="M9.5 3.5 5 8l4.5 4.5M5 8h8" />
  if (name === 'forward') return <path d="M6.5 3.5 11 8l-4.5 4.5M11 8H3" />
  if (name === 'reload') return <path d="M13 8a5 5 0 1 1-1.6-3.7M13 2.5v3h-3" />
  if (name === 'stop') return <path d="m4 4 8 8M12 4l-8 8" />
  if (name === 'camera') return <path d="M2 5h3l1-1.5h4L11 5h3v8H2ZM8 11.2a2.2 2.2 0 1 0 0-4.4 2.2 2.2 0 0 0 0 4.4Z" />
  if (name === 'compare') return <path d="M2.5 3h4.5v10H2.5ZM9 3h4.5v10H9Z" />
  if (name === 'issue') return <path d="M8 13.5a5.5 5.5 0 1 0 0-11 5.5 5.5 0 0 0 0 11ZM8 5.5v3.2M8 10.8v.01" />
  if (name === 'help') return <path d="M6 6a2 2 0 1 1 3 1.7c-.7.4-1 .8-1 1.5M8 12v.01" />
  return <path d="M2 4h4l1.5 1.5H14V13H2Z" />
}
