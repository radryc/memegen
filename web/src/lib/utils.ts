export function cn(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(' ')
}

export function formatTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return 'just now'
  const seconds = Math.round((Date.now() - date.getTime()) / 1000)
  if (seconds < 45) return 'just now'
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  if (days < 30) return `${days}d ago`
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, '')}m`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1).replace(/\.0$/, '')}k`
  return String(n)
}

export function scoreTone(score: number): string {
  if (score > 0) return 'text-acid-400'
  if (score < 0) return 'text-hot-400'
  return 'text-zinc-400'
}

export function initials(name: string): string {
  const cleaned = name.trim()
  if (!cleaned) return '??'
  const parts = cleaned.split(/\s+/)
  return (parts[0][0] + (parts[1]?.[0] ?? parts[0][1] ?? '')).toUpperCase()
}

/** Link that reopens the editor with this meme's template and caption. */
export function remixHref(meme: {
  templateId?: string
  topText: string
  bottomText: string
  style: string
}): string {
  const params = new URLSearchParams()
  if (meme.templateId) params.set('template', meme.templateId)
  if (meme.topText) params.set('top', meme.topText)
  if (meme.bottomText) params.set('bottom', meme.bottomText)
  if (meme.style && meme.style !== '{}') params.set('style', meme.style)
  const query = params.toString()
  return `/create${query ? `?${query}` : ''}`
}
