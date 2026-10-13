import type { ReactNode } from 'react'

export function PageHeader({ title, description, status, actions }: { title: string; description?: string; status?: string; actions?: ReactNode }) {
  return <header className="flex flex-wrap items-start justify-between gap-5"><div className="min-w-0"><h1 className="text-title-1 font-semibold tracking-tight">{title}</h1>{description && <p className="mt-2 max-w-2xl text-sm leading-relaxed text-muted-foreground">{description}</p>}{status && <p role="status" className="mt-2 text-sm text-muted-foreground">{status}</p>}</div>{actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}</header>
}
