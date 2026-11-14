import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import type { Meme } from '../lib/types'
import { fetchMeme, fetchMemes } from '../lib/api'
import { cn, formatTime, initials, remixHref, scoreTone } from '../lib/utils'
import MemeCard, { memeTitle } from '../components/MemeCard'
import { VoteButtons, useVote, type VoteState } from '../components/VoteButtons'
import { EmptyState } from '../components/UI'
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Download,
  Flame,
  ImageIcon,
  LinkIcon,
  Loader,
  Remix,
  Sparkles,
  Trophy,
} from '../components/Icons'

export default function MemePage() {
  const { id = '' } = useParams()
  const location = useLocation()
  const [meme, setMeme] = useState<Meme | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchMeme(id)
      .then((data) => {
        if (!cancelled) setMeme(data.meme)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Could not load this meme')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [id])

  const justPublished = Boolean(
    (location.state as { justPublished?: boolean } | null)?.justPublished,
  )

  return (
    <div className="pb-12">
      <Link
        to="/"
        className="inline-flex items-center gap-1.5 text-xs font-medium text-zinc-500 transition hover:text-zinc-200"
      >
        <ArrowLeft className="size-3.5" />
        Back to the board
      </Link>

      {loading ? (
        <div className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className="skeleton aspect-[4/3] rounded-3xl" />
          <div className="space-y-4">
            <div className="skeleton h-8 w-3/4 rounded-full" />
            <div className="skeleton h-4 w-1/3 rounded-full" />
            <div className="skeleton h-36 w-full rounded-3xl" />
            <div className="skeleton h-52 w-full rounded-3xl" />
          </div>
        </div>
      ) : error || !meme ? (
        <div className="mt-8">
          <EmptyState
            title="This meme got away"
            description={error || 'It may have been deleted, or the link is wrong.'}
            action={
              <Link to="/" className="btn-primary">
                Back to the board
              </Link>
            }
          />
        </div>
      ) : (
        <>
          {justPublished && (
            <div className="mt-5 flex animate-pop items-center gap-3 rounded-2xl border border-acid-400/30 bg-acid-400/10 px-4 py-3">
              <span className="grid size-9 shrink-0 place-items-center rounded-full bg-acid-400 text-ink-950">
                <Check className="size-4" />
              </span>
              <div>
                <p className="text-sm font-bold text-white">Your meme is live.</p>
                <p className="text-xs text-acid-200/70">
                  Now let the scoring begin. Share the link and collect the glory.
                </p>
              </div>
            </div>
          )}

          <MemeView meme={meme} onUpdate={setMeme} />
        </>
      )}
    </div>
  )
}

function MemeView({ meme, onUpdate }: { meme: Meme; onUpdate: (meme: Meme) => void }) {
  const { state: vote, error: voteError } = useVote(meme, (update: VoteState) =>
    onUpdate({ ...meme, ...update }),
  )
  const [related, setRelated] = useState<Meme[]>([])
  const [copied, setCopied] = useState(false)
  const [imageLoaded, setImageLoaded] = useState(false)
  const [imageFailed, setImageFailed] = useState(false)
  const [imageAttempt, setImageAttempt] = useState(0)
  const imageRef = useRef<HTMLImageElement>(null)

  useEffect(() => {
    setImageLoaded(false)
    setImageFailed(false)
    setImageAttempt(0)
  }, [meme.id])

  useEffect(() => {
    // A cached image can finish before React attaches onLoad. Check the DOM
    // directly so the page never gets stuck on the rendering spinner.
    const img = imageRef.current
    if (img?.complete && img.naturalWidth > 0) {
      setImageLoaded(true)
      setImageFailed(false)
    }
  }, [meme.id, imageAttempt])

  useEffect(() => {
    let cancelled = false
    fetchMemes({ template: meme.templateId, sort: 'top', limit: 8 })
      .then((data) => {
        if (cancelled) return
        setRelated(data.memes.filter((item) => item.id !== meme.id).slice(0, 3))
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [meme.id, meme.templateId])

  useEffect(() => {
    if (!copied) return
    const timer = setTimeout(() => setCopied(false), 2000)
    return () => clearTimeout(timer)
  }, [copied])

  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href)
      setCopied(true)
    } catch {
      setCopied(false)
    }
  }

  const totalVotes = vote.upvotes + vote.downvotes
  const upPercent = totalVotes > 0 ? Math.round((vote.upvotes / totalVotes) * 100) : 50

  const handleRelatedVote = (updated: Meme) => {
    setRelated((prev) =>
      prev
        .map((item) => (item.id === updated.id ? updated : item))
        .sort((a, b) => b.score - a.score),
    )
  }

  return (
    <>
      <div className="mt-5 grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <div>
          <div className="panel overflow-hidden">
            <div className="checker relative flex min-h-64 items-center justify-center p-3 sm:p-5">
              <img
                key={`${meme.id}-${imageAttempt}`}
                ref={imageRef}
                src={`${meme.imageUrl}${imageAttempt > 0 ? `?retry=${imageAttempt}` : ''}`}
                alt={memeTitle(meme)}
                onLoad={() => {
                  setImageLoaded(true)
                  setImageFailed(false)
                }}
                onError={() => {
                  setImageLoaded(false)
                  setImageFailed(true)
                }}
                className={cn(
                  'mx-auto block h-auto max-h-[60vh] w-auto max-w-full rounded-xl transition-opacity duration-300',
                  imageLoaded ? 'opacity-100' : 'opacity-0',
                )}
              />
              {!imageLoaded && !imageFailed && (
                <div className="absolute inset-0 grid place-items-center">
                  <div className="flex flex-col items-center gap-2 text-[11px] text-zinc-500">
                    <Loader className="size-6 text-acid-400" />
                    Rendering meme…
                  </div>
                </div>
              )}
              {imageFailed && (
                <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 px-6 text-center">
                  <p className="text-sm font-medium text-zinc-300">This meme could not be rendered.</p>
                  <p className="max-w-sm text-[11px] text-zinc-500">
                    The server may have been restarting. Try again, or refresh the page.
                  </p>
                  <button
                    type="button"
                    onClick={() => {
                      setImageFailed(false)
                      setImageLoaded(false)
                      setImageAttempt((attempt) => attempt + 1)
                    }}
                    className="btn-secondary px-4 py-2 text-xs"
                  >
                    Retry rendering
                  </button>
                </div>
              )}
            </div>

            <div className="flex flex-wrap items-center gap-3 border-t border-white/[0.07] px-4 py-3">
              <VoteButtons
                meme={meme}
                size="lg"
                onChange={(update) => onUpdate({ ...meme, ...update })}
              />

              <div className="ml-auto flex items-center gap-2">
                <a
                  href={meme.imageUrl}
                  download={`memebank-${meme.id}.${meme.animated ? 'gif' : 'png'}`}
                  className="btn-icon"
                  title="Download PNG"
                  aria-label="Download PNG"
                >
                  <Download className="size-4" />
                </a>
                <button
                  type="button"
                  onClick={() => void copyLink()}
                  className={cn('btn-icon', copied && 'border-acid-400/40 text-acid-400')}
                  title="Copy link"
                  aria-label="Copy link"
                >
                  {copied ? <Check className="size-4" /> : <LinkIcon className="size-4" />}
                </button>
                <Link to={remixHref(meme)} className="btn-secondary px-4 py-2 text-xs">
                  <Remix className="size-3.5" />
                  Remix
                </Link>
              </div>
            </div>
          </div>

          {voteError && <p className="mt-2 text-xs text-hot-300">{voteError}</p>}

          <div className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-white/[0.07] bg-ink-850/50 px-4 py-3 text-xs text-zinc-500">
            <span className="flex items-center gap-1.5">
              <Sparkles className="size-3.5 text-acid-400" />
              Posted {formatTime(meme.createdAt)}
            </span>
            <span className="flex items-center gap-1.5">
              <Flame className="size-3.5 text-hot-400" />
              Hot ranking blends score with recency
            </span>
          </div>
        </div>

        <aside className="space-y-4 lg:sticky lg:top-24 lg:self-start">
          <div>
            <h1 className="text-2xl leading-tight font-bold tracking-tight text-white sm:text-3xl">
              {memeTitle(meme)}
            </h1>
            <div className="mt-3 flex items-center gap-2 text-sm text-zinc-500">
              <span className="grid size-8 place-items-center rounded-full bg-gradient-to-br from-acid-400/25 to-ice-400/20 text-[11px] font-bold text-acid-300">
                {initials(meme.author)}
              </span>
              <span>
                by <span className="font-medium text-zinc-300">{meme.author}</span>
              </span>
              <span className="text-zinc-700">·</span>
              <span className="text-xs">{formatTime(meme.createdAt)}</span>
              {meme.animated && (
                <span className="rounded-full border border-hot-400/30 bg-hot-500/10 px-2 py-0.5 text-[9px] font-bold tracking-wider text-hot-300">
                  GIF
                </span>
              )}
            </div>
          </div>

          <div className="panel p-5">
            <div className="flex items-end justify-between">
              <div>
                <span className="label mb-0">Meme score</span>
                <p className={cn('mt-1 font-display text-5xl leading-none', scoreTone(vote.score))}>
                  {vote.score > 0 ? `+${vote.score}` : vote.score}
                </p>
              </div>
              <span className="chip border-white/10 bg-white/[0.04]">
                <Trophy className="size-3.5 text-gold-400" />
                {totalVotes} vote{totalVotes === 1 ? '' : 's'}
              </span>
            </div>

            <div className="mt-4 h-2 overflow-hidden rounded-full bg-hot-500/60">
              <div
                className="h-full rounded-full bg-gradient-to-r from-acid-400 to-ice-400 transition-all duration-500"
                style={{ width: `${upPercent}%` }}
              />
            </div>
            <div className="mt-2 flex items-center justify-between text-[11px] font-medium">
              <span className="text-acid-400">{vote.upvotes} upvotes</span>
              <span className="text-hot-300">{vote.downvotes} downvotes</span>
            </div>

            <p className="mt-4 text-[11px] leading-relaxed text-zinc-600">
              One vote per visitor. Hit the same arrow again to take your vote back.
            </p>
          </div>

          <div className="panel space-y-4 p-5 text-sm">
            {meme.templateName && (
              <div>
                <span className="label">Made with</span>
                <Link
                  to={meme.templateId ? `/create?template=${encodeURIComponent(meme.templateId)}` : '/templates'}
                  className="inline-flex items-center gap-2 rounded-xl border border-white/10 bg-ink-900/70 px-3 py-2 text-xs font-medium text-zinc-300 transition hover:border-acid-400/40 hover:text-white"
                >
                  <span className="grid size-8 place-items-center rounded-md bg-white/[0.05] text-zinc-500">
                    <ImageIcon className="size-4" />
                  </span>
                  {meme.templateName}
                  <ArrowRight className="size-3 text-zinc-500" />
                </Link>
              </div>
            )}

            <div>
              <span className="label">Caption</span>
              <div className="space-y-2">
                {meme.topText && (
                  <p className="rounded-xl border-l-2 border-acid-400/50 bg-white/[0.03] px-3 py-2 text-xs text-zinc-300">
                    {meme.topText}
                  </p>
                )}
                {meme.bottomText && (
                  <p className="rounded-xl border-l-2 border-hot-400/50 bg-white/[0.03] px-3 py-2 text-xs text-zinc-300">
                    {meme.bottomText}
                  </p>
                )}
                {!meme.topText && !meme.bottomText && (
                  <p className="text-xs text-zinc-600">No caption — pure vibes.</p>
                )}
              </div>
            </div>
          </div>

          <div className="relative overflow-hidden rounded-3xl border border-acid-400/25 bg-gradient-to-br from-acid-400/12 via-ink-850 to-hot-500/10 p-5">
            <div className="absolute -top-8 -right-6 size-24 rounded-full bg-acid-400/20 blur-2xl" />
            <h3 className="text-sm font-bold text-white">Think you can beat it?</h3>
            <p className="mt-1 text-xs leading-relaxed text-zinc-400">
              Same template, better caption. The board is waiting.
            </p>
            <Link
              to={meme.templateId ? `/create?template=${encodeURIComponent(meme.templateId)}` : '/create'}
              className="btn-primary mt-4 w-full py-2.5 text-xs"
            >
              <Sparkles className="size-3.5" />
              Make your own
            </Link>
          </div>
        </aside>
      </div>

      {related.length > 0 && (
        <section className="mt-12">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-bold tracking-tight text-white">More from this template</h2>
            <Link
              to="/"
              className="inline-flex items-center gap-1 text-xs font-medium text-zinc-500 transition hover:text-acid-400"
            >
              See the feed
              <ArrowRight className="size-3" />
            </Link>
          </div>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {related.map((item) => (
              <MemeCard key={item.id} meme={item} onVote={handleRelatedVote} layout="grid" />
            ))}
          </div>
        </section>
      )}
    </>
  )
}
