import type { ReactNode } from 'react'
import { CircleAlert, CircleCheck, CircleDashed, Info, TriangleAlert } from 'lucide-react'

import { cn } from '@/shared/lib/utils'

export type Tone = 'success' | 'warning' | 'info' | 'danger' | 'unknown'

// SPEC-0035: a state is always a word plus an icon; the tone only reinforces it.
const tones: Record<Tone, { icon: typeof Info; box: string; mark: string; badge: string }> = {
  success: { icon: CircleCheck, box: 'border-success/25 bg-success-subtle', mark: 'text-success', badge: 'bg-success-subtle text-success' },
  warning: { icon: TriangleAlert, box: 'border-warning/25 bg-warning-subtle', mark: 'text-warning', badge: 'bg-warning-subtle text-warning' },
  info: { icon: Info, box: 'border-info/25 bg-info-subtle', mark: 'text-info', badge: 'bg-info-subtle text-info' },
  danger: { icon: CircleAlert, box: 'border-destructive/25 bg-destructive-subtle', mark: 'text-destructive', badge: 'bg-destructive-subtle text-destructive' },
  unknown: { icon: CircleDashed, box: 'border-dashed border-input bg-unknown-subtle', mark: 'text-unknown', badge: 'border border-dashed border-input bg-unknown-subtle text-unknown' },
}

// Inline message for coverage, consent, model errors and other states the
// person must not miss. `danger` is announced as an alert, the rest as status.
export function Notice({
  tone = 'info',
  title,
  children,
  action,
  className,
}: {
  tone?: Tone
  title: ReactNode
  children?: ReactNode
  /** An outline or link Button — never the primary action. */
  action?: ReactNode
  className?: string
}) {
  const { icon: Icon, box, mark } = tones[tone]
  return (
    <div
      data-slot="notice"
      data-tone={tone}
      role={tone === 'danger' ? 'alert' : 'status'}
      className={cn('flex gap-3 rounded-xl border p-4 text-sm', box, className)}
    >
      <Icon aria-hidden="true" className={cn('mt-0.5 size-5 shrink-0', mark)} />
      <div className="grid min-w-0 gap-1">
        <p className="font-semibold text-foreground">{title}</p>
        {children && <div className="text-foreground">{children}</div>}
        {action && <div className="mt-2">{action}</div>}
      </div>
    </div>
  )
}

// A 24px state pill with a mandatory icon. Use plain Badge for kinds.
export function StatusBadge({ tone, children, className }: { tone: Tone; children: ReactNode; className?: string }) {
  const { icon: Icon, badge } = tones[tone]
  return (
    <span
      data-slot="status-badge"
      data-tone={tone}
      className={cn('inline-flex h-6 shrink-0 items-center gap-1 rounded-full px-2 text-xs font-medium whitespace-nowrap', badge, className)}
    >
      <Icon aria-hidden="true" className="size-3.5" />
      {children}
    </span>
  )
}
