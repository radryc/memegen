import type { MemeStyle } from './types'

const FONT_STACK = `Anton, "Arial Black", Impact, "Helvetica Neue", sans-serif`
const MAX_EXPORT_SIDE = 1600

export function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('Could not load that image'))
    img.src = src
  })
}

/** Wait for the display font so canvas text matches what the user sees. */
export async function ensureFonts(): Promise<void> {
  try {
    await Promise.all([document.fonts.load(`400 96px Anton`), document.fonts.ready])
  } catch {
    // Drawing falls back to Arial Black / Impact.
  }
}

function fontFor(size: number): string {
  return `400 ${size}px ${FONT_STACK}`
}

function wrapText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string[] {
  const lines: string[] = []
  for (const paragraph of text.split('\n')) {
    const words = paragraph.trim().split(/\s+/).filter(Boolean)
    if (words.length === 0) {
      lines.push('')
      continue
    }
    let line = words[0]
    for (let i = 1; i < words.length; i += 1) {
      const candidate = `${line} ${words[i]}`
      if (ctx.measureText(candidate).width <= maxWidth) {
        line = candidate
      } else {
        lines.push(line)
        line = words[i]
      }
    }
    lines.push(line)
  }
  return lines
}

interface Layout {
  size: number
  top: string[]
  bottom: string[]
}

function layoutText(
  ctx: CanvasRenderingContext2D,
  topText: string,
  bottomText: string,
  width: number,
  height: number,
  style: MemeStyle,
): Layout {
  const maxWidth = width * 0.92
  const base = width * 0.095 * style.fontScale
  const min = Math.max(12, width * 0.032)
  const lineHeightRatio = 1.08
  let size = Math.max(min, base)
  let top: string[] = []
  let bottom: string[] = []

  for (let guard = 0; guard < 40; guard += 1) {
    ctx.font = fontFor(size)
    top = wrapText(ctx, topText, maxWidth)
    bottom = wrapText(ctx, bottomText, maxWidth)
    const widest = [...top, ...bottom].reduce(
      (widestLine, line) => Math.max(widestLine, ctx.measureText(line).width),
      0,
    )
    const topHeight = top.length * size * lineHeightRatio
    const bottomHeight = bottom.length * size * lineHeightRatio

    const fits =
      widest <= width * 0.94 && topHeight <= height * 0.42 && bottomHeight <= height * 0.42
    if (fits) break
    if (size <= min) break
    size = Math.max(min, size - Math.max(1, Math.round(size * 0.055)))
  }

  return { size, top, bottom }
}

/** Draws an image plus classic top/bottom meme captions onto a canvas. */
export function renderMeme(
  canvas: HTMLCanvasElement,
  image: HTMLImageElement,
  topText: string,
  bottomText: string,
  style: MemeStyle,
): void {
  const naturalWidth = image.naturalWidth || image.width
  const naturalHeight = image.naturalHeight || image.height
  if (!naturalWidth || !naturalHeight) return

  const scale = Math.min(1, MAX_EXPORT_SIDE / Math.max(naturalWidth, naturalHeight))
  const width = Math.max(1, Math.round(naturalWidth * scale))
  const height = Math.max(1, Math.round(naturalHeight * scale))
  if (canvas.width !== width) canvas.width = width
  if (canvas.height !== height) canvas.height = height

  const ctx = canvas.getContext('2d')
  if (!ctx) return

  ctx.clearRect(0, 0, width, height)
  ctx.imageSmoothingEnabled = true
  ctx.imageSmoothingQuality = 'high'
  ctx.drawImage(image, 0, 0, width, height)

  drawCaptions(ctx, width, height, topText, bottomText, style)
}

/** Draws the caption blocks on a canvas that already contains the artwork. */
export function drawCaptions(
  ctx: CanvasRenderingContext2D,
  width: number,
  height: number,
  topText: string,
  bottomText: string,
  style: MemeStyle,
): void {
  const upper = (value: string) => (style.uppercase ? value.toUpperCase() : value)
  const top = upper(topText).trim()
  const bottom = upper(bottomText).trim()
  if (!top && !bottom) return

  const { size, top: topLines, bottom: bottomLines } = layoutText(
    ctx,
    top,
    bottom,
    width,
    height,
    style,
  )
  const lineHeight = size * 1.08

  ctx.font = fontFor(size)
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  ctx.lineJoin = 'round'
  ctx.miterLimit = 2
  ctx.fillStyle = style.color
  ctx.strokeStyle = style.stroke
  ctx.lineWidth = Math.max(2, size * 0.1)

  const drawBlock = (lines: string[], startY: number) => {
    lines.forEach((line, index) => {
      if (!line) return
      const x = width / 2
      const y = startY + index * lineHeight
      if (style.outline) {
        ctx.strokeText(line, x, y, width * 0.94)
      }
      ctx.fillText(line, x, y, width * 0.94)
    })
  }

  const pad = height * 0.028
  drawBlock(topLines, pad)
  drawBlock(bottomLines, height - pad - bottomLines.length * lineHeight)
}

export function canvasToBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (blob) resolve(blob)
      else reject(new Error('Could not export the meme image'))
    }, 'image/png')
  })
}
