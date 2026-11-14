import type { Meme, MemeStyle, SortMode, Stats, Template } from './types'

const VOTER_KEY = 'memebank:voter-id'

/** Stable anonymous voter id kept in localStorage. */
export function getVoterId(): string {
  let id = ''
  try {
    id = localStorage.getItem(VOTER_KEY) ?? ''
  } catch {
    id = ''
  }
  if (!/^[A-Za-z0-9_-]{8,64}$/.test(id)) {
    id =
      typeof crypto !== 'undefined' && 'randomUUID' in crypto
        ? crypto.randomUUID().replace(/-/g, '')
        : Math.random().toString(36).slice(2) + Date.now().toString(36)
    try {
      localStorage.setItem(VOTER_KEY, id)
    } catch {
      // Storage can be unavailable (private mode); votes still work per page load.
    }
  }
  return id
}

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('X-Voter-ID', getVoterId())
  if (init.body && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  let res: Response
  try {
    res = await fetch(path, { ...init, headers })
  } catch {
    throw new ApiError('Could not reach the server. Is the API running?', 0)
  }
  if (!res.ok) {
    let message = `Request failed (${res.status})`
    try {
      const body = (await res.json()) as { error?: string }
      if (body.error) message = body.error
    } catch {
      // ignore non-JSON errors
    }
    throw new ApiError(message, res.status)
  }
  return (await res.json()) as T
}

export function fetchTemplates(params: { source?: string; q?: string; media?: string } = {}) {
  const q = new URLSearchParams()
  if (params.source) q.set('source', params.source)
  if (params.q) q.set('q', params.q)
  if (params.media) q.set('media', params.media)
  const suffix = q.toString() ? `?${q}` : ''
  return request<{ templates: Template[]; total: number }>(`/api/templates${suffix}`)
}

export function uploadTemplate(file: File, name: string) {
  const form = new FormData()
  form.set('image', file)
  form.set('name', name)
  return request<{ template: Template; duplicate?: boolean; match?: 'exact' | 'similar' }>(
    '/api/templates',
    { method: 'POST', body: form },
  )
}

export function fetchMemes(params: {
  sort?: SortMode
  limit?: number
  offset?: number
  q?: string
  template?: string
} = {}) {
  const q = new URLSearchParams()
  if (params.sort) q.set('sort', params.sort)
  if (params.limit) q.set('limit', String(params.limit))
  if (params.offset) q.set('offset', String(params.offset))
  if (params.q) q.set('q', params.q)
  if (params.template) q.set('template', params.template)
  const suffix = q.toString() ? `?${q}` : ''
  return request<{ memes: Meme[]; hasMore: boolean; nextOffset: number }>(`/api/memes${suffix}`)
}

export function fetchMeme(id: string) {
  return request<{ meme: Meme }>(`/api/memes/${encodeURIComponent(id)}`)
}

export function createMeme(input: {
  image?: Blob
  title: string
  author: string
  topText: string
  bottomText: string
  templateId: string
  style: MemeStyle
}) {
  const form = new FormData()
  if (input.image) {
    form.set('image', input.image, input.image.type === 'image/gif' ? 'meme.gif' : 'meme.png')
  }
  form.set('title', input.title)
  form.set('author', input.author)
  form.set('topText', input.topText)
  form.set('bottomText', input.bottomText)
  form.set('templateId', input.templateId)
  form.set('style', JSON.stringify(input.style))
  return request<{ meme: Meme }>('/api/memes', { method: 'POST', body: form })
}

export function castVote(id: string, value: -1 | 1) {
  return request<{ upvotes: number; downvotes: number; score: number; myVote: number }>(
    `/api/memes/${encodeURIComponent(id)}/vote`,
    { method: 'POST', body: JSON.stringify({ value }) },
  )
}

export function fetchLeaderboard(limit = 8) {
  return request<{ memes: Meme[] }>(`/api/leaderboard?limit=${limit}`)
}

export function fetchStats() {
  return request<Stats>('/api/stats')
}
