import { createContext } from 'react'

import type { components } from '@/shared/api/generated/schema'

export type AuthSession = components['schemas']['AuthSession']
export const AuthContext = createContext<{
  session: AuthSession
  logout: () => Promise<void>
  revokeSessions: () => Promise<void>
} | null>(null)
