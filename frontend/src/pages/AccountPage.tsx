import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Copy, Check } from 'lucide-react'

type Profile = {
  username: string
  totpEnabled: boolean
  phone?: string
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <Button
      variant="ghost"
      size="icon"
      className="h-6 w-6 shrink-0"
      onClick={async () => {
        await navigator.clipboard.writeText(value)
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
      }}
    >
      {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
    </Button>
  )
}

export default function AccountPage() {
  const { t } = useTranslation()
  const [profile, setProfile] = useState<Profile | null>(null)
  const [msg, setMsg] = useState('')
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [phone, setPhone] = useState('')
  const [totpSecret, setTotpSecret] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const [qrPng, setQrPng] = useState('')
  const [otpUrl, setOtpUrl] = useState('')
  const [disablePassword, setDisablePassword] = useState('')

  const load = async () => {
    try {
      const data = (await api.get('/account/profile')).data
      setProfile(data)
      setPhone(data.phone || '')
    } catch {
      setProfile(null)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  return (
    <Card className="min-w-xs sm:min-w-sm">
      <CardHeader>
        <CardTitle className="text-center text-3xl">{t('accountPage.title')}</CardTitle>
        <CardDescription className="text-center">{t('accountPage.description')}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {msg && <div className="rounded-md border bg-muted px-3 py-2 text-sm">{msg}</div>}

        {profile && (
          <div className="rounded-md border bg-background/45 p-3 text-sm">
            <p>
              <span className="font-medium">{t('common.username')}:</span> {profile.username}
            </p>
            <p>
              <span className="font-medium">{t('accountPage.totpEnabled')}:</span> {String(profile.totpEnabled)}
            </p>
            {profile.phone && (
              <p>
                <span className="font-medium">{t('common.phone')}:</span> {profile.phone}
              </p>
            )}
          </div>
        )}

        <Separator />
        <h3 className="text-base font-semibold">{t('accountPage.changePassword')}</h3>
        <div className="grid gap-2">
          <Label htmlFor="oldPassword">{t('accountPage.currentPassword')}</Label>
          <Input id="oldPassword" type="password" value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="newPassword">{t('common.newPassword')}</Label>
          <Input id="newPassword" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
        </div>
        <Button
          onClick={async () => {
            try {
              await api.post('/account/change-password', { oldPassword, newPassword })
              setMsg(t('accountPage.passwordChanged'))
              setOldPassword('')
              setNewPassword('')
            } catch (e: any) {
              setMsg(e?.response?.data?.error || t('accountPage.genericError'))
            }
          }}
        >
          {t('accountPage.changePassword')}
        </Button>

        <Separator />
        <h3 className="text-base font-semibold">{t('common.phoneNumber')}</h3>
        <div className="flex flex-wrap gap-2">
          <Input
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="+31612345678"
            className="flex-1 min-w-[180px]"
          />
          <Button
            variant="outline"
            onClick={async () => {
              try {
                await api.post('/account/phone', { phone })
                setMsg(t('accountPage.phoneUpdated'))
                void load()
              } catch (e: any) {
                setMsg(e?.response?.data?.error || t('accountPage.genericError'))
              }
            }}
          >
            {t('common.save')}
          </Button>
        </div>

        <Separator />
        <h3 className="text-base font-semibold">{t('accountPage.totpSetup')}</h3>
        <Button
          variant="outline"
          onClick={async () => {
            const data = (await api.post('/account/totp/setup')).data
            setTotpSecret(data.secret)
            setQrPng(data.qrPng)
            setOtpUrl(data.otpUrl)
          }}
        >
          {t('accountPage.generateSecret')}
        </Button>

        {qrPng && <img src={qrPng} width={220} className="self-center max-w-full rounded-md border" alt={t('accountPage.totpQrAlt')} />}

        {totpSecret && (
          <div className="flex items-center gap-2 rounded-md border bg-background/45 p-2 text-xs break-all">
            <span className="flex-1">{t('accountPage.secret')}: {totpSecret}</span>
            <CopyButton value={totpSecret} />
          </div>
        )}

        {otpUrl && (
          <div className="flex items-center gap-2 rounded-md border bg-background/45 p-2 text-xs">
            <span className="flex-1 truncate">{otpUrl}</span>
            <CopyButton value={otpUrl} />
          </div>
        )}

        <div className="flex flex-wrap gap-2">
          <Input
            value={totpCode}
            onChange={(e) => setTotpCode(e.target.value)}
            placeholder={t('common.code')}
            className="flex-1 min-w-[120px]"
          />
          <Button
            onClick={async () => {
              try {
                await api.post('/account/totp/enable', { secret: totpSecret, code: totpCode })
                setMsg(t('accountPage.totpEnabledSuccess'))
                void load()
              } catch (e: any) {
                setMsg(e?.response?.data?.error || t('accountPage.genericError'))
              }
            }}
          >
            {t('accountPage.enableTotp')}
          </Button>
        </div>

        <div className="flex flex-wrap gap-2">
          <Input
            type="password"
            value={disablePassword}
            onChange={(e) => setDisablePassword(e.target.value)}
            placeholder={t('common.password')}
            className="flex-1 min-w-[120px]"
          />
          <Button
            variant="outline"
            onClick={async () => {
              try {
                await api.post('/account/totp/disable', { password: disablePassword })
                setMsg(t('accountPage.totpDisabledSuccess'))
                void load()
              } catch (e: any) {
                setMsg(e?.response?.data?.error || t('accountPage.genericError'))
              }
            }}
          >
            {t('accountPage.disableTotp')}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
