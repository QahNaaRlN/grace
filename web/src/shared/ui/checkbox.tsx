import type { ComponentProps } from 'react'
import { Checkbox as Primitive } from 'radix-ui'
import { Check } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

export function Checkbox({
  className,
  onCheckedChange,
  ...props
}: Omit<ComponentProps<typeof Primitive.Root>, 'onCheckedChange'> & {
  onCheckedChange?: (checked: boolean) => void
}) {
  return (
    <Primitive.Root
      data-slot="checkbox"
      className={cn(
        'grid size-5 shrink-0 cursor-pointer place-items-center rounded-md border border-input bg-card text-primary-foreground transition-colors outline-none data-[state=checked]:border-primary data-[state=checked]:bg-primary disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      onCheckedChange={(checked) => onCheckedChange?.(checked === true)}
      {...props}
    >
      <Primitive.Indicator>
        <Check className="size-3.5" strokeWidth={2.5} aria-hidden="true" />
      </Primitive.Indicator>
    </Primitive.Root>
  )
}
