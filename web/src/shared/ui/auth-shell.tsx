import type { ReactNode } from 'react'
import { Card, CardContent, CardHeader } from './card'
import { GraceMark } from './GraceMark'

export function AuthShell({ title, children }: { title: string; children: ReactNode }) {
  return <main className="mx-auto flex min-h-dvh w-full max-w-md items-center px-4 py-8"><Card className="w-full"><CardHeader><div className="mb-5 flex items-center gap-2.5 text-primary"><GraceMark className="size-8" aria-hidden="true" /><span className="text-xl font-semibold tracking-tight">Grace</span></div><h1 className="text-2xl leading-tight font-semibold tracking-tight">{title}</h1></CardHeader><CardContent>{children}</CardContent></Card></main>
}
