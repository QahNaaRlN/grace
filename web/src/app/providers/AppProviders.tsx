import type { ReactNode } from 'react'
import { ThemeProvider } from 'next-themes'

import { QueryProvider } from '@/app/providers/QueryProvider'
import { Toaster } from '@/shared/ui/sonner'

type AppProvidersProps = {
  children: ReactNode
}

/**
 * Провайдер, который объединяет глобальные контексты приложения.
 */
export function AppProviders({ children }: AppProvidersProps) {
  return (
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem={false}>
      <QueryProvider>
        {children}
        <Toaster richColors closeButton position="top-right" />
      </QueryProvider>
    </ThemeProvider>
  )
}
