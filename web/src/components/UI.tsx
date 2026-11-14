import type { ReactNode } from 'react'
import { Sparkles } from './Icons'

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string
  description: string
  action?: ReactNode
}) {
  return (
    <div className="panel flex flex-col items-center justify-center px-6 py-16 text-center">
      <div className="grid size-14 place-items-center rounded-2xl border border-acid-400/20 bg-acid-400/10 text-acid-400">
        <Sparkles className="size-7" />
      </div>
      <h3 className="mt-5 text-lg font-bold text-white">{title}</h3>
      <p className="mt-2 max-w-sm text-sm text-zinc-500">{description}</p>
      {action && <div className="mt-6">{action}</div>}
    </div>
  )
}

export function CardSkeleton({ tall = false }: { tall?: boolean }) {
  return (
    <div className="mb-4 break-inside-avoid overflow-hidden rounded-3xl border border-white/[0.06] bg-ink-850/50">
      <div className={`skeleton rounded-none ${tall ? 'aspect-[3/4]' : 'aspect-square'}`} />
      <div className="space-y-3 p-4">
        <div className="skeleton h-4 w-2/3 rounded-full" />
        <div className="skeleton h-3 w-1/3 rounded-full" />
        <div className="skeleton h-8 w-28 rounded-full" />
      </div>
    </div>
  )
}

export function PageSkeleton() {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <CardSkeleton key={i} tall={i % 3 === 1} />
      ))}
    </div>
  )
}
