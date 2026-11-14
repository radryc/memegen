import type { ReactNode } from 'react'

type IconProps = { className?: string }

function Svg({
  className = 'size-4',
  children,
}: IconProps & { children: ReactNode }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className={className}
    >
      {children}
    </svg>
  )
}

export function ArrowUp({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M12 19V5" />
      <path d="m5 12 7-7 7 7" />
    </Svg>
  )
}

export function ArrowDown({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M12 5v14" />
      <path d="m19 12-7 7-7-7" />
    </Svg>
  )
}

export function Flame({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M12 22c4.4 0 7-2.8 7-6.5 0-2.4-1.3-4.5-3-6.3-.5 1.7-1.6 2.8-3 3.3.4-3.4-1.4-6.2-4-8.5.2 3.4-1.3 5-2.8 7A8.2 8.2 0 0 0 5 15.5C5 19.2 7.6 22 12 22Z" />
    </Svg>
  )
}

export function Sparkles({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="m12 3 1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8L12 3Z" />
      <path d="M19 15.5 19.8 18l2.2.8-2.2.8-.8 2.4-.8-2.4-2.2-.8 2.2-.8.8-2.5Z" />
      <path d="M5 3.5 5.6 5.4 7.5 6l-1.9.6L5 8.5l-.6-1.9L2.5 6l1.9-.6L5 3.5Z" />
    </Svg>
  )
}

export function Upload({ className }: IconProps) {  return (
    <Svg className={className}>
      <path d="M21 15v3a3 3 0 0 1-3 3H6a3 3 0 0 1-3-3v-3" />
      <path d="M12 15V3" />
      <path d="m7 8 5-5 5 5" />
    </Svg>
  )
}

export function Download({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M21 15v3a3 3 0 0 1-3 3H6a3 3 0 0 1-3-3v-3" />
      <path d="M12 3v12" />
      <path d="m7 10 5 5 5-5" />
    </Svg>
  )
}

export function Dice({ className }: IconProps) {
  return (
    <Svg className={className}>
      <rect x="3" y="3" width="18" height="18" rx="4" />
      <circle cx="8.5" cy="8.5" r=".8" fill="currentColor" />
      <circle cx="15.5" cy="15.5" r=".8" fill="currentColor" />
      <circle cx="12" cy="12" r=".8" fill="currentColor" />
    </Svg>
  )
}

export function LinkIcon({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M10 13a5 5 0 0 0 7.5.5l2-2a5 5 0 0 0-7-7l-1.2 1.2" />
      <path d="M14 11a5 5 0 0 0-7.5-.5l-2 2a5 5 0 0 0 7 7l1.2-1.2" />
    </Svg>
  )
}

export function Check({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="m5 12 4.5 4.5L19 6.5" />
    </Svg>
  )
}

export function ArrowRight({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M4 12h16" />
      <path d="m14 6 6 6-6 6" />
    </Svg>
  )
}

export function ArrowLeft({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M20 12H4" />
      <path d="m10 18-6-6 6-6" />
    </Svg>
  )
}

export function Search({ className }: IconProps) {
  return (
    <Svg className={className}>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" />
    </Svg>
  )
}

export function Plus({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M12 5v14" />
      <path d="M5 12h14" />
    </Svg>
  )
}

export function Trophy({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M8 4h8v5a4 4 0 0 1-8 0V4Z" />
      <path d="M8 5H5v2a3 3 0 0 0 3 3" />
      <path d="M16 5h3v2a3 3 0 0 1-3 3" />
      <path d="M12 13v4" />
      <path d="M9 21h6" />
      <path d="M10 17h4l.5 4h-5l.5-4Z" />
    </Svg>
  )
}

export function X({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M6 6l12 12" />
      <path d="M18 6 6 18" />
    </Svg>
  )
}

export function Loader({ className }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className={`animate-spin ${className ?? 'size-4'}`}>
      <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="2.5" opacity="0.2" />
      <path
        d="M21 12a9 9 0 0 0-9-9"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
      />
    </svg>
  )
}

export function Spark({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M12 2v4" />
      <path d="M12 18v4" />
      <path d="M2 12h4" />
      <path d="M18 12h4" />
      <path d="M4.9 4.9 7.8 7.8" />
      <path d="m16.2 16.2 2.9 2.9" />
      <path d="m19.1 4.9-2.9 2.9" />
      <path d="m7.8 16.2-2.9 2.9" />
    </Svg>
  )
}

export function Remix({ className }: IconProps) {
  return (
    <Svg className={className}>
      <path d="M17 2.5 21 6l-4 3.5" />
      <path d="M3 11V9a3 3 0 0 1 3-3h15" />
      <path d="M7 21.5 3 18l4-3.5" />
      <path d="M21 13v2a3 3 0 0 1-3 3H3" />
    </Svg>
  )
}

export function ImageIcon({ className }: IconProps) {
  return (
    <Svg className={className}>
      <rect x="3" y="3" width="18" height="18" rx="3" />
      <circle cx="8.5" cy="8.5" r="1.5" />
      <path d="m21 15-5-5L5 21" />
    </Svg>
  )
}
