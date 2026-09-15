import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import { api } from '../lib/api'

interface AuthContextValue {
  isAuthenticated: boolean
  signIn: (tenantId: string, email: string, password: string) => Promise<void>
  signOut: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(api.hasToken())

  const signIn = useCallback(
    async (tenantId: string, email: string, password: string) => {
      await api.login(tenantId, email, password)
      setIsAuthenticated(true)
    },
    [],
  )

  const signOut = useCallback(() => {
    api.setToken(null)
    setIsAuthenticated(false)
  }, [])

  const value = useMemo(
    () => ({ isAuthenticated, signIn, signOut }),
    [isAuthenticated, signIn, signOut],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}