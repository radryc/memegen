import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import type { MemeStyle, Template } from '../lib/types'
import { DEFAULT_STYLE, parseStyle } from '../lib/types'
import { createMeme, fetchTemplates, uploadTemplate } from '../lib/api'
import { canvasToBlob, ensureFonts, loadImage, renderMeme } from '../lib/memeCanvas'
import { renderGifMeme } from '../lib/gifMeme'
import { imageFromClipboard } from '../lib/clipboard'
import { cn } from '../lib/utils'
import {
  Check,
  Dice,
  Download,
  ImageIcon,
  Loader,
  Plus,
  Search,
  Sparkles,
  Upload,
} from '../components/Icons'

type TemplateFilter = 'all' | 'imgflip' | 'upload' | 'gif'

const tabs: { key: TemplateFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'gif', label: 'GIFs' },
  { key: 'imgflip', label: 'Popular' },
  { key: 'upload', label: 'Uploads' },
]

const swatches = [
  { name: 'Snow', value: '#ffffff' },
  { name: 'Acid', value: '#c8ff4d' },
  { name: 'Bubblegum', value: '#ff5c9e' },
  { name: 'Ice', value: '#5ce1ff' },
  { name: 'Gold', value: '#ffd166' },
  { name: 'Ink', value: '#111111' },
]

const randomCaptions: [string, string][] = [
  ["me: I'll go to bed early", 'also me at 3am:'],
  ['nobody:', 'my brain at 3am:'],
  ['my code', 'after I add one small change'],
  ['boss: we need this by friday', 'me, who started monday:'],
  ['when the wifi drops', 'and you touch grass for the first time'],
  ['me pretending to understand', 'the conversation'],
  ['my bank account', 'after one (1) payday'],
  ["them: just be yourself", 'me:'],
  ["when someone says 'quick call'", "and it's been 2 hours"],
  ['step 1: make a meme', 'step 2: ???'],
  ['when the deadline was yesterday', "and you're still finishing up"],
  ['my last two brain cells', 'carrying the whole team'],
  ['the bug in production', 'the bug on my machine'],
  ['me reading the docs', 'me guessing anyway'],
  ['my 5-year plan', 'a nap'],
]

