import type { PendingImage } from './AgentosContext'

export type PreviewImage = PendingImage & { preview: string }

export function imageFiles(data: DataTransfer | null): File[] {
  return [...(data?.files ?? [])].filter((file) => file.type.startsWith('image/'))
}

export function readImage(file: File): Promise<PreviewImage> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject('Could not read the image')
    reader.onload = () => {
      const preview = String(reader.result)
      resolve({ base64: preview.slice(preview.indexOf(',') + 1), mime: file.type, preview })
    }
    reader.readAsDataURL(file)
  })
}
