import { Icon } from './Icon'

type ImageViewerProps = { url: string; onClose(): void }

export function ImageViewer({ url, onClose }: ImageViewerProps) {
  return (
    <div
      className="fade-in fixed inset-0 grid place-items-center p-6"
      style={{ zIndex: 'var(--z-overlay)', background: 'var(--bg-overlay-strong)' }}
      role="dialog"
      aria-label="Image"
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
      onKeyDown={(e) => e.key === 'Escape' && onClose()}
    >
      <div className="relative max-h-full max-w-full">
        <img src={url} alt="Note attachment" className="block max-h-[85vh] max-w-full rounded-md border border-line-strong object-contain" />
        <button className="btn absolute top-2 right-2 h-8 w-8 justify-center px-0" autoFocus aria-label="Close image" onClick={onClose}>
          <Icon name="close" />
        </button>
      </div>
    </div>
  )
}
