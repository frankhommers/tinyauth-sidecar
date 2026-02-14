import { Navigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { loggedIn, loading } = useAuth()

  if (loading) return null

  if (!loggedIn) {
    // Not authenticated — show reset password page instead
    return <Navigate to="/reset-password" replace />
  }

  return <>{children}</>
}
