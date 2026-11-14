import { GIFEncoder, applyPalette, quantize } from 'gifenc'
import { GifReader } from 'omggif'
import type { MemeStyle } from './types'
import { drawCaptions, ensureFonts } from './memeCanvas'

const MAX_OUTPUT_SIDE = 900
const MAX_SOURCE_PIXELS = 4096 * 4096
const MAX_FRAMES = 120
const MAX_OUTPUT_BYTES = 11 * 1024 * 1024
const MAX_COLORS = 256

export interface GifMemeOptions {
  src: string
  topText: string
  bottomText: string
  style: MemeStyle
  onProgress?: (fraction: number) => void
}

function clearRect(
  pixels: Uint8ClampedArray,
  width: number,
  height: number,
  x: number,
  y: number,
  w: number,
  h: number,
) {
  const x0 = Math.max(0, x)
  const x1 = Math.min(width, x + w)
  const y0 = Math.max(0, y)
  const y1 = Math.min(height, y + h)
  if (x1 <= x0) return
  for (let py = y0; py < y1; py += 1) {
    const start = (py * width + x0) * 4
    pixels.fill(0, start, start + (x1 - x0) * 4)
  }
}

const yieldToUI = () => new Promise<void>((resolve) => setTimeout(resolve, 0))

/**
 * Composites meme captions onto every frame of a GIF and re-encodes it in the
 * browser. Disposal modes are applied so partial frames stay correct.
 */
export async function renderGifMeme({
  src,
  topText,
  bottomText,
  style,
  onProgress,
}: GifMemeOptions): Promise<Blob> {
  await ensureFonts()

  const response = await fetch(src)
  if (!response.ok) throw new Error('Could not load the GIF template')
  const bytes = new Uint8Array(await response.arrayBuffer())
  const reader = new GifReader(bytes)

  const srcWidth = reader.width
  const srcHeight = reader.height
  if (!srcWidth || !srcHeight) throw new Error('That GIF has invalid dimensions')
  if (srcWidth * srcHeight > MAX_SOURCE_PIXELS) {
    throw new Error('That GIF is too large to process')
  }
  const frameCount = reader.numFrames()
  if (frameCount === 0) throw new Error('That GIF has no frames')
  if (frameCount > MAX_FRAMES) {
    throw new Error(`That GIF has too many frames (max ${MAX_FRAMES})`)
  }

  const scale = Math.min(1, MAX_OUTPUT_SIDE / Math.max(srcWidth, srcHeight))
  const outWidth = Math.max(1, Math.round(srcWidth * scale))
  const outHeight = Math.max(1, Math.round(srcHeight * scale))

  const source = document.createElement('canvas')
  source.width = srcWidth
  source.height = srcHeight
  const sourceCtx = source.getContext('2d', { willReadFrequently: true })
  if (!sourceCtx) throw new Error('Canvas is not available in this browser')

  const output = document.createElement('canvas')
  output.width = outWidth
  output.height = outHeight
  const outputCtx = output.getContext('2d', { willReadFrequently: true })
  if (!outputCtx) throw new Error('Canvas is not available in this browser')
  outputCtx.imageSmoothingEnabled = true
  outputCtx.imageSmoothingQuality = 'high'

  const frame = new Uint8ClampedArray(srcWidth * srcHeight * 4)
  const sourceImage = sourceCtx.createImageData(srcWidth, srcHeight)
  let snapshot: Uint8ClampedArray | null = null
  let disposer = { x: 0, y: 0, width: 0, height: 0, mode: 0 }

  const gif = GIFEncoder()
  const loopCount = reader.loopCount()
  const repeat = loopCount === null ? -1 : loopCount

  for (let i = 0; i < frameCount; i += 1) {
    if (i > 0) {
      if (disposer.mode === 2) {
        clearRect(frame, srcWidth, srcHeight, disposer.x, disposer.y, disposer.width, disposer.height)
      } else if (disposer.mode === 3 && snapshot) {
        frame.set(snapshot)
      }
    }

    const info = reader.frameInfo(i)
    if (info.disposal === 3) snapshot = frame.slice()
    reader.decodeAndBlitFrameRGBA(i, frame)

    sourceImage.data.set(frame)
    sourceCtx.putImageData(sourceImage, 0, 0)
    outputCtx.clearRect(0, 0, outWidth, outHeight)
    outputCtx.drawImage(source, 0, 0, outWidth, outHeight)
    drawCaptions(outputCtx, outWidth, outHeight, topText, bottomText, style)

    const imageData = outputCtx.getImageData(0, 0, outWidth, outHeight).data
    const palette = quantize(imageData, MAX_COLORS, { format: 'rgba4444', oneBitAlpha: true })
    const index = applyPalette(imageData, palette, 'rgba4444')
    const transparentIndex = palette.findIndex((color) => (color[3] ?? 255) === 0)
    if (transparentIndex >= 0) {
      for (let p = 0; p < index.length; p += 1) {
        const entry = palette[index[p]]
        if ((entry[3] ?? 255) === 0) index[p] = transparentIndex
      }
    }

    const delay = info.delay > 0 ? Math.min(1000, info.delay * 10) : 100
    gif.writeFrame(index, outWidth, outHeight, {
      palette,
      delay,
      repeat: i === 0 ? repeat : undefined,
      transparent: transparentIndex >= 0,
      transparentIndex: transparentIndex >= 0 ? transparentIndex : 0,
    })

    disposer = {
      x: info.x,
      y: info.y,
      width: info.width,
      height: info.height,
      mode: info.disposal,
    }

    if (i % 2 === 0) {
      onProgress?.((i + 1) / frameCount)
      await yieldToUI()
    }
  }

  gif.finish()
  const encoded = gif.bytes()
  const buffer = new ArrayBuffer(encoded.byteLength)
  new Uint8Array(buffer).set(encoded)
  const blob = new Blob([buffer], { type: 'image/gif' })
  if (blob.size > MAX_OUTPUT_BYTES) {
    throw new Error('That GIF is too large to post (max 11MB). Try a shorter or smaller GIF.')
  }
  onProgress?.(1)
  return blob
}
