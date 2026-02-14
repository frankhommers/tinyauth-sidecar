import { Navigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'

export function SmartRedirect() {
  const { loggedIn, loading } = useAuth()
  if (loading) return null
  return <Navigate to={loggedIn ? '/account' : '/reset-password'} replace />
}