export default function CreatePage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [params] = useSearchParams()

  const [templates, setTemplates] = useState<Template[]>([])
  const [templatesLoading, setTemplatesLoading] = useState(true)
  const [source, setSource] = useState<TemplateFilter>('all')
  const [templateQuery, setTemplateQuery] = useState('')
  const [debouncedTemplateQuery, setDebouncedTemplateQuery] = useState('')
  const [selectedId, setSelectedId] = useState(params.get('template') ?? '')
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
  const [templatesReload, setTemplatesReload] = useState(0)

  const [topText, setTopText] = useState(params.get('top') ?? '')
  const [bottomText, setBottomText] = useState(params.get('bottom') ?? '')
  const [title, setTitle] = useState(params.get('title') ?? '')
  const [author, setAuthor] = useState(() => {
    try {
      return localStorage.getItem('memebank:author') ?? ''
    } catch {
      return ''
    }
  })
  const [style, setStyle] = useState<MemeStyle>(() => {
    const raw = params.get('style')
    return raw ? parseStyle(raw) : { ...DEFAULT_STYLE }
  })

  const [image, setImage] = useState<HTMLImageElement | null>(null)
  const [imageLoading, setImageLoading] = useState(false)
  const [imageError, setImageError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [gifProgress, setGifProgress] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [uploadNotice, setUploadNotice] = useState(() => {
    const state = location.state as { duplicateTemplate?: boolean; similar?: boolean } | null
    if (!state?.duplicateTemplate) return ''
    return state.similar
      ? 'A visually identical template was already in the library — using the existing one.'
      : 'That image was already in the library — the existing template is selected.'
  })

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const selectedRef = useRef<HTMLButtonElement | null>(null)
  const gridRef = useRef<HTMLDivElement>(null)

  const selectedImageUrl = selectedTemplate?.imageUrl
  const animated = Boolean(selectedTemplate?.animated)

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedTemplateQuery(templateQuery.trim()), 250)
    return () => clearTimeout(timer)
  }, [templateQuery])

  useEffect(() => {
    let cancelled = false
    setTemplatesLoading(true)
    fetchTemplates({
      source: source === 'all' || source === 'gif' ? undefined : source,
      media: source === 'gif' ? 'gif' : undefined,
      q: debouncedTemplateQuery || undefined,
    })
      .then((data) => {
        if (cancelled) return
        setTemplates(data.templates)
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Could not load templates')
        }
      })
      .finally(() => {
        if (!cancelled) setTemplatesLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [source, debouncedTemplateQuery, templatesReload])

  useEffect(() => {
    // Keep the active template even when it is filtered out of the visible
    // list (for example after switching to the GIFs tab).
    const fromList = templates.find((template) => template.id === selectedId)
    if (fromList) {
      setSelectedTemplate(fromList)
    } else if (!selectedId && templates.length > 0) {
      setSelectedId(templates[0].id)
    }
  }, [selectedId, templates])

  useEffect(() => {
    // Scroll only the thumbnail list, never the page.
    const item = selectedRef.current
    const grid = gridRef.current
    if (!item || !grid) return
    const itemRect = item.getBoundingClientRect()
    const gridRect = grid.getBoundingClientRect()
    if (itemRect.top < gridRect.top) {
      grid.scrollTop += itemRect.top - gridRect.top
    } else if (itemRect.bottom > gridRect.bottom) {
      grid.scrollTop += itemRect.bottom - gridRect.bottom
    }
  }, [selectedId, templates])

  useEffect(() => {
    if (!selectedImageUrl) return
    let cancelled = false
    setImageLoading(true)
    setImageError('')
    ensureFonts()
      .then(() => loadImage(selectedImageUrl))
      .then((loaded) => {
        if (!cancelled) setImage(loaded)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        setImage(null)
        setImageError(err instanceof Error ? err.message : 'Could not load that image')
      })
      .finally(() => {
        if (!cancelled) setImageLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [selectedImageUrl])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!image || !canvas) return
    if (!animated) {
      renderMeme(canvas, image, topText, bottomText, style)
      return
    }
    // GIF templates animate in the browser; redraw the current frame often
    // enough to preview the motion without burning a full 60fps.
    let raf = 0
    let last = 0
    const tick = (time: number) => {
      if (time - last >= 45) {
        renderMeme(canvas, image, topText, bottomText, style)
        last = time
      }
      raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [image, topText, bottomText, style, animated])

  const handleUpload = useCallback(async (file: File | undefined, nameOverride?: string) => {
    if (!file) return
    if (!file.type.startsWith('image/')) {
      setError('Pick an image file (png, jpeg, webp or gif).')
      return
    }
    setUploading(true)
    setError('')
    setUploadNotice('')
    try {
      const fallbackName = file.name.replace(/\.[^.]+$/, '').slice(0, 60)
      const name = (nameOverride ?? fallbackName) || 'Community upload'
      const result = await uploadTemplate(file, name)
      setTemplates((prev) => [result.template, ...prev.filter((t) => t.id !== result.template.id)])
      setSelectedId(result.template.id)
      setSelectedTemplate(result.template)
      setSource('all')
      if (result.duplicate) {
        setUploadNotice(
          result.match === 'similar'
            ? 'A visually identical template was already in the library — using the existing one.'
            : 'That image was already in the library — the existing template is selected.',
        )
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }, [])

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

  const lucky = () => {
    const [top, bottom] = randomCaptions[Math.floor(Math.random() * randomCaptions.length)]
    setTopText(top)
    setBottomText(bottom)
  }

  const downloadDraft = async () => {
    if (!canvasRef.current) return
    setError('')
    try {
      let blob: Blob
      if (animated && selectedTemplate) {
        setGifProgress(0)
        blob = await renderGifMeme({
          src: selectedTemplate.imageUrl,
          topText,
          bottomText,
          style,
          onProgress: setGifProgress,
        })
      } else {
        blob = await canvasToBlob(canvasRef.current)
      }
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = animated ? 'memebank.gif' : 'memebank-draft.png'
      anchor.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not export this meme')
    } finally {
      setGifProgress(null)
    }
  }

  const publish = async () => {
    if (!canvasRef.current || !image || publishing) return
    if (!topText.trim() && !bottomText.trim()) {
      setError('Add a top or bottom caption before posting.')
      return
    }
    setPublishing(true)
    setError('')
    setGifProgress(null)
    try {
      // Memes are stored as a spec only. The server renders stills and
      // animated GIFs on demand, so no image file is written here.
      const result = await createMeme({
        title,
        author,
        topText,
        bottomText,
        templateId: selectedTemplate?.id ?? selectedId,
        style,
      })
      try {
        localStorage.setItem('memebank:author', author)
      } catch {
        // ignore storage errors
      }
      navigate(`/m/${result.meme.id}`, { state: { justPublished: true } })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not publish your meme')
      setPublishing(false)
    } finally {
      setGifProgress(null)
    }
  }

  return (
    <div className="mx-auto max-w-[1240px] pb-10">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <span className="chip border-acid-400/25 bg-acid-400/[0.07] text-acid-300">
            <Sparkles className="size-3.5" />
            the studio
          </span>
          <h1 className="mt-3 font-display text-4xl tracking-tight text-white uppercase sm:text-5xl">
            Create a meme
          </h1>
          <p className="mt-2 text-sm text-zinc-500">
            Template, caption, chaos. Takes about thirty seconds.
          </p>
        </div>
        <Link to="/templates" className="btn-secondary self-start">
          <ImageIcon className="size-4" />
          Browse templates
        </Link>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)]">
        <div className="lg:sticky lg:top-24 lg:self-start">
          <div className="panel overflow-hidden">
            <div className="flex items-center justify-between gap-3 border-b border-white/[0.07] px-4 py-3 text-xs">
              <span className="flex items-center gap-2 font-medium text-zinc-300">
                <span className={cn('size-2 rounded-full', image ? 'bg-acid-400' : 'bg-zinc-600')} />
                Live preview
              </span>
              <span className="flex min-w-0 items-center gap-2">
                {animated && (
                  <span className="rounded-full border border-hot-400/40 bg-hot-500/15 px-1.5 py-0.5 text-[9px] font-bold tracking-wide text-hot-300">
                    GIF
                  </span>
                )}
                <span className="truncate text-zinc-600">
                  {selectedTemplate?.name ?? 'No template selected'}
                </span>
              </span>
            </div>

            <div className="checker p-3 sm:p-5">
              <div className="relative mx-auto flex min-h-56 items-center justify-center overflow-hidden rounded-xl">
                <canvas
                  ref={canvasRef}
                  className={cn(
                    'mx-auto block h-auto max-h-[64vh] w-auto max-w-full rounded-xl transition-opacity duration-300',
                    image && !imageLoading ? 'opacity-100' : 'opacity-0',
                  )}
                />
                {(imageLoading || (!image && !imageError && templatesLoading)) && (
                  <div className="absolute inset-0 grid place-items-center bg-ink-950/40">
                    <Loader className="size-7 text-acid-400" />
                  </div>
                )}
                {!imageLoading && !image && !imageError && !templatesLoading && (
                  <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-center">
                    <ImageIcon className="size-8 text-zinc-700" />
                    <p className="text-xs text-zinc-500">Pick a template to get started</p>
                  </div>
                )}
                {imageError && (
                  <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 px-6 text-center">
                    <p className="text-xs text-hot-300">{imageError}</p>
                    <button
                      type="button"
                      onClick={() => {
                        setImageError('')
                        setSelectedId('')
                        setSelectedTemplate(null)
                        setTemplatesReload((n) => n + 1)
                      }}
                      className="text-xs font-semibold text-white underline underline-offset-4"
                    >
                      Try again
                    </button>
                  </div>
                )}
              </div>
            </div>

            <div className="flex items-center justify-between gap-3 border-t border-white/[0.07] px-4 py-3 text-[11px] text-zinc-600">
              <span>
                {topText.length + bottomText.length} caption characters
              </span>
              <button
                type="button"
                onClick={() => {
                  setTopText('')
                  setBottomText('')
                }}
                className="font-medium text-zinc-500 transition hover:text-zinc-200"
              >
                Clear captions
              </button>
            </div>
          </div>
          <p className="mt-3 text-center text-[11px] leading-relaxed text-zinc-600">
            {animated
              ? 'Animated template — your captions get baked into every frame when you post.'
              : 'Tip: short captions hit harder. Stretch the font with the size slider if it feels cramped.'}
          </p>
        </div>

        <div className="space-y-4">
          <section className="panel p-5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h2 className="text-sm font-bold text-white">Pick your canvas</h2>
                <p className="mt-0.5 text-[11px] text-zinc-600">
                  {templatesLoading ? 'Loading…' : `${templates.length} available`}
                </p>
              </div>
              <button
                type="button"
                onClick={() => fileRef.current?.click()}
                disabled={uploading}
                className="btn-secondary px-3 py-1.5 text-xs"
              >
                {uploading ? <Loader className="size-3.5" /> : <Upload className="size-3.5" />}
                {uploading ? 'Uploading…' : 'Upload'}
              </button>
              <input
                ref={fileRef}
                type="file"
                accept="image/png,image/jpeg,image/webp,image/gif"
                className="hidden"
                onChange={(event) => void handleUpload(event.target.files?.[0])}
              />
            </div>

            <div className="mt-4 flex flex-col gap-2 sm:flex-row sm:items-center">
              <div className="flex items-center gap-0.5 rounded-full border border-white/10 bg-ink-900/80 p-0.5">
                {tabs.map((tab) => (
                  <button
                    key={tab.key}
                    type="button"
                    onClick={() => setSource(tab.key)}
                    className={cn(
                      'cursor-pointer rounded-full px-2.5 py-1 text-[11px] font-semibold transition',
                      source === tab.key
                        ? 'bg-white/10 text-white'
                        : 'text-zinc-500 hover:text-zinc-200',
                    )}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>
              <label className="relative flex-1">
                <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-zinc-600" />
                <input
                  value={templateQuery}
                  onChange={(event) => setTemplateQuery(event.target.value)}
                  placeholder="Search templates…"
                  aria-label="Search templates"
                  className="w-full rounded-full border border-white/10 bg-ink-900/80 py-1.5 pr-3 pl-8 text-[11px] text-zinc-200 placeholder:text-zinc-600 focus:border-acid-400/50 focus:outline-none"
                />
              </label>
            </div>

            {templatesLoading ? (
              <div className="mt-3 grid grid-cols-4 gap-2">
                {[0, 1, 2, 3, 4, 5, 6, 7].map((i) => (
                  <div key={i} className="skeleton aspect-square rounded-lg" />
                ))}
              </div>
            ) : templates.length === 0 ? (
              <div className="mt-3 rounded-xl border border-dashed border-white/10 px-4 py-6 text-center text-xs text-zinc-500">
                No templates match. Try another search or upload your own image.
              </div>
            ) : (
              <div className="relative mt-3">
                <div ref={gridRef} className="grid max-h-64 grid-cols-3 gap-2 overflow-y-auto pr-1 sm:grid-cols-4">
                {templates.map((template) => (
                  <button
                    key={template.id}
                    type="button"
                    ref={selectedId === template.id ? selectedRef : undefined}
                    onClick={() => setSelectedId(template.id)}
                    title={template.name}
                    className={cn(
                      'group relative cursor-pointer overflow-hidden rounded-lg border text-left transition',
                      selectedId === template.id
                        ? 'border-acid-400 ring-2 ring-acid-400/30'
                        : 'border-white/10 hover:border-white/30',
                    )}
                  >
                    <img
                      src={template.imageUrl}
                      alt={template.name}
                      loading="lazy"
                      className="aspect-square w-full object-cover transition duration-300 group-hover:scale-105"
                    />
                    <span className="absolute inset-x-0 bottom-0 truncate bg-gradient-to-t from-black/90 via-black/50 to-transparent px-1.5 pt-4 pb-1 text-[9px] font-medium text-zinc-200">
                      {template.name}
                    </span>
                    {template.animated && (
                      <span className="absolute top-1 left-1 rounded bg-ink-950/80 px-1 py-0.5 text-[8px] font-bold tracking-wider text-hot-300 backdrop-blur">
                        GIF
                      </span>
                    )}
                    {selectedId === template.id && (
                      <span className="absolute top-1 right-1 grid size-4 place-items-center rounded-full bg-acid-400 text-ink-950">
                        <Check className="size-2.5" />
                      </span>
                    )}
                  </button>
                ))}
                </div>
                <div className="pointer-events-none absolute inset-x-0 bottom-0 h-8 bg-gradient-to-t from-ink-850 to-transparent" />
              </div>
            )}

            {uploadNotice && (
              <p className="mt-3 flex items-center gap-1.5 rounded-xl border border-acid-400/25 bg-acid-400/[0.06] px-2.5 py-1.5 text-[10px] font-medium text-acid-300">
                <Check className="size-3 shrink-0" />
                {uploadNotice}
              </p>
            )}
            <p className="mt-3 flex items-start gap-1.5 text-[10px] leading-relaxed text-zinc-600">
              <Upload className="mt-0.5 size-3 shrink-0" />
              <span>
                Upload or paste an image/GIF with{' '}
                <kbd className="rounded border border-white/10 bg-white/[0.04] px-1 py-0.5 font-sans text-[9px] text-zinc-400">
                  ⌘V
                </kbd>{' '}
                /{' '}
                <kbd className="rounded border border-white/10 bg-white/[0.04] px-1 py-0.5 font-sans text-[9px] text-zinc-400">
                  Ctrl+V
                </kbd>
                . Duplicate images reuse the existing template.
              </span>
            </p>
          </section>

          <section className="panel p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-bold text-white">Drop the caption</h2>
              <button type="button" onClick={lucky} className="btn-ghost px-2 py-1 text-[11px]">
                <Dice className="size-3.5 text-acid-400" />
                Feeling lucky
              </button>
            </div>

            <div className="mt-4 space-y-3">
              <label className="block">
                <span className="label">Top text</span>
                <textarea
                  value={topText}
                  onChange={(event) => setTopText(event.target.value)}
                  maxLength={280}
                  rows={2}
                  placeholder="WHEN THE CODE WORKS…"
                  className="input resize-none"
                />
              </label>
              <label className="block">
                <span className="label">Bottom text</span>
                <textarea
                  value={bottomText}
                  onChange={(event) => setBottomText(event.target.value)}
                  maxLength={280}
                  rows={2}
                  placeholder="…ON THE FIRST TRY"
                  className="input resize-none"
                />
              </label>
            </div>
          </section>

          <section className="panel p-5">
            <h2 className="text-sm font-bold text-white">Make it pop</h2>

            <div className="mt-4">
              <div className="flex items-center justify-between">
                <span className="label mb-0">Font size</span>
                <span className="text-[11px] font-semibold text-zinc-400">
                  {Math.round(style.fontScale * 100)}%
                </span>
              </div>
              <input
                type="range"
                min={0.7}
                max={1.5}
                step={0.05}
                value={style.fontScale}
                onChange={(event) =>
                  setStyle((prev) => ({ ...prev, fontScale: Number(event.target.value) }))
                }
                className="mt-2 w-full cursor-pointer"
                aria-label="Font size"
              />
            </div>

            <div className="mt-4">
              <span className="label">Text color</span>
              <div className="flex flex-wrap items-center gap-2">
                {swatches.map((swatch) => (
                  <button
                    key={swatch.value}
                    type="button"
                    onClick={() => setStyle((prev) => ({ ...prev, color: swatch.value }))}
                    title={swatch.name}
                    aria-label={`${swatch.name} text`}
                    aria-pressed={style.color === swatch.value}
                    className={cn(
                      'size-7 cursor-pointer rounded-full border-2 transition hover:scale-110',
                      style.color === swatch.value
                        ? 'border-acid-400 ring-2 ring-acid-400/30'
                        : 'border-white/15',
                    )}
                    style={{ backgroundColor: swatch.value }}
                  />
                ))}
                <label
                  className="relative grid size-7 cursor-pointer place-items-center overflow-hidden rounded-full border-2 border-dashed border-white/20 text-zinc-500 transition hover:border-white/40 hover:text-white"
                  title="Custom color"
                >
                  <Plus className="pointer-events-none size-3.5" />
                  <input
                    type="color"
                    value={style.color}
                    onChange={(event) => setStyle((prev) => ({ ...prev, color: event.target.value }))}
                    className="absolute inset-0 cursor-pointer opacity-0"
                    aria-label="Custom text color"
                  />
                </label>
              </div>
            </div>

            <div className="mt-4 grid grid-cols-2 gap-2">
              <Toggle
                label="Outline"
                checked={style.outline}
                onChange={(checked) => setStyle((prev) => ({ ...prev, outline: checked }))}
              />
              <Toggle
                label="Uppercase"
                checked={style.uppercase}
                onChange={(checked) => setStyle((prev) => ({ ...prev, uppercase: checked }))}
              />
            </div>
          </section>

          <section className="panel p-5">
            <h2 className="text-sm font-bold text-white">Send it</h2>
            <div className="mt-4 space-y-3">
              <label className="block">
                <span className="label">Title (optional)</span>
                <input
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  maxLength={120}
                  placeholder="Give it a title"
                  className="input"
                />
              </label>
              <label className="block">
                <span className="label">Your name</span>
                <input
                  value={author}
                  onChange={(event) => setAuthor(event.target.value)}
                  maxLength={40}
                  placeholder="anon"
                  className="input"
                />
              </label>
            </div>

            {error && (
              <p className="mt-3 rounded-xl border border-hot-500/30 bg-hot-500/10 px-3 py-2 text-xs text-hot-300">
                {error}
              </p>
            )}

            <div className="mt-4 flex gap-2">
              <button
                type="button"
                onClick={() => void publish()}
                disabled={publishing || !image}
                className="btn-primary flex-1 py-3"
              >
                {publishing ? <Loader className="size-4" /> : <Sparkles className="size-4" />}
                {publishing ? 'Publishing…' : 'Publish to the board'}
              </button>
              <button
                type="button"
                onClick={() => void downloadDraft()}
                disabled={!image || publishing || gifProgress !== null}
                className="btn-secondary px-3.5"
                title="Download this meme instead"
                aria-label="Download this meme instead"
              >
                {gifProgress !== null ? <Loader className="size-4" /> : <Download className="size-4" />}
              </button>
            </div>

            {gifProgress !== null && (
              <p className="mt-2 text-center text-[10px] font-medium text-acid-300">
                Baking GIF… {Math.round(gifProgress * 100)}%
              </p>
            )}

            <p className="mt-3 text-[10px] leading-relaxed text-zinc-600">
              Posting is public and anonymous by default. Your browser gets one vote per meme — make
              it count.
            </p>
          </section>
        </div>
      </div>
    </div>
  )
}

function Toggle({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={cn(
        'flex cursor-pointer items-center justify-between rounded-xl border px-3 py-2.5 transition',
        checked
          ? 'border-acid-400/30 bg-acid-400/[0.07]'
          : 'border-white/10 bg-ink-900/60 hover:border-white/20',
      )}
    >
      <span className="text-xs font-medium text-zinc-300">{label}</span>
      <span
        className={cn(
          'relative h-5 w-9 rounded-full transition',
          checked ? 'bg-acid-400' : 'bg-white/10',
        )}
      >
        <span
          className={cn(
            'absolute top-0.5 left-0.5 size-4 rounded-full transition',
            checked ? 'translate-x-4 bg-ink-950' : 'bg-zinc-500',
          )}
        />
      </span>
    </button>
  )
}
