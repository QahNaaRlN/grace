import { Tag } from 'lucide-react'

import { categoryIcons, type CategoryIconName } from '@/shared/lib/category-icons'
import { cn } from '@/shared/lib/utils'

export function CategoryIcon({ icon, className }: { icon?: CategoryIconName | null; className?: string }) {
  const Icon = (icon && categoryIcons[icon]?.icon) || Tag
  return <Icon aria-hidden="true" className={cn('size-5 shrink-0', className)} />
}
