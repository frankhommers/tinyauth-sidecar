import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { PasswordStrengthBar } from '@/components/PasswordStrengthBar'
import { RefreshCw } from 'lucide-react'
import { useFeatures } from '@/context/FeaturesContext'

export default function ResetPasswordPage() {
  const { t } = useTranslation()
  const features = useFeatures()
  const defaultTab = features.emailEnabled ? 'email' : 'sms'
  const [tab, setTab] = useState<'email' | 'sms'>(defaultTab)

  const [username, setUsername] = useState('')
  const [token, setToken] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [msg, setMsg] = useState('')
  const [resetRequested, setResetRequested] = useState(false)

  const [phone, setPhone] = useState('')
  const [smsCode, setSmsCode] = useState('')
  const [smsNewPassword, setSmsNewPassword] = useState('')
  const [smsConfirmPassword, setSmsConfirmPassword] = useState('')
  const [smsMsg, setSmsMsg] = useState('')
  const [codeSent, setCodeSent] = useState(false)
  const [resetting, setResetting] = useState(false)

  // If neither email nor SMS is configured, show a message
  if (features.loaded && !features.emailEnabled && !features.smsEnabled) {
    return (
      <Card className="w-full max-w-sm sm:max-w-md">
        <CardHeader>
          <CardTitle className="text-center text-3xl">{t('resetPage.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-center text-muted-foreground">{t('resetPage.notConfigured')}</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-sm sm:max-w-md">
      <CardHeader>
        <CardTitle className="text-center text-3xl">{t('resetPage.title')}</CardTitle>
        <CardDescription className="text-center">{t('resetPage.description')}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {features.emailEnabled && features.smsEnabled && (
          <div className="grid grid-cols-2 gap-2">
            <Button variant={tab === 'email' ? 'default' : 'outline'} onClick={() => setTab('email')}>
              {t('resetPage.tabEmail')}
            </Button>
            <Button variant={tab === 'sms' ? 'default' : 'outline'} onClick={() => setTab('sms')}>
              {t('resetPage.tabSms')}
            </Button>
          </div>
        )}

        {tab === 'email' && features.emailEnabled && (
          <>
            {msg && <div className="rounded-md border bg-muted px-3 py-2 text-sm">{msg}</div>}
            <div className="grid gap-2">
              <Label htmlFor="username">{features.usernameIsEmail ? t('resetPage.emailAddressLabel') : t('resetPage.usernameOrEmailLabel')}</Label>
              <Input id="username" value={username} onChange={(e) => setUsername(e.target.value)} />
            </div>
            {!resetRequested ? (
              <Button
                variant="outline"
                onClick={async () => {
                  await api.post('/password-reset/request', { username })
                  setMsg(t('resetPage.requestResetSuccess'))
                  setResetRequested(true)
                }}
              >
                {t('resetPage.requestReset')}
              </Button>
            ) : (
              <>
                <Separator />
                <div className="grid gap-2">
                  <Label htmlFor="token">{t('common.token')}</Label>
                  <Input id="token" value={token} onChange={(e) => setToken(e.target.value)} />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="newPassword">{t('common.newPassword')}</Label>
                  <Input id="newPassword" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
                  <PasswordStrengthBar password={newPassword} />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="confirmPassword">{t('common.confirmPassword')}</Label>
                  <Input id="confirmPassword" type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} />
                  {confirmPassword && newPassword !== confirmPassword && (
                    <p className="text-xs text-destructive">{t('accountPage.passwordMismatch')}</p>
                  )}
                </div>
                <Button
                  disabled={!newPassword || newPassword !== confirmPassword || resetting}
                  onClick={async () => {
                    setResetting(true)
                    try {
                      await api.post('/password-reset/confirm', { token, newPassword })
                      setMsg(t('resetPage.resetSuccess'))
                    } catch (e: any) {
                      setMsg(e?.response?.data?.error || t('resetPage.resetError'))
                    } finally {
                      setResetting(false)
                    }
                  }}
                >
                  {resetting && <RefreshCw className="h-3.5 w-3.5 animate-spin mr-1.5" />}
                  {t('resetPage.resetPassword')}
                </Button>
              </>
            )}
          </>
        )}

        {tab === 'sms' && features.smsEnabled && (
          <>
            {smsMsg && <div className="rounded-md border bg-muted px-3 py-2 text-sm">{smsMsg}</div>}
            {!codeSent ? (
              <>
                <div className="grid gap-2">
                  <Label htmlFor="phone">{t('common.phoneNumber')}</Label>
                  <Input id="phone" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+31612345678" />
                </div>
                <Button
                  variant="outline"
                  onClick={async () => {
                    try {
                      await api.post('/auth/forgot-password-sms', { phone })
                      setSmsMsg(t('resetPage.smsSent'))
                      setCodeSent(true)
                    } catch (e: any) {
                      setSmsMsg(e?.response?.data?.error || t('resetPage.smsSendError'))
                    }
                  }}
                >
                  {t('resetPage.sendResetCode')}
                </Button>
              </>
            ) : (
              <>
                <div className="grid gap-2">
                  <Label htmlFor="smsCode">{t('resetPage.smsCode')}</Label>
                  <Input id="smsCode" value={smsCode} onChange={(e) => setSmsCode(e.target.value)} />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="smsNewPassword">{t('common.newPassword')}</Label>
                  <Input id="smsNewPassword" type="password" value={smsNewPassword} onChange={(e) => setSmsNewPassword(e.target.value)} />
                  <PasswordStrengthBar password={smsNewPassword} />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="smsConfirmPassword">{t('common.confirmPassword')}</Label>
                  <Input id="smsConfirmPassword" type="password" value={smsConfirmPassword} onChange={(e) => setSmsConfirmPassword(e.target.value)} />
                  {smsConfirmPassword && smsNewPassword !== smsConfirmPassword && (
                    <p className="text-xs text-destructive">{t('accountPage.passwordMismatch')}</p>
                  )}
                </div>
                <Button
                  disabled={!smsNewPassword || smsNewPassword !== smsConfirmPassword || resetting}
                  onClick={async () => {
                    setResetting(true)
                    try {
                      await api.post('/auth/reset-password-sms', { phone, code: smsCode, newPassword: smsNewPassword })
                      setSmsMsg(t('resetPage.resetSuccess'))
                    } catch (e: any) {
                      setSmsMsg(e?.response?.data?.error || t('resetPage.resetError'))
                    } finally {
                      setResetting(false)
                    }
                  }}
                >
                  {resetting && <RefreshCw className="h-3.5 w-3.5 animate-spin mr-1.5" />}
                  {t('resetPage.resetPassword')}
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => {
                    setCodeSent(false)
                    setSmsMsg('')
                  }}
                >
                  {t('resetPage.resendCode')}
                </Button>
              </>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
