import { AlertCircle } from 'lucide-react'
import type { ReactNode } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'

export function Confidence({ value, route }: { value: number; route: string }) {
  return (
    <p className="text-xs text-muted-foreground">
      Маршрут: {route} · уверенность {Math.round(value * 100)}%
    </p>
  )
}

export function Notice({
  title,
  text,
  children,
}: {
  title: string
  text: string
  children?: ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2 text-amber-700">
          <AlertCircle className="size-5" aria-hidden="true" />
          <CardTitle>{title}</CardTitle>
        </div>
        <CardDescription>{text}</CardDescription>
      </CardHeader>
      {children && <CardContent>{children}</CardContent>}
    </Card>
  )
}

export function MalformedResult() {
  return (
    <Notice
      title="Не удалось обработать ответ"
      text="Повторите запрос позже. Финансовые данные не изменялись."
    />
  )
}
