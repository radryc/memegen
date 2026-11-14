import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'
import type { Meme, Template } from '../lib/types'
import { fetchMemes, fetchTemplates } from '../lib/api'
import { cn } from '../lib/utils'
import MemeCard from '../components/MemeCard'
import TemplateCard from '../components/TemplateCard'
import { EmptyState } from '../components/UI'
import { ArrowRight, Flame, Search, Sparkles, X } from '../components/Icons'

type Tab = 'all' | 'memes' | 'templates'

const suggestions = ['drake', 'cat', 'code', 'monday', 'production', 'work']

export default function SearchPage() {
  const [params, setParams] = useSearchParams()
  const [query, setQuery] = useState(params.get('q') ?? '')
  const [debounced, setDebounced] = useState(query.trim())
  const [tab, setTab] = useState<Tab>('all')
  const [memes, setMemes] = useState<Meme[]>([])
  const [templates, setTemplates] = useState<Template[]>([])
  const [trending, setTrending] = useState<Meme[]>([])
  const [popular, setPopular] = useState<Template[]>([])
  const [loading, setLoading] = useState(false)
  const [landingLoading, setLandingLoading] = useState(true)
  const [error, setError] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 300)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== '/' || event.metaKey || event.ctrlKey || event.altKey) return
      const target = event.target as HTMLElement | null
      if (
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.tagName === 'SELECT' ||
          target.isContentEditable)
      ) {
        return
      }
      event.preventDefault()
      inputRef.current?.focus()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  useEffect(() => {
    const next = new URLSearchParams()
    if (debounced) next.set('q', debounced)
    setParams(next, { replace: true })
  }, [debounced, setParams])

  useEffect(() => {
    if (!debounced) {
      setMemes([])
      setTemplates([])
      setError('')
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setError('')
    Promise.all([
      fetchMemes({ q: debounced, sort: 'top', limit: 48 }),
      fetchTemplates({ q: debounced }),
    ])
      .then(([memeData, templateData]) => {
        if (cancelled) return
        setMemes(memeData.memes)
        setTemplates(templateData.templates)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Search failed')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [debounced])

  useEffect(() => {
    let cancelled = false
    Promise.all([fetchMemes({ sort: 'hot', limit: 6 }), fetchTemplates()])
      .then(([memeData, templateData]) => {
        if (cancelled) return
        setTrending(memeData.memes)
        setPopular(templateData.templates.filter((t) => !t.animated).slice(0, 8))
      })
      .catch(() => undefined)
      .finally(() => {
        if (!cancelled) setLandingLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const applyMeme = (updated: Meme) => {
    const patch = (list: Meme[]) =>
      list
        .map((item) => (item.id === updated.id ? updated : item))
        .sort((a, b) => b.score - a.score)
    setMemes(patch)
    setTrending(patch)
  }

  const hasQuery = debounced.length > 0
  const total = memes.length + templates.length
  const tabs: { key: Tab; label: string; count: number }[] = [
    { key: 'all', label: 'Everything', count: total },
    { key: 'memes', label: 'Memes', count: memes.length },
    { key: 'templates', label: 'Templates', count: templates.length },
  ]

  const showMemes = tab === 'all' ? memes.slice(0, 6) : tab === 'memes' ? memes : []
  const showTemplates =
    tab === 'all' ? templates.slice(0, 8) : tab === 'templates' ? templates : []
  const activeHasResults =
    tab === 'all' ? total > 0 : tab === 'memes' ? memes.length > 0 : templates.length > 0

  return (
    <div className="pb-10">
      <section className="animate-rise pt-4 pb-8 text-center sm:pt-10">
        <span className="chip border-ice-400/25 bg-ice-400/[0.07] text-ice-400">
          <Search className="size-3.5" />
          search engine
        </span>
        <h1 className="mt-4 font-display text-4xl tracking-tight text-white uppercase sm:text-6xl">
          Find the good stuff.
        </h1>
        <p className="mx-auto mt-3 max-w-lg text-sm leading-relaxed text-zinc-500">
          Every meme, caption, template and creator on the board — one search box.
        </p>

        <div className="mx-auto mt-7 max-w-2xl">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-5 size-5 -translate-y-1/2 text-zinc-500" />
            <input
              ref={inputRef}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search memes, captions, templates, creators…"
              aria-label="Search memes and templates"
              className="w-full rounded-2xl border border-white/12 bg-ink-900/90 py-4 pr-28 pl-14 text-sm text-zinc-100 shadow-[0_20px_60px_-30px_rgb(0,0,0/0.9)] transition placeholder:text-zinc-600 focus:border-acid-400/60 focus:outline-none"
            />
            {query && (
              <button
                type="button"
                onClick={() => {
                  setQuery('')
                  inputRef.current?.focus()
                }}
                className="absolute top-1/2 right-14 grid size-7 -translate-y-1/2 cursor-pointer place-items-center rounded-full text-zinc-500 transition hover:bg-white/10 hover:text-white"
                aria-label="Clear search"
              >
                <X className="size-4" />
              </button>
            )}
            <kbd className="pointer-events-none absolute top-1/2 right-4 -translate-y-1/2 rounded-md border border-white/10 bg-white/[0.04] px-1.5 py-0.5 text-[10px] font-semibold text-zinc-600">
              /
            </kbd>
          </div>

          <div className="mt-3 flex flex-wrap items-center justify-center gap-1.5 text-[11px]">
            <span className="text-zinc-600">Try</span>
            {suggestions.map((suggestion) => (
              <button
                key={suggestion}
                type="button"
                onClick={() => {
                  setQuery(suggestion)
                  inputRef.current?.focus()
                }}
                className={cn(
                  'cursor-pointer rounded-full border px-2.5 py-1 font-medium transition',
                  suggestion === debounced
                    ? 'border-acid-400/40 bg-acid-400/10 text-acid-300'
                    : 'border-white/10 bg-white/[0.03] text-zinc-400 hover:border-white/25 hover:text-white',
                )}
              >
                {suggestion}
              </button>
            ))}
          </div>
        </div>
      </section>

      {hasQuery && (
        <div className="mx-auto mb-6 flex w-fit max-w-full flex-wrap items-center justify-center gap-1 rounded-full border border-white/10 bg-ink-900/70 p-1">
          {tabs.map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={() => setTab(item.key)}
              className={cn(
                'flex cursor-pointer items-center gap-1.5 rounded-full px-3.5 py-1.5 text-xs font-semibold transition',
                tab === item.key ? 'bg-acid-400 text-ink-950' : 'text-zinc-400 hover:text-white',
              )}
            >
              {item.label}
              <span
                className={cn(
                  'rounded-full px-1.5 py-0.5 text-[10px] tabular-nums',
                  tab === item.key ? 'bg-ink-950/15' : 'bg-white/[0.06]',
                )}
              >
                {loading ? '…' : item.count}
              </span>
            </button>
          ))}
        </div>
      )}

      {error && (
        <div className="mb-5 rounded-2xl border border-hot-500/30 bg-hot-500/10 px-4 py-3 text-center text-sm text-hot-300">
          {error}
        </div>
      )}

      {hasQuery && loading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, index) => (
            <div
              key={index}
              className={cn('skeleton rounded-3xl', index % 2 === 0 ? 'h-72' : 'h-96')}
            />
          ))}
        </div>
      ) : hasQuery && !activeHasResults ? (
        <EmptyState
          title={
            total === 0
              ? `No results for “${debounced}”`
              : `No ${tab === 'memes' ? 'memes' : 'templates'} matched`
          }
          description={
            total === 0
              ? 'Try a shorter caption, a different spelling, or browse the board for inspiration.'
              : 'There are matches in the other tab. Take a look there instead.'
          }
          action={
            total === 0 ? (
              <button type="button" onClick={() => setQuery('')} className="btn-primary">
                <Sparkles className="size-4" />
                Clear search
              </button>
            ) : (
              <button type="button" onClick={() => setTab('all')} className="btn-primary">
                <ArrowRight className="size-4" />
                See all results
              </button>
            )
          }
        />
      ) : hasQuery ? (
        <div className="space-y-10">
          {showMemes.length > 0 && (
            <section>
              <SectionHeader
                icon={<Flame className="size-4 text-hot-400" />}
                title="Memes"
                count={memes.length}
                showAll={tab === 'all' && memes.length > 6}
                onAll={() => setTab('memes')}
              />
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
                {showMemes.map((meme) => (
                  <MemeCard key={meme.id} meme={meme} onVote={applyMeme} layout="grid" />
                ))}
              </div>
            </section>
          )}

          {showTemplates.length > 0 && (
            <section>
              <SectionHeader
                icon={<Sparkles className="size-4 text-acid-400" />}
                title="Templates"
                count={templates.length}
                showAll={tab === 'all' && templates.length > 8}
                onAll={() => setTab('templates')}
              />
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-5 grid-wide">
                {showTemplates.map((template) => (
                  <TemplateCard key={template.id} template={template} />
                ))}
              </div>
            </section>
          )}
        </div>
      ) : (
        <div className="space-y-10">
          <section>
            <SectionHeader
              icon={<Flame className="size-4 text-hot-400" />}
              title="Trending right now"
              count={trending.length}
            />
            {landingLoading ? (
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {[0, 1, 2].map((index) => (
                  <div key={index} className="skeleton h-72 rounded-3xl" />
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
                {trending.map((meme) => (
                  <MemeCard key={meme.id} meme={meme} onVote={applyMeme} layout="grid" />
                ))}
              </div>
            )}
          </section>

          <section>
            <SectionHeader
              icon={<Sparkles className="size-4 text-acid-400" />}
              title="Start from a classic"
              count={popular.length}
            />
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-5 grid-wide">
              {popular.map((template) => (
                <TemplateCard key={template.id} template={template} />
              ))}
            </div>
          </section>
        </div>
      )}
    </div>
  )
}

function SectionHeader({
  icon,
  title,
  count,
  showAll,
  onAll,
}: {
  icon: ReactNode
  title: string
  count: number
  showAll?: boolean
  onAll?: () => void
}) {
  return (
    <div className="mb-4 flex items-center justify-between gap-3">
      <h2 className="flex items-center gap-2 text-lg font-bold tracking-tight text-white">
        {icon}
        {title}
        <span className="text-xs font-medium text-zinc-600">{count}</span>
      </h2>
      {showAll && (
        <button
          type="button"
          onClick={onAll}
          className="inline-flex cursor-pointer items-center gap-1 text-xs font-medium text-zinc-500 transition hover:text-acid-400"
        >
          See all
          <ArrowRight className="size-3" />
        </button>
      )}
    </div>
  )
}
