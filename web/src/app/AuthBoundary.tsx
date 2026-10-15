import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'

import { apiClient } from '@/shared/api'
import { AuthContext } from '@/shared/model/auth-context'
import { setCSRFToken } from '@/shared/api'
import { Button } from '@/shared/ui/button'

async function loadSession() {
  const { data, response } = await apiClient.GET('/api/auth/me')
  if (response.status === 401) return null
  if (!data) throw new Error('Проверка доступа временно недоступна')
  setCSRFToken(data.csrfToken)
  return data
}

export function AuthBoundary({ children }: { children: ReactNode }) {
  const cache = useQueryClient()
  const navigate = useNavigate()
  const [knownUser, setKnownUser] = useState<string | undefined>(undefined)
  const session = useQuery({
    queryKey: ['auth-session'],
    queryFn: async () => {
      const result = await loadSession()
      if (result) setKnownUser((previous) => previous ?? result.user.id)
      return result
    },
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: true,
  })
  const clear = useCallback(() => {
    setCSRFToken('')
    void cache.cancelQueries()
    cache.clear()
    void navigate('/login', { replace: true })
  }, [cache, navigate])

  useEffect(() => {
    window.addEventListener('grace:unauthenticated', clear)
    return () => window.removeEventListener('grace:unauthenticated', clear)
  }, [clear])

  useEffect(() => {
    if (session.data === null) clear()
    if (session.data) {
      if (knownUser && knownUser !== session.data.user.id) clear()
    }
  }, [session.data, clear, knownUser])

  async function logout() {
    const { response } = await apiClient.POST('/api/auth/logout')
    if (!response.ok) throw new Error('Не удалось выйти. Повторите действие.')
    clear()
  }
  async function revokeSessions() {
    const { response } = await apiClient.POST('/api/auth/revoke-sessions')
    if (!response.ok) throw new Error('Не удалось завершить сеансы.')
    clear()
  }

  if (session.isPending)
    return (
      <main className="p-6" aria-live="polite">
        Проверяем доступ…
      </main>
    )
  if (session.isError)
    return (
      <main className="space-y-4 p-6">
        <p role="alert">Проверка доступа временно недоступна.</p>
        <Button onClick={() => void session.refetch()}>Повторить</Button>
      </main>
    )
  if (!session.data) return <Navigate to="/login" replace />
  if (knownUser && knownUser !== session.data.user.id) return <Navigate to="/login" replace />

  return (
    <AuthContext.Provider value={{ session: session.data, logout, revokeSessions }}>
      {children}
    </AuthContext.Provider>
  )
}
