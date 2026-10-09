import { Children, Fragment, isValidElement, type ReactNode, type ComponentProps, type ReactElement } from 'react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './select'
import { cn } from '@/shared/lib/utils'

const emptyValue = '__grace_empty_selection__'
type Option = ReactElement<ComponentProps<'option'>>
function options(children: ReactNode): Option[] {
  return Children.toArray(children).flatMap(child => {
    if (!isValidElement(child)) return []
    if (child.type === Fragment) return options((child.props as { children: ReactNode }).children)
    return child.type === 'option' ? [child as Option] : []
  })
}
export function ChoiceSelect({ id, value, children, onValueChange, disabled, className, ...props }: Pick<ComponentProps<typeof SelectTrigger>, 'id' | 'className' | 'aria-label' | 'aria-describedby' | 'aria-invalid'> & { value: string; children: ReactNode; disabled?: boolean; onValueChange: (value: string) => void }) {
  const items = options(children)
  return <Select value={value || emptyValue} disabled={disabled || items.length === 0} onValueChange={value => onValueChange(value === emptyValue ? '' : value)}><SelectTrigger id={id} className={cn('w-full', className)} {...props}><SelectValue placeholder="Выберите значение" /></SelectTrigger><SelectContent position="popper" align="start">{items.map(item => {
    const value = String(item.props.value ?? item.props.children ?? '')
    return <SelectItem key={value || emptyValue} value={value || emptyValue} disabled={item.props.disabled}>{item.props.children}</SelectItem>
  })}</SelectContent></Select>
}
