import { useState, type ComponentProps } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { Input } from './input'
import { Button } from './button'
import { cn } from '@/shared/lib/utils'

export function PasswordInput({
  className,
  visibilityLabel = 'пароль',
  disabled,
  id,
  ...props
}: Omit<ComponentProps<typeof Input>, 'type'> & { visibilityLabel?: string }) {
  const [visible, setVisible] = useState(false)
  const Icon = visible ? Eye : EyeOff
  return (
    <div className="relative">
      <Input
        {...props}
        id={id}
        disabled={disabled}
        type={visible ? 'text' : 'password'}
        className={cn('pr-12', className)}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        disabled={disabled}
        className="absolute top-0 right-0 rounded-l-none text-muted-foreground hover:text-foreground"
        aria-label={`${visible ? 'Скрыть' : 'Показать'} ${visibilityLabel}`}
        aria-pressed={visible}
        aria-controls={id}
        onClick={() => setVisible(!visible)}
      >
        <Icon className="size-[18px]" aria-hidden="true" />
      </Button>
    </div>
  )
}
