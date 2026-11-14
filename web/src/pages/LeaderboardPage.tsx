import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import type { Meme } from '../lib/types'
import { fetchMemes } from '../lib/api'
import { cn, formatTime, remixHref } from '../lib/utils'
import { VoteButtons } from '../components/VoteButtons'
import { EmptyState, PageSkeleton } from '../components/UI'
import { ArrowRight, Flame, Remix, Sparkles, Trophy } from '../components/Icons'

const medals = [
  {
    label: 'Champion',
    badge: 'bg-gold-400 text-ink-950',
    border: 'border-gold-400/40',
    glow: 'shadow-[0_30px_80px_-40px_rgb(255_209_102/0.55)]',
  },
  {
    label: 'Runner-up',
    badge: 'bg-zinc-300 text-ink-950',
    border: 'border-white/20',
    glow: '',
  },
  {
    label: 'Third place',
    badge: 'bg-[#cd7f32] text-ink-950',
    border: 'border-[#cd7f32]/40',
    glow: '',
  },
]

export default function LeaderboardPage() {
  const [memes, setMemes] = useState<Meme[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    fetchMemes({ sort: 'top', limit: 40 })
      .then((data) => {
        if (!cancelled) setMemes(data.memes)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Could not load the ranking')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const handleVote = (updated: Meme) => {
    setMemes((prev) =>
      prev
        .map((meme) => (meme.id === updated.id ? updated : meme))
        .sort(
          (a, b) =>
            b.score - a.score || new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
        ),
    )
  }

  const podium = memes.slice(0, 3)
  const rest = memes.slice(3)
  const totalScore = memes.reduce((sum, meme) => sum + meme.score, 0)

  return (
    <div className="pb-10">
      <div className="flex flex-col gap-4 pt-4 pb-8 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <span className="chip border-gold-400/25 bg-gold-400/[0.07] text-gold-400">
            <Trophy className="size-3.5" />
            all-time ranking
          </span>
          <h1 className="mt-3 font-display text-4xl tracking-tight text-white uppercase sm:text-5xl">
            Hall of fame
          </h1>
          <p className="mt-2 max-w-lg text-sm leading-relaxed text-zinc-500">
            The highest-scoring memes on the board. No algorithms, no favors — just votes.
          </p>
        </div>

        <div className="flex items-center gap-3 text-xs text-zinc-500">
          <span className="chip">
            <Sparkles className="size-3.5 text-acid-400" />
            {memes.length} ranked
          </span>
          <span className="chip">
            <Flame className="size-3.5 text-hot-400" />
            {totalScore} combined score
          </span>
        </div>
      </div>

      {error && (
        <div className="mb-5 rounded-2xl border border-hot-500/30 bg-hot-500/10 px-4 py-3 text-sm text-hot-300">
          {error}
        </div>
      )}

      {loading ? (
        <PageSkeleton />
      ) : memes.length === 0 ? (
        <EmptyState
          title="The podium is empty"
          description="No memes have been posted yet. Be the first to claim the crown."
          action={
            <Link to="/create" className="btn-primary">
              <Sparkles className="size-4" />
              Post the first meme
            </Link>
          }
        />
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-3 md:items-start">
            {podium.map((meme, index) => (
              <PodiumCard
                key={meme.id}
                meme={meme}
                index={index}
                onVote={handleVote}
                className={
                  index === 0
                    ? 'md:order-2 md:-mt-5'
                    : index === 1
                      ? 'md:order-1 md:mt-6'
                      : 'md:order-3 md:mt-6'
                }
              />
            ))}
          </div>

          {rest.length > 0 && (
            <div className="panel mt-6 overflow-hidden">
              <div className="hidden grid-cols-[2.5rem_minmax(0,1fr)_5rem_7rem_2rem] items-center gap-3 border-b border-white/[0.07] px-4 py-2.5 text-[10px] font-semibold tracking-[0.16em] text-zinc-600 uppercase sm:grid">
                <span>Rank</span>
                <span>Meme</span>
                <span className="text-right">Votes</span>
                <span className="text-center">Vote</span>
                <span />
              </div>

              {rest.map((meme, index) => (
                <div
                  key={meme.id}
                  className="grid grid-cols-[1.75rem_minmax(0,1fr)_auto] items-center gap-3 border-b border-white/[0.05] px-4 py-3 last:border-0 hover:bg-white/[0.02] sm:grid-cols-[2.5rem_minmax(0,1fr)_5rem_7rem_2rem]"
                >
                  <span className="text-center font-display text-lg text-zinc-600">
                    {index + 4}
                  </span>

                  <div className="flex min-w-0 items-center gap-3">
                    <Link to={`/m/${meme.id}`} className="shrink-0">
                      <img
                        src={meme.imageUrl}
                        alt=""
                        loading="lazy"
                        className="size-11 rounded-lg border border-white/10 object-cover"
                      />
                    </Link>
                    <div className="min-w-0">
                      <Link
                        to={`/m/${meme.id}`}
                        className="block truncate text-sm font-medium text-zinc-200 transition hover:text-acid-400"
                      >
                        {meme.title || meme.topText || 'Untitled banger'}
                      </Link>
                      <span className="block truncate text-[11px] text-zinc-600">
                        by {meme.author} · {formatTime(meme.createdAt)}
                      </span>
                    </div>
                  </div>

                  <span className="hidden text-right text-xs text-zinc-500 sm:block">
                    {meme.upvotes}▲ {meme.downvotes}▼
                  </span>

                  <div className="flex justify-start sm:justify-center">
                    <VoteButtons meme={meme} onChange={(update) => handleVote({ ...meme, ...update })} />
                  </div>

                  <div className="hidden justify-self-end sm:block">
                    <Link
                      to={`/m/${meme.id}`}
                      className="btn-icon size-7"
                      title="Open meme"
                      aria-label="Open meme"
                    >
                      <ArrowRight className="size-3.5" />
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}

function PodiumCard({
  meme,
  index,
  onVote,
  className,
}: {
  meme: Meme
  index: number
  onVote: (meme: Meme) => void
  className?: string
}) {
  const medal = medals[index] ?? medals[2]

  return (
    <article
      className={cn(
        'group relative overflow-hidden rounded-3xl border bg-ink-850/80 transition duration-300 hover:-translate-y-0.5',
        medal.border,
        medal.glow,
        className,
      )}
    >
      <div className="flex items-center justify-between gap-2 border-b border-white/[0.07] px-3 py-2">
        <span
          className={cn(
            'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[10px] font-bold tracking-wide uppercase',
            medal.badge,
          )}
        >
          <Trophy className="size-3" />
          {medal.label}
        </span>
        <span className="font-display text-sm text-zinc-600">#{index + 1}</span>
      </div>
      <Link to={`/m/${meme.id}`} className="relative block overflow-hidden bg-ink-900">
        <img
          src={meme.imageUrl}
          alt={meme.title || 'Meme'}
          className="w-full transition duration-500 group-hover:scale-[1.015]"
        />
      </Link>

      <div className="p-4">
        <Link
          to={`/m/${meme.id}`}
          className="line-clamp-1 text-sm font-bold text-white transition hover:text-acid-400"
        >
          {meme.title || meme.topText || 'Untitled banger'}
        </Link>
        <p className="mt-1 text-[11px] text-zinc-500">
          by {meme.author} · {formatTime(meme.createdAt)}
        </p>

        <div className="mt-3 flex items-center justify-between gap-2 border-t border-white/[0.06] pt-3">
          <VoteButtons meme={meme} onChange={(update) => onVote({ ...meme, ...update })} />
          <Link
            to={remixHref(meme)}
            className="btn-icon size-7"
            title="Remix this meme"
            aria-label="Remix this meme"
          >
            <Remix className="size-3" />
          </Link>
        </div>
      </div>
    </article>
  )
}
