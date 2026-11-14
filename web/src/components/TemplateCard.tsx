import { useNavigate } from 'react-router-dom'
import type { Template } from '../lib/types'
import { cn } from '../lib/utils'
import { ArrowRight } from './Icons'

const sourceLabels: Record<string, string> = {
  imgflip: 'classic',
  upload: 'community',
  generated: 'canvas',
  giphy: 'giphy',
}

export default function TemplateCard({
  template,
  className,
}: {
  template: Template
  className?: string
}) {
  const navigate = useNavigate()

  return (
    <button
      type="button"
      onClick={() => navigate(`/create?template=${encodeURIComponent(template.id)}`)}
      className={cn(
        'group relative cursor-pointer overflow-hidden rounded-2xl border border-white/[0.08] bg-ink-850/70 text-left transition duration-300 hover:-translate-y-0.5 hover:border-acid-400/40 hover:shadow-[0_20px_40px_-24px_rgb(200_255_77/0.35)]',
        className,
      )}
    >
      <div className="relative overflow-hidden">
        <img
          src={template.imageUrl}
          alt={template.name}
          loading="lazy"
          decoding="async"
          className="aspect-[4/3] w-full object-cover transition duration-500 group-hover:scale-105"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-ink-950 via-ink-950/10 to-transparent opacity-80" />
        {template.animated && (
          <span className="absolute top-2 right-2 rounded-full border border-hot-400/40 bg-ink-950/80 px-2 py-0.5 text-[9px] font-bold tracking-wider text-hot-300 backdrop-blur">
            GIF
          </span>
        )}
        <span className="absolute inset-x-0 bottom-0 flex translate-y-1 items-center justify-between gap-2 p-3 transition duration-300 group-hover:translate-y-0">
          <span className="min-w-0 truncate text-xs font-semibold text-white">{template.name}</span>
          <span className="grid size-7 shrink-0 place-items-center rounded-full bg-acid-400 text-ink-950 opacity-0 transition group-hover:opacity-100">
            <ArrowRight className="size-3.5" />
          </span>
        </span>
      </div>
      <div className="flex items-center justify-between gap-2 px-3 py-2">
        <span className="text-[10px] font-medium tracking-wide text-zinc-600 uppercase">
          {sourceLabels[template.source] ?? template.source}
        </span>
        <span className="text-[10px] text-zinc-700">
          {template.width}×{template.height}
        </span>
      </div>
    </button>
  )
}
