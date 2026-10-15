import { cloneElement, isValidElement, useId, type ReactElement, type ReactNode } from 'react'
import { CircleAlert } from 'lucide-react'

import { cn } from '@/shared/lib/utils'
import { Label } from './label'

type ControlProps = {
  id?: string
  'aria-describedby'?: string
  'aria-invalid'?: boolean | 'true' | 'false'
}

// SPEC-0035: one control with its permanent label, optional hint and error,
// wired for assistive technology. The error is a sentence with an icon.
export function Field({
  label,
  hint,
  error,
  children,
  className,
}: {
  label: ReactNode
  hint?: ReactNode
  error?: ReactNode
  /** Exactly one control: Input, MoneyInput, Textarea, ChoiceSelect, PasswordInput. */
  children: ReactElement<ControlProps>
  className?: string
}) {
  const base = useId()
  const control = isValidElement(children) ? children : null
  const controlId = control?.props.id ?? `${base}-control`
  const hintId = `${base}-hint`
  const errorId = `${base}-error`
  const describedBy =
    [control?.props['aria-describedby'], hint ? hintId : null, error ? errorId : null]
      .filter(Boolean)
      .join(' ') || undefined
  return (
    <div data-slot="field" className={cn('grid gap-2', className)}>
      <Label htmlFor={controlId}>{label}</Label>
      {control
        ? cloneElement(control, {
            id: controlId,
            'aria-describedby': describedBy,
            'aria-invalid': error ? true : control.props['aria-invalid'],
          })
        : children}
      {hint && (
        <p id={hintId} className="text-sm text-muted-foreground">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} className="flex items-center gap-1.5 text-sm text-destructive">
          <CircleAlert aria-hidden="true" className="size-4 shrink-0" />
          {error}
        </p>
      )}
    </div>
  )
}
