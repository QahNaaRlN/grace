import type * as React from 'react'

import { cn } from '@/shared/lib/utils'

export function Textarea({ className, ...props }: React.ComponentProps<'textarea'>) {
  return (
    <textarea
      className={cn(
        'border-input placeholder:text-muted-foreground aria-invalid:ring-destructive/20 aria-invalid:border-destructive min-h-32 w-full resize-y rounded-xl border bg-card px-3 py-3 text-base shadow-none outline-none transition-[color,box-shadow] disabled:cursor-not-allowed disabled:opacity-50 ',
        className,
      )}
      {...props}
    />
  )
}
