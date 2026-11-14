declare module 'omggif' {
  export interface GifFrameInfo {
    x: number
    y: number
    width: number
    height: number
    delay: number
    disposal: number
    transparent_index: number | null
    interlaced: boolean
  }

  export class GifReader {
    width: number
    height: number
    constructor(buffer: Uint8Array)
    numFrames(): number
    loopCount(): number | null
    frameInfo(frame: number): GifFrameInfo
    decodeAndBlitFrameRGBA(frame: number, pixels: Uint8ClampedArray): void
  }
}
