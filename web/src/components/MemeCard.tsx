import { Link } from 'react-router-dom'
import type { Meme } from '../lib/types'
import { cn, formatTime, initials, remixHref } from '../lib/utils'
import { Remix } from './Icons'
import { VoteButtons, type VoteState } from './VoteButtons'

function AuthorBadge({ name }: { name: string }) {
  return (
    <span className="grid size-5 shrink-0 place-items-center rounded-full bg-gradient-to-br from-acid-400/25 to-ice-400/20 text-[9px] font-bold text-acid-300">
      {initials(name)}
    </span>
  )
}

export function memeTitle(meme: Meme): string {
  return meme.title.trim() || meme.topText.trim() || 'Untitled banger'
}

export default function MemeCard({
  meme,
  onVote,
  rank,
  layout = 'masonry',
}: {
  meme: Meme
  onVote?: (meme: Meme) => void
  rank?: number
  layout?: 'masonry' | 'grid'
}) {
  const handleVote = (update: VoteState) => {
    onVote?.({ ...meme, ...update })
  }

  return (
    <article
      className={cn(
        'group relative overflow-hidden rounded-3xl border border-white/[0.08] bg-ink-850/80 backdrop-blur-sm transition duration-300 hover:border-white/20 hover:bg-ink-850 hover:shadow-[0_20px_50px_-24px_rgb(0_0_0/0.9)]',
        layout === 'grid' ? 'flex h-full flex-col' : 'mb-4 break-inside-avoid',
      )}
    >
      <Link to={`/m/${meme.id}`} className="relative block shrink-0 overflow-hidden bg-ink-950">
        {layout === 'grid' ? (
          <div className="relative aspect-[4/3] w-full">
            <img
              src={meme.imageUrl}
              alt={memeTitle(meme)}
              loading="lazy"
              decoding="async"
              className="absolute inset-0 h-full w-full object-contain transition duration-500 ease-out group-hover:scale-[1.015]"
            />
          </div>
        ) : (
          <img
            src={meme.imageUrl}
            alt={memeTitle(meme)}
            loading="lazy"
            decoding="async"
            style={
              meme.width && meme.height
                ? { aspectRatio: `${meme.width} / ${meme.height}` }
                : undefined
            }
            className="w-full bg-ink-900 object-cover transition duration-500 ease-out group-hover:scale-[1.015]"
          />
        )}
      </Link>

      <div className={cn('p-4', layout === 'grid' && 'flex flex-1 flex-col')}>
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              {rank !== undefined && (
                <span className="grid size-5 shrink-0 place-items-center rounded-full bg-acid-400 text-[10px] font-bold text-ink-950">
                  {rank}
                </span>
              )}
              <Link
                to={`/m/${meme.id}`}
                className="line-clamp-1 text-sm font-semibold text-zinc-100 transition hover:text-acid-400"
              >
                {memeTitle(meme)}
              </Link>
            </div>
            <div className="mt-1.5 flex items-center gap-1.5 text-[11px] text-zinc-500">
              <AuthorBadge name={meme.author} />
              <span className="max-w-24 truncate">{meme.author}</span>
              <span className="text-zinc-700">·</span>
              <span>{formatTime(meme.createdAt)}</span>
            </div>
          </div>
          <Link
            to={remixHref(meme)}
            className="btn-icon size-8 shrink-0"
            title="Remix this meme"
            aria-label="Remix this meme"
          >
            <Remix className="size-3.5" />
          </Link>
        </div>

        <div
          className={cn(
            'mt-3 flex items-center justify-between gap-2 border-t border-white/[0.06] pt-3',
            layout === 'grid' && 'mt-auto',
          )}
        >
          <VoteButtons meme={meme} onChange={handleVote} />
          <div className="flex min-w-0 items-center justify-end gap-1.5">
            {meme.animated && (
              <span className="shrink-0 rounded border border-hot-400/30 bg-hot-500/10 px-1 py-0.5 text-[8px] font-bold tracking-wider text-hot-300">
                GIF
              </span>
            )}
            {meme.templateName && (
              <span className="truncate text-[10px] font-medium tracking-wide text-zinc-600 uppercase">
                {meme.templateName}
              </span>
            )}
          </div>
        </div>
      </div>
    </article>
  )
}
