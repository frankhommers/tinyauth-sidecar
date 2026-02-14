import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react'
import { api } from '../api/client'

interface AuthState {
  loggedIn: boolean
  loading: boolean
  username: string
  refresh: () => void
}

const AuthContext = createContext<AuthState>({ loggedIn: false, loading: true, username: '', refresh: () => {} })

export function AuthProvider({ children }: { children: ReactNode }) {
  const [loggedIn, setLoggedIn] = useState(false)
  const [loading, setLoading] = useState(true)
  const [username, setUsername] = useState('')

  const refresh = useCallback(() => {
    setLoading(true)
    api.get('/auth/check')
      .then((res) => {
        setLoggedIn(true)
        setUsername(res.data.username || '')
      })
      .catch(() => {
        setLoggedIn(false)
        setUsername('')
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { refresh() }, [refresh])

  return <AuthContext.Provider value={{ loggedIn, loading, username, refresh }}>{children}</AuthContext.Provider>
}

export function useAuth() {
  return useContext(AuthContext)
}
