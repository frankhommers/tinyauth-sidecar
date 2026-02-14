import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react'
import { api } from '../api/client'

interface AuthState {
  loggedIn: boolean
  loading: boolean
  refresh: () => void
}

const AuthContext = createContext<AuthState>({ loggedIn: false, loading: true, refresh: () => {} })

export function AuthProvider({ children }: { children: ReactNode }) {
  const [loggedIn, setLoggedIn] = useState(false)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(() => {
    setLoading(true)
    api.get('/account/profile')
      .then(() => setLoggedIn(true))
      .catch(() => setLoggedIn(false))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { refresh() }, [refresh])

  return <AuthContext.Provider value={{ loggedIn, loading, refresh }}>{children}</AuthContext.Provider>
}

export function useAuth() {
  return useContext(AuthContext)
}
