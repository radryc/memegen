import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import type { Template } from '../lib/types'
import { fetchTemplates, uploadTemplate } from '../lib/api'
import { cn } from '../lib/utils'
import { EmptyState } from '../components/UI'
import { ImageIcon, Loader, Search, Sparkles, Upload } from '../components/Icons'
import TemplateCard from '../components/TemplateCard'
import { imageFromClipboard } from '../lib/clipboard'

type TemplateFilter = 'all' | 'gif' | 'imgflip' | 'upload' | 'generated'

const tabs: { key: TemplateFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'gif', label: 'GIFs' },
  { key: 'imgflip', label: 'Popular' },
  { key: 'upload', label: 'Community' },
  { key: 'generated', label: 'Canvases' },
]

export default function TemplatesPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [templates, setTemplates] = useState<Template[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [source, setSource] = useState<TemplateFilter>('all')
  const [query, setQuery] = useState(searchParams.get('q') ?? '')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [uploading, setUploading] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query.trim()), 250)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchTemplates({
      source: source === 'all' || source === 'gif' ? undefined : source,
      media: source === 'gif' ? 'gif' : undefined,
      q: debouncedQuery || undefined,
    })
      .then((data) => {
        if (!cancelled) setTemplates(data.templates)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Could not load templates')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [source, debouncedQuery])

  const handleUpload = useCallback(
    async (file: File | undefined, nameOverride?: string) => {
      if (!file) return
      setUploading(true)
      setError('')
      try {
        const fallbackName = file.name.replace(/\.[^.]+$/, '').slice(0, 60)
        const name = (nameOverride ?? fallbackName) || 'Community upload'
        const result = await uploadTemplate(file, name)
        navigate(`/create?template=${encodeURIComponent(result.template.id)}`, {
          state: result.duplicate
            ? { duplicateTemplate: true, similar: result.match === 'similar' }
            : undefined,
        })
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Upload failed')
        setUploading(false)
        if (fileRef.current) fileRef.current.value = ''
      }
    },
    [navigate],
  )

  useEffect(() => {
    const onPaste = (event: ClipboardEvent) => {
      const file = imageFromClipboard(event)
      if (!file) return
      event.preventDefault()
      void handleUpload(file, 'Pasted image')
    }
    window.addEventListener('paste', onPaste)
    return () => window.removeEventListener('paste', onPaste)
  }, [handleUpload])

  return (
    <div className="pb-10">
      <div className="flex flex-col gap-5 pt-4 pb-8 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <span className="chip border-ice-400/25 bg-ice-400/[0.07] text-ice-400">
            <ImageIcon className="size-3.5" />
            the library
          </span>
          <h1 className="mt-3 font-display text-4xl tracking-tight text-white uppercase sm:text-5xl">
            Template wall
          </h1>
          <p className="mt-2 max-w-lg text-sm leading-relaxed text-zinc-500">
            Classic reaction images, animated GIFs, community uploads and blank canvases. Pick one,
            caption it, and send it to the board.
          </p>
        </div>

        <button
          type="button"
          onClick={() => fileRef.current?.click()}
          disabled={uploading}
          className="btn-primary self-start py-3 sm:self-auto"
        >
          {uploading ? <Loader className="size-4" /> : <Upload className="size-4" />}
          {uploading ? 'Uploading…' : 'Upload GIF or image'}
        </button>
        <input
          ref={fileRef}
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          className="hidden"
          onChange={(event) => void handleUpload(event.target.files?.[0])}
        />
      </div>

      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-1 rounded-full border border-white/10 bg-ink-900/80 p-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              type="button"
              onClick={() => setSource(tab.key)}
              className={cn(
                'cursor-pointer rounded-full px-3.5 py-1.5 text-xs font-semibold transition',
                source === tab.key ? 'bg-acid-400 text-ink-950' : 'text-zinc-400 hover:text-white',
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <label className="relative w-full sm:w-64">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-zinc-600" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search templates…"
            aria-label="Search templates"
            className="w-full rounded-full border border-white/10 bg-ink-900/80 py-2 pr-3 pl-9 text-xs text-zinc-200 placeholder:text-zinc-600 focus:border-acid-400/50 focus:outline-none"
          />
        </label>
      </div>

      {error && (
        <div className="mb-5 rounded-2xl border border-hot-500/30 bg-hot-500/10 px-4 py-3 text-sm text-hot-300">
          {error}
        </div>
      )}

      {loading ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-5 grid-wide">
          {Array.from({ length: 12 }).map((_, index) => (
            <div key={index} className="skeleton aspect-[4/3] rounded-2xl" />
          ))}
        </div>
      ) : templates.length === 0 ? (
        <EmptyState
          title="No templates here"
          description="Try a different search or upload your own picture to the library."
          action={
            <button type="button" onClick={() => fileRef.current?.click()} className="btn-primary">
              <Upload className="size-4" />
              Upload an image
            </button>
          }
        />
      ) : (
        <>
          <div className="mb-4 flex items-center justify-between text-xs text-zinc-500">
            <span>
              Showing <span className="font-semibold text-zinc-300">{templates.length}</span>{' '}
              templates
            </span>
            <span className="hidden items-center gap-1.5 sm:flex">
              <Sparkles className="size-3.5 text-acid-400" />
              Click a card to use it — or paste an image with ⌘V / Ctrl+V
            </span>
          </div>

          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-5 grid-wide">
            {templates.map((template) => (
              <TemplateCard key={template.id} template={template} />
            ))}
          </div>
        </>
      )}
    </div>
  )
}
