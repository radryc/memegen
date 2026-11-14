import { useEffect, useState } from 'react'
import type { Meme } from '../lib/types'
import { castVote } from '../lib/api'
import { cn, scoreTone } from '../lib/utils'
import { ArrowDown, ArrowUp, Loader } from './Icons'

export interface VoteState {
  upvotes: number
  downvotes: number
  score: number
  myVote: number
}

/** Shared optimistic voting state for cards and the detail page. */
export function useVote(meme: Meme, onChange?: (update: VoteState) => void) {
  const [state, setState] = useState<VoteState>({
    upvotes: meme.upvotes,
    downvotes: meme.downvotes,
    score: meme.score,
    myVote: meme.myVote,
  })
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setState({
      upvotes: meme.upvotes,
      downvotes: meme.downvotes,
      score: meme.score,
      myVote: meme.myVote,
    })
  }, [meme.id, meme.upvotes, meme.downvotes, meme.score, meme.myVote])

  const vote = async (value: -1 | 1) => {
    if (pending) return
    const previous = state
    const myVote = previous.myVote === value ? 0 : value
    const upvotes =
      previous.upvotes - (previous.myVote === 1 ? 1 : 0) + (myVote === 1 ? 1 : 0)
    const downvotes =
      previous.downvotes - (previous.myVote === -1 ? 1 : 0) + (myVote === -1 ? 1 : 0)

    setState({ upvotes, downvotes, score: upvotes - downvotes, myVote })
    setPending(true)
    setError('')
    try {
      const result = await castVote(meme.id, value)
      setState(result)
      onChange?.(result)
    } catch (err) {
      setState(previous)
      setError(err instanceof Error ? err.message : 'Vote failed')
    } finally {
      setPending(false)
    }
  }

  return { state, pending, error, vote }
}

export function VoteButtons({
  meme,
  onChange,
  size = 'sm',
}: {
  meme: Meme
  onChange?: (update: VoteState) => void
  size?: 'sm' | 'lg'
}) {
  const { state, pending, vote } = useVote(meme, onChange)
  const large = size === 'lg'

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1 rounded-full border border-white/10 bg-white/[0.04] p-1',
        large && 'gap-2 p-1.5',
      )}
    >
      <button
        type="button"
        onClick={() => void vote(1)}
        disabled={pending}
        aria-label="Upvote"
        aria-pressed={state.myVote === 1}
        title="Upvote"
        className={cn(
          'grid cursor-pointer place-items-center rounded-full text-zinc-400 transition hover:bg-acid-400/15 hover:text-acid-400 disabled:cursor-wait',
          large ? 'size-10' : 'size-7',
          state.myVote === 1 && 'bg-acid-400 text-ink-950 hover:bg-acid-300 hover:text-ink-950',
        )}
      >
        <ArrowUp className={large ? 'size-5' : 'size-4'} />
      </button>

      <span
        className={cn(
          'min-w-6 text-center text-sm font-bold tabular-nums',
          large && 'min-w-10 text-lg',
          scoreTone(state.score),
        )}
        aria-live="polite"
      >
        {pending ? <Loader className={large ? 'size-4' : 'size-3'} /> : state.score}
      </span>

      <button
        type="button"
        onClick={() => void vote(-1)}
        disabled={pending}
        aria-label="Downvote"
        aria-pressed={state.myVote === -1}
        title="Downvote"
        className={cn(
          'grid cursor-pointer place-items-center rounded-full text-zinc-400 transition hover:bg-hot-400/15 hover:text-hot-400 disabled:cursor-wait',
          large ? 'size-10' : 'size-7',
          state.myVote === -1 && 'bg-hot-500 text-white hover:bg-hot-400 hover:text-white',
        )}
      >
        <ArrowDown className={large ? 'size-5' : 'size-4'} />
      </button>
    </div>
  )
}
