import { Collapsible } from 'radix-ui'
import { ChevronDown } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { cn } from '@/shared/lib/utils'
export function Disclosure({ title, children, defaultOpen = false, className, triggerClassName, id }: {
  title: ReactNode; children: ReactNode; defaultOpen?: boolean; className?: string; triggerClassName?: string; id?: string
}) {
  const [open, setOpen] = useState(defaultOpen)
  return <Collapsible.Root id={id} open={open} onOpenChange={setOpen} className={className}>
    <Collapsible.Trigger type="button" className={cn('group flex min-h-11 w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-left text-sm font-medium hover:bg-accent',triggerClassName)}>
      <span className="min-w-0">{title}</span><ChevronDown aria-hidden="true" className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-data-[state=open]:rotate-180" />
    </Collapsible.Trigger>
    <Collapsible.Content forceMount className="grace-disclosure" inert={!open} aria-hidden={!open}>
      <div className="min-h-0 overflow-hidden"><div className="pt-3">{children}</div></div>
    </Collapsible.Content>
  </Collapsible.Root>
}
