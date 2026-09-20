import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import { api } from '../lib/api'
import type { AuthUser, LoginResponse } from '../types'

interface AuthContextValue {
  isAuthenticated: boolean
  role: string | null
  user: AuthUser | null
  signIn: (tenantId: string, email: string, password: string) => Promise<void>
  signOut: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

function userFromLogin(data: LoginResponse, tenantId: string): AuthUser | null {
  const u = data.user
  if (!u || typeof u !== 'object') {
    return null
  }
  const role = typeof u.role === 'string' ? u.role : ''
  if (!role) {
    return null
  }
  return {
    id: typeof u.id === 'string' ? u.id : '',
    tenant_id: tenantId,
    email: typeof u.email === 'string' ? u.email : undefined,
    display_name: typeof u.display_name === 'string' ? u.display_name : '',
    role,
    account_type: typeof u.account_type === 'string' ? u.account_type : undefined,
    permissions:
      u.permissions && typeof u.permissions === 'object'
        ? (u.permissions as Record<string, unknown>)
        : {},
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(api.hasToken())
  const [user, setUser] = useState<AuthUser | null>(() => api.readUser())

  const signIn = useCallback(
    async (tenantId: string, email: string, password: string) => {
      const data = await api.login(tenantId, email, password)
      const parsed = userFromLogin(data, tenantId)
      if (parsed) {
        api.setUser(parsed)
        setUser(parsed)
      }
      setIsAuthenticated(true)
    },
    [],
  )

  const signOut = useCallback(() => {
    api.setToken(null)
    api.clearUser()
    setUser(null)
    setIsAuthenticated(false)
  }, [])

  const value = useMemo(
    () => ({
      isAuthenticated,
      role: user?.role ?? null,
      user,
      signIn,
      signOut,
    }),
    [isAuthenticated, user, signIn, signOut],
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