const GB = 2 ** 30

export function gigabytes(bytes: number): string {
  return `${(bytes / GB).toFixed(1)} GB`
}

export const memoryNote = 'Memory as Activity Monitor counts it, added up over the processes: an estimate.'
