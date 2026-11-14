/** Returns the first image file contained in a clipboard or drop payload. */
export function imageFromDataTransfer(data: DataTransfer | null): File | null {
  if (!data) return null
  for (const item of Array.from(data.items ?? [])) {
    if (item.kind === 'file' && item.type.startsWith('image/')) {
      const file = item.getAsFile()
      if (file) return file
    }
  }
  for (const file of Array.from(data.files ?? [])) {
    if (file.type.startsWith('image/')) return file
  }
  return null
}

export function imageFromClipboard(event: ClipboardEvent): File | null {
  return imageFromDataTransfer(event.clipboardData)
}
