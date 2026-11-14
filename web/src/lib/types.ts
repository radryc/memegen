export interface Template {
  id: string
  name: string
  imageUrl: string
  width: number
  height: number
  source: string
  animated: boolean
  createdAt: string
}

export interface MemeStyle {
  fontScale: number
  color: string
  stroke: string
  uppercase: boolean
  outline: boolean
}

export interface Meme {
  id: string
  title: string
  author: string
  imageUrl: string
  templateId?: string
  templateName?: string
  topText: string
  bottomText: string
  style: string
  width?: number
  height?: number
  score: number
  upvotes: number
  downvotes: number
  myVote: number
  animated: boolean
  createdAt: string
}

export interface Stats {
  memes: number
  votes: number
  templates: number
}

export type SortMode = 'hot' | 'new' | 'top'

export const DEFAULT_STYLE: MemeStyle = {
  fontScale: 1,
  color: '#ffffff',
  stroke: '#000000',
  uppercase: true,
  outline: true,
}

export function parseStyle(raw: string | undefined | null): MemeStyle {
  if (!raw) return { ...DEFAULT_STYLE }
  try {
    const parsed = JSON.parse(raw) as Partial<MemeStyle>
    return {
      fontScale:
        typeof parsed.fontScale === 'number'
          ? Math.min(1.6, Math.max(0.6, parsed.fontScale))
          : DEFAULT_STYLE.fontScale,
      color: /^#[0-9a-fA-F]{6}$/.test(parsed.color ?? '') ? parsed.color! : DEFAULT_STYLE.color,
      stroke: /^#[0-9a-fA-F]{6}$/.test(parsed.stroke ?? '') ? parsed.stroke! : DEFAULT_STYLE.stroke,
      uppercase: typeof parsed.uppercase === 'boolean' ? parsed.uppercase : DEFAULT_STYLE.uppercase,
      outline: typeof parsed.outline === 'boolean' ? parsed.outline : DEFAULT_STYLE.outline,
    }
  } catch {
    return { ...DEFAULT_STYLE }
  }
}
