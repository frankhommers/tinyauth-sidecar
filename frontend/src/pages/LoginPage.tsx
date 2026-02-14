import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useFeatures } from '@/context/FeaturesContext'
import { useAuth } from '@/context/AuthContext'

export default function LoginPage() {
  const { t } = useTranslation()
  const { signupEnabled } = useFeatures()
  const navigate = useNavigate()
  const { refresh } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [msg, setMsg] = useState('')
  const [loading, setLoading] = useState(false)

  // Auto-login via tinyauth SSO
  useEffect(() => {
    api.get('/auth/sso').then((res) => {
      if (res.data.authenticated) {
        navigate('/account')
      }
    }).catch(() => {})
  }, [navigate])

  const submit = async () => {
    setLoading(true)
    try {
      await api.post('/auth/login', { username, password })
      refresh()
      navigate('/account')
    } catch (e: any) {
      setMsg(e?.response?.data?.error || t('loginPage.error'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card className="min-w-xs sm:min-w-sm">
      <CardHeader>
        <CardTitle className="text-center text-3xl">{t('loginPage.title')}</CardTitle>
        <CardDescription className="text-center">{t('loginPage.description')}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {msg && <div className="rounded-md border bg-muted px-3 py-2 text-sm">{msg}</div>}
        <div className="grid gap-2">
          <Label htmlFor="username">{t('common.username')}</Label>
          <Input id="username" value={username} onChange={(e) => setUsername(e.target.value)} />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="password">{t('common.password')}</Label>
          <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </div>
        <Button onClick={submit} loading={loading} disabled={!username || !password}>
          {t('loginPage.submit')}
        </Button>
      </CardContent>
      <CardFooter className={signupEnabled ? "flex justify-between text-sm" : "flex justify-end text-sm"}>
        {signupEnabled && (
          <Link to="/signup" className="text-muted-foreground hover:text-foreground">
            {t('loginPage.createAccount')}
          </Link>
        )}
        <Link to="/reset-password" className="text-muted-foreground hover:text-foreground">
          {t('loginPage.forgotPassword')}
        </Link>
      </CardFooter>
    </Card>
  )
}
