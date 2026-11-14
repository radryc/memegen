import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import type { Meme, SortMode, Stats, Template } from '../lib/types'
import { fetchLeaderboard, fetchMemes, fetchStats, fetchTemplates } from '../lib/api'
import { cn, formatCount, scoreTone } from '../lib/utils'
import MemeCard from '../components/MemeCard'
import { EmptyState, PageSkeleton } from '../components/UI'
import {
  ArrowRight,
  Dice,
  Flame,
  Loader,
  Plus,
  Search,
  Sparkles,
  Trophy,
} from '../components/Icons'

const sorts: { key: SortMode; label: string }[] = [
  { key: 'hot', label: 'Hot' },
  { key: 'new', label: 'New' },
  { key: 'top', label: 'Top' },
]

export default function FeedPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [sort, setSort] = useState<SortMode>('hot')
  const [query, setQuery] = useState(searchParams.get('q') ?? '')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [memes, setMemes] = useState<Meme[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState('')
  const [hasMore, setHasMore] = useState(false)
  const [nextOffset, setNextOffset] = useState(0)
  const [refreshKey, setRefreshKey] = useState(0)

  const [stats, setStats] = useState<Stats | null>(null)
  const [top, setTop] = useState<Meme[]>([])
  const [templates, setTemplates] = useState<Template[]>([])

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query.trim()), 300)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchMemes({ sort, q: debouncedQuery, limit: 32, offset: 0 })
      .then((data) => {
        if (cancelled) return
        setMemes(data.memes)
        setHasMore(data.hasMore)
        setNextOffset(data.nextOffset)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Could not load memes')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [sort, debouncedQuery, refreshKey])

  useEffect(() => {
    let cancelled = false
    fetchStats()
      .then((data) => !cancelled && setStats(data))
      .catch(() => undefined)
    fetchLeaderboard(5)
      .then((data) => !cancelled && setTop(data.memes))
      .catch(() => undefined)
    fetchTemplates()
      .then((data) => !cancelled && setTemplates(data.templates))
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [refreshKey])

  const applyMeme = (updated: Meme) => {
    const patch = (list: Meme[]) => list.map((m) => (m.id === updated.id ? updated : m))
    setMemes((prev) => sortForBoard(patch(prev), sort))
    setTop((prev) =>
      patch(prev)
        .sort((a, b) => b.score - a.score)
        .slice(0, 5),
    )
  }

  const loadMore = async () => {
    if (loadingMore || !hasMore) return
    setLoadingMore(true)
    try {
      const data = await fetchMemes({
        sort,
        q: debouncedQuery,
        limit: 32,
        offset: nextOffset,
      })
      setMemes((prev) => {
        const seen = new Set(prev.map((m) => m.id))
        return sortForBoard([...prev, ...data.memes.filter((m) => !seen.has(m.id))], sort)
      })
      setHasMore(data.hasMore)
      setNextOffset(data.nextOffset)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not load more memes')
    } finally {
      setLoadingMore(false)
    }
  }

  const roulette = () => {
    if (templates.length === 0) {
      navigate('/create')
      return
    }
    const pick = templates[Math.floor(Math.random() * templates.length)]
    navigate(`/create?template=${encodeURIComponent(pick.id)}`)
  }

  return (
    <div>
      <section className="mb-6 animate-rise">
        <div className="panel relative overflow-hidden px-5 py-5 sm:px-6 sm:py-6">
          <div className="pointer-events-none absolute -top-24 -left-20 size-64 rounded-full bg-acid-400/10 blur-[90px]" />
          <div className="pointer-events-none absolute -right-16 -bottom-24 size-56 rounded-full bg-hot-500/10 blur-[90px]" />

          <div className="relative flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
            <div className="min-w-0">
              <span className="chip border-acid-400/25 bg-acid-400/[0.07] text-acid-300">
                <span className="relative flex size-2">
                  <span className="absolute inline-flex size-full animate-ping rounded-full bg-acid-400 opacity-60" />
                  <span className="relative inline-flex size-2 rounded-full bg-acid-400" />
                </span>
                the internet&apos;s meme board
              </span>

              <h1 className="mt-3 font-display text-[clamp(2.2rem,4.6vw,4rem)] leading-[0.92] tracking-tight text-white uppercase">
                Make it. Post it.{' '}
                <span className="block bg-gradient-to-r from-acid-400 via-ice-400 to-hot-400 bg-clip-text text-transparent sm:inline">
                  Get roasted.
                </span>
              </h1>

              <p className="mt-2 max-w-2xl text-sm leading-relaxed text-zinc-400">
                Pick a template or upload your own image or GIF, slap on a caption, and drop it on
                the board. The crowd does the scoring — no accounts, no mercy.
              </p>
            </div>

            <div className="flex shrink-0 flex-col gap-4 lg:items-end">
              <div className="flex flex-wrap items-center gap-2.5">
                <Link to="/create" className="btn-primary px-5 py-2.5">
                  <Plus className="size-4" />
                  Start a meme
                </Link>
                <button type="button" onClick={roulette} className="btn-secondary px-5 py-2.5">
                  <Dice className="size-4 text-acid-400" />
                  Roulette
                </button>
              </div>

              <dl className="flex items-center gap-5 lg:justify-end">
                <Stat label="memes" value={stats ? formatCount(stats.memes) : '—'} />
                <span className="h-7 w-px bg-white/10" />
                <Stat label="votes" value={stats ? formatCount(stats.votes) : '—'} />
                <span className="h-7 w-px bg-white/10" />
                <Stat label="templates" value={stats ? formatCount(stats.templates) : '—'} />
              </dl>
            </div>
          </div>
        </div>
      </section>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_300px]">
        <div className="min-w-0">
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <h2 className="text-xl font-bold tracking-tight text-white">The board</h2>
              {stats && (
                <span className="hidden text-xs text-zinc-500 sm:inline">
                  {formatCount(stats.memes)} memes and counting
                </span>
              )}
            </div>

            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1 rounded-full border border-white/10 bg-ink-900/80 p-1">
                {sorts.map((item) => (
                  <button
                    key={item.key}
                    type="button"
                    onClick={() => setSort(item.key)}
                    className={cn(
                      'cursor-pointer rounded-full px-3.5 py-1.5 text-xs font-semibold transition',
                      sort === item.key
                        ? 'bg-acid-400 text-ink-950'
                        : 'text-zinc-400 hover:text-white',
                    )}
                  >
                    {item.label}
                  </button>
                ))}
              </div>

              <label className="relative flex-1 sm:w-52 sm:flex-none">
                <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-zinc-600" />
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="Search memes…"
                  aria-label="Search memes"
                  className="w-full rounded-full border border-white/10 bg-ink-900/80 py-2 pr-3 pl-9 text-xs text-zinc-200 transition placeholder:text-zinc-600 focus:border-acid-400/50 focus:outline-none"
                />
              </label>
            </div>
          </div>

          {error && (
            <div className="mb-5 flex items-center justify-between gap-4 rounded-2xl border border-hot-500/30 bg-hot-500/10 px-4 py-3 text-sm text-hot-300">
              <span>{error}</span>
              <button
                type="button"
                onClick={() => setRefreshKey((key) => key + 1)}
                className="shrink-0 font-semibold text-white underline underline-offset-4"
              >
                Retry
              </button>
            </div>
          )}

          {loading ? (
            <PageSkeleton />
          ) : memes.length === 0 ? (
            <EmptyState
              title={debouncedQuery ? 'No memes found' : 'The board is empty'}
              description={
                debouncedQuery
                  ? 'Nothing matches that search. Try a different caption or theme.'
                  : 'Be the first legend to post. Pick a template, add a caption, and claim the top spot.'
              }
              action={
                <Link to="/create" className="btn-primary">
                  <Sparkles className="size-4" />
                  {debouncedQuery ? 'Create something new' : 'Post the first meme'}
                </Link>
              }
            />
          ) : (
            <>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
                {memes.map((meme, index) => (
                  <MemeCard
                    key={meme.id}
                    meme={meme}
                    onVote={applyMeme}
                    rank={sort === 'top' ? index + 1 : undefined}
                    layout="grid"
                  />
                ))}
              </div>

              {hasMore && (
                <div className="mt-6 flex justify-center">
                  <button
                    type="button"
                    onClick={loadMore}
                    disabled={loadingMore}
                    className="btn-secondary"
                  >
                    {loadingMore ? (
                      <Loader className="size-4" />
                    ) : (
                      <ArrowRight className="size-4 rotate-90" />
                    )}
                    {loadingMore ? 'Loading…' : 'Load more memes'}
                  </button>
                </div>
              )}
            </>
          )}
        </div>

        <aside className="hidden self-start xl:sticky xl:top-24 xl:block">
          <div className="space-y-4">
            <div className="panel p-5">
              <div className="flex items-center justify-between">
                <h3 className="flex items-center gap-2 text-sm font-bold text-white">
                  <Trophy className="size-4 text-gold-400" />
                  Hall of fame
                </h3>
                <Link
                  to="/top"
                  className="text-[11px] font-semibold text-zinc-500 transition hover:text-acid-400"
                >
                  View all
                </Link>
              </div>

              <div className="mt-4 space-y-3">
                {top.length === 0
                  ? [0, 1, 2].map((i) => <div key={i} className="skeleton h-10 w-full" />)
                  : top.map((meme, index) => (
                      <Link
                        key={meme.id}
                        to={`/m/${meme.id}`}
                        className="group flex items-center gap-3"
                      >
                        <span className="w-3 shrink-0 text-center font-display text-sm text-zinc-600">
                          {index + 1}
                        </span>
                        <img
                          src={meme.imageUrl}
                          alt=""
                          loading="lazy"
                          className="size-10 shrink-0 rounded-lg border border-white/10 object-cover"
                        />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-xs font-medium text-zinc-300 transition group-hover:text-white">
                            {meme.title || meme.topText || 'Untitled banger'}
                          </span>
                          <span className="block text-[10px] text-zinc-600">
                            by {meme.author}
                          </span>
                        </span>
                        <span
                          className={cn(
                            'inline-flex items-center gap-0.5 text-xs font-bold',
                            scoreTone(meme.score),
                          )}
                        >
                          <Flame className="size-3" />
                          {meme.score}
                        </span>
                      </Link>
                    ))}
              </div>
            </div>

            <div className="relative overflow-hidden rounded-3xl border border-acid-400/25 bg-gradient-to-br from-acid-400/12 via-ink-850 to-hot-500/10 p-5">
              <div className="absolute -top-10 -right-8 size-28 rounded-full bg-hot-500/20 blur-2xl" />
              <Dice className="size-6 text-acid-400" />
              <h3 className="mt-3 text-sm font-bold text-white">Can&apos;t pick a template?</h3>
              <p className="mt-1.5 text-xs leading-relaxed text-zinc-400">
                Let the dice decide. You bring the caption, chaos does the rest.
              </p>
              <button
                type="button"
                onClick={roulette}
                className="btn-primary mt-4 w-full py-2 text-xs"
              >
                <Dice className="size-3.5" />
                Roll the dice
              </button>
            </div>

            <div className="panel p-5">
              <h3 className="flex items-center gap-2 text-sm font-bold text-white">
                <Sparkles className="size-4 text-ice-400" />
                How scoring works
              </h3>
              <ul className="mt-3 space-y-2.5 text-xs leading-relaxed text-zinc-500">
                <li className="flex gap-2">
                  <span className="mt-0.5 text-acid-400">→</span>
                  Upvote what hits. Downvote what misses.
                </li>
                <li className="flex gap-2">
                  <span className="mt-0.5 text-acid-400">→</span>
                  Score = upvotes minus downvotes. One vote per meme per visitor.
                </li>
                <li className="flex gap-2">
                  <span className="mt-0.5 text-acid-400">→</span>
                  Hot blends score with recency so fresh posts get their shot.
                </li>
              </ul>
            </div>

            <Link to="/templates" className="btn-secondary w-full py-2.5 text-xs">
              Browse all {templates.length > 0 ? templates.length : ''} templates
              <ArrowRight className="size-3.5" />
            </Link>
          </div>
        </aside>
      </div>
    </div>
  )
}

function hotScore(meme: Meme): number {
  const ageHours = Math.max(0, (Date.now() - new Date(meme.createdAt).getTime()) / 3_600_000)
  return (meme.score + 1) / Math.pow(ageHours + 2, 1.35)
}

function sortForBoard(memes: Meme[], sort: SortMode): Meme[] {
  if (sort === 'top') {
    return [...memes].sort(
      (a, b) => b.score - a.score || new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    )
  }
  if (sort === 'hot') {
    return [...memes].sort((a, b) => hotScore(b) - hotScore(a))
  }
  return memes
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[10px] tracking-[0.14em] text-zinc-600 uppercase">{label}</dt>
      <dd className="mt-0.5 font-display text-xl tracking-wide text-white">{value}</dd>
    </div>
  )
}
