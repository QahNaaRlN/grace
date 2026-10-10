import type { ComponentProps } from 'react'
import { cn } from '@/shared/lib/utils'
export function Skeleton({ className, ...props }: ComponentProps<'div'>) {
  return <div aria-hidden="true" data-slot="skeleton" className={cn('animate-pulse rounded-lg bg-muted', className)} {...props} />
}
export function LoadingSkeleton({ label = 'Загружаем данные…' }: { label?: string }) {
  return <div role="status" aria-busy="true" className="space-y-5 rounded-2xl border bg-card p-5">
    <span className="sr-only">{label}</span><Skeleton className="h-5 w-1/3" /><Skeleton className="h-10 w-1/2" />
    {[0, 1, 2].map(i => <div key={i} className="flex gap-4"><Skeleton className="size-11" /><div className="flex-1 space-y-2"><Skeleton className="h-4 w-3/4" /><Skeleton className="h-4 w-1/2" /></div></div>)}
  </div>
}
