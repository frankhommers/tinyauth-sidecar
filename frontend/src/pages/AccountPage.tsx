import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { AnimatedHeight } from '@/components/AnimatedHeight'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { PasswordStrengthBar } from '@/components/PasswordStrengthBar'
import { Copy, Check, ShieldCheck, ShieldAlert, User, Lock, Shield, Settings, CheckCircle, XCircle, RefreshCw } from 'lucide-react'

import { useFeatures } from '@/context/FeaturesContext'

type Profile = {
  username: string
  totpEnabled: boolean
  phone?: string
  email?: string
  role?: string
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
  const features = useFeatures()
  const [profile, setProfile] = useState<Profile | null>(null)
  const [msg, setMsg] = useState('')

  // Profile fields
  const [phone, setPhone] = useState('')
  const [profileEmail, setProfileEmail] = useState('')

  // Password fields
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  // TOTP fields
  const [totpSecret, setTotpSecret] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const [qrPng, setQrPng] = useState('')
  const [otpUrl, setOtpUrl] = useState('')
  const [disablePassword, setDisablePassword] = useState('')
  const [showTotpSetup, setShowTotpSetup] = useState(false)
  const [totpLoading, setTotpLoading] = useState(false)

  // Admin fields
  const [adminStatus, setAdminStatus] = useState<{ email: boolean; sms: boolean; usernameIsEmail: boolean; userCount: number } | null>(null)
  const [testEmailTo, setTestEmailTo] = useState('')
  const [testSmsTo, setTestSmsTo] = useState('')
  const [testEmailMsg, setTestEmailMsg] = useState('')
  const [testSmsMsg, setTestSmsMsg] = useState('')
  const [reloadMsg, setReloadMsg] = useState('')
  const [tinyauthUp, setTinyauthUp] = useState<boolean | null>(null)
  const [restarting, setRestarting] = useState(false)

  const load = async () => {
    try {
      const data = (await api.get('/account/profile')).data
      setProfile(data)
      setPhone(data.phone || '')
      setProfileEmail(data.email || '')
    } catch {
      setProfile(null)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  // Load admin status when profile indicates admin role
  useEffect(() => {
    if (profile?.role === 'admin') {
      api.get('/admin/status').then((res) => setAdminStatus(res.data)).catch(() => {})
      api.get('/admin/tinyauth-health').then((res) => setTinyauthUp(res.data.running)).catch(() => setTinyauthUp(false))
    }
  }, [profile?.role])

  // Poll tinyauth health while restarting
  useEffect(() => {
    if (!restarting) return
    const interval = setInterval(async () => {
      try {
        const res = await api.get('/admin/tinyauth-health')
        if (res.data.running) {
          setTinyauthUp(true)
          setRestarting(false)
        }
      } catch {
        setTinyauthUp(false)
      }
    }, 2000)
    return () => clearInterval(interval)
  }, [restarting])

  const startTotpSetup = async () => {
    setTotpLoading(true)
    try {
      const data = (await api.post('/account/totp/setup')).data
      setTotpSecret(data.secret)
      setQrPng(data.qrPng)
      setOtpUrl(data.otpUrl)
      setShowTotpSetup(true)
    } catch (e: any) {
      setMsg(e?.response?.data?.error || t('accountPage.genericError'))
    } finally {
      setTotpLoading(false)
    }
  }

  return (
    <Card className="w-full max-w-sm sm:max-w-md overflow-hidden">
      <CardHeader>
        <CardTitle className="text-center text-3xl">{t('accountPage.title')}</CardTitle>
        <CardDescription className="text-center">{t('accountPage.description')}</CardDescription>
      </CardHeader>
      <CardContent>
        <AnimatedHeight>
        {msg && <div className="mb-4 rounded-md border bg-muted px-3 py-2 text-sm">{msg}</div>}

        {profile && (
          <Tabs defaultValue="profile">
            <TabsList>
              <TabsTrigger value="profile" className="gap-1.5">
                <User className="h-3.5 w-3.5" />
                {t('accountPage.profile')}
              </TabsTrigger>
              <TabsTrigger value="password" className="gap-1.5">
                <Lock className="h-3.5 w-3.5" />
                {t('accountPage.tabPassword')}
              </TabsTrigger>
              <TabsTrigger value="security" className="gap-1.5">
                <Shield className="h-3.5 w-3.5" />
                {t('accountPage.tabSecurity')}
              </TabsTrigger>
              {profile.role === 'admin' && (
                <TabsTrigger value="admin" className="gap-1.5">
                  <Settings className="h-3.5 w-3.5" />
                  {t('accountPage.tabAdmin')}
                </TabsTrigger>
              )}
            </TabsList>

            {/* Tab: Profile */}
            <TabsContent value="profile">
              <div className="grid gap-3">
                <div className="grid gap-2">
                  <Label>{t('common.username')}</Label>
                  <Input value={profile.username} disabled />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="phone">{t('common.phoneNumber')}</Label>
                  <Input
                    id="phone"
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    placeholder="+31612345678"
                  />
                </div>
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
                {!features.usernameIsEmail && (
                  <>
                    <div className="grid gap-2">
                      <Label htmlFor="profileEmail">{t('accountPage.emailLabel')}</Label>
                      <Input
                        id="profileEmail"
                        type="email"
                        value={profileEmail}
                        onChange={(e) => setProfileEmail(e.target.value)}
                        placeholder="user@example.com"
                      />
                    </div>
                    <Button
                      variant="outline"
                      onClick={async () => {
                        try {
                          await api.post('/account/email', { email: profileEmail })
                          setMsg(t('accountPage.emailUpdated'))
                          void load()
                        } catch (e: any) {
                          setMsg(e?.response?.data?.error || t('accountPage.genericError'))
                        }
                      }}
                    >
                      {t('common.save')}
                    </Button>
                  </>
                )}
              </div>
            </TabsContent>

            {/* Tab: Password */}
            <TabsContent value="password">
              <div className="grid gap-3">
                <div className="grid gap-2">
                  <Label htmlFor="oldPassword">{t('accountPage.currentPassword')}</Label>
                  <Input id="oldPassword" type="password" value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} />
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
                  disabled={!newPassword || newPassword !== confirmPassword}
                  onClick={async () => {
                    try {
                      await api.post('/account/change-password', { oldPassword, newPassword })
                      setMsg(t('accountPage.passwordChanged'))
                      setOldPassword('')
                      setNewPassword('')
                      setConfirmPassword('')
                    } catch (e: any) {
                      setMsg(e?.response?.data?.error || t('accountPage.genericError'))
                    }
                  }}
                >
                  {t('accountPage.changePassword')}
                </Button>
              </div>
            </TabsContent>

            {/* Tab: Security (2FA) */}
            <TabsContent value="security">
              <div className="grid gap-4">
                {/* Status display */}
                <div className="flex items-center gap-2">
                  {profile.totpEnabled ? (
                    <>
                      <ShieldCheck className="h-5 w-5 text-green-500" />
                      <span className="font-medium text-green-700 dark:text-green-400">{t('accountPage.totpStatusEnabled')}</span>
                    </>
                  ) : (
                    <>
                      <ShieldAlert className="h-5 w-5 text-amber-500" />
                      <span className="font-medium text-amber-700 dark:text-amber-400">{t('accountPage.totpStatusDisabled')}</span>
                    </>
                  )}
                </div>

                {/* TOTP disabled: show enable flow */}
                {!profile.totpEnabled && (
                  <>
                    {!showTotpSetup ? (
                      <Button onClick={startTotpSetup} disabled={totpLoading}>
                        {t('accountPage.enableTotp')}
                      </Button>
                    ) : (
                      <div className="grid gap-3 rounded-md border p-4 overflow-hidden">
                        <p className="text-sm text-muted-foreground">{t('accountPage.totpSetupInstructions')}</p>

                        {qrPng && (
                          <img
                            src={qrPng}
                            width={220}
                            className="self-center max-w-full rounded-md border"
                            alt={t('accountPage.totpQrAlt')}
                          />
                        )}

                        {totpSecret && (
                          <div className="flex items-center gap-2 rounded-md border bg-background/45 p-2 text-xs break-all">
                            <span className="flex-1">{t('accountPage.secret')}: {totpSecret}</span>
                            <CopyButton value={totpSecret} />
                          </div>
                        )}

                        {otpUrl && (
                          <div className="flex items-center gap-2 rounded-md border bg-background/45 p-2 text-xs min-w-0">
                            <span className="flex-1 truncate min-w-0">{otpUrl}</span>
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
                                setShowTotpSetup(false)
                                setTotpSecret('')
                                setQrPng('')
                                setOtpUrl('')
                                setTotpCode('')
                                void load()
                              } catch (e: any) {
                                setMsg(e?.response?.data?.error || t('accountPage.genericError'))
                              }
                            }}
                          >
                            {t('common.enable')}
                          </Button>
                        </div>
                      </div>
                    )}
                  </>
                )}

                {/* TOTP enabled: show disable with password */}
                {profile.totpEnabled && (
                  <div className="flex flex-wrap gap-2">
                    <Input
                      type="password"
                      value={disablePassword}
                      onChange={(e) => setDisablePassword(e.target.value)}
                      placeholder={t('common.password')}
                      className="flex-1 min-w-[120px]"
                    />
                    <Button
                      variant="destructive"
                      onClick={async () => {
                        try {
                          await api.post('/account/totp/disable', { password: disablePassword })
                          setMsg(t('accountPage.totpDisabledSuccess'))
                          setDisablePassword('')
                          void load()
                        } catch (e: any) {
                          setMsg(e?.response?.data?.error || t('accountPage.genericError'))
                        }
                      }}
                    >
                      {t('accountPage.disableTotp')}
                    </Button>
                  </div>
                )}
              </div>
            </TabsContent>

            {/* Tab: Admin (admin only) */}
            {profile.role === 'admin' && (
            <TabsContent value="admin">
              <div className="grid gap-4">
                {adminStatus && (
                  <>
                    <h3 className="font-medium">{t('accountPage.adminStatus')}</h3>
                    <div className="grid gap-2">
                      <div className="flex items-center gap-2">
                        {adminStatus.email ? (
                          <><CheckCircle className="h-4 w-4 text-green-500" /><span>{t('accountPage.emailConfigured')}</span></>
                        ) : (
                          <><XCircle className="h-4 w-4 text-red-500" /><span>{t('accountPage.emailNotConfigured')}</span></>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {adminStatus.sms ? (
                          <><CheckCircle className="h-4 w-4 text-green-500" /><span>{t('accountPage.smsConfigured')}</span></>
                        ) : (
                          <><XCircle className="h-4 w-4 text-red-500" /><span>{t('accountPage.smsNotConfigured')}</span></>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      {tinyauthUp === true ? (
                        <><CheckCircle className="h-4 w-4 text-green-500" /><span>{t('accountPage.tinyauthUp')}</span></>
                      ) : tinyauthUp === false ? (
                        <><XCircle className="h-4 w-4 text-red-500" /><span>{restarting ? t('accountPage.tinyauthRestarting') : t('accountPage.tinyauthDown')}</span></>
                      ) : (
                        <span className="text-sm text-muted-foreground">…</span>
                      )}
                    </div>

                    <div className="flex items-center gap-2 flex-wrap">
                      <Button
                        variant="outline"
                        className="gap-1.5"
                        onClick={async () => {
                          setReloadMsg('')
                          try {
                            await api.post('/admin/reload-config')
                            setReloadMsg(t('accountPage.reloadSuccess'))
                            api.get('/admin/status').then((res) => setAdminStatus(res.data)).catch(() => {})
                          } catch (e: any) {
                            setReloadMsg(t('accountPage.reloadFailed') + ': ' + (e?.response?.data?.error || ''))
                          }
                        }}
                      >
                        <RefreshCw className="h-3.5 w-3.5" />
                        {t('accountPage.reloadConfig')}
                      </Button>
                      <Button
                        variant="outline"
                        className="gap-1.5"
                        disabled={restarting}
                        onClick={async () => {
                          try {
                            setRestarting(true)
                            setTinyauthUp(false)
                            await api.post('/admin/restart-tinyauth')
                          } catch (e: any) {
                            setRestarting(false)
                            setReloadMsg(t('accountPage.restartFailed') + ': ' + (e?.response?.data?.error || ''))
                          }
                        }}
                      >
                        <RefreshCw className={`h-3.5 w-3.5 ${restarting ? 'animate-spin' : ''}`} />
                        {t('accountPage.restartTinyauth')}
                      </Button>
                      {reloadMsg && <span className="text-sm">{reloadMsg}</span>}
                    </div>

                    {adminStatus.email && (
                      <div className="grid gap-2 rounded-md border p-3">
                        <Label>{t('accountPage.testEmail')}</Label>
                        <div className="flex flex-wrap gap-2">
                          <Input
                            type="email"
                            value={testEmailTo}
                            onChange={(e) => { setTestEmailTo(e.target.value); setTestEmailMsg('') }}
                            placeholder="test@example.com"
                            className="flex-1 min-w-[180px]"
                          />
                          <Button
                            variant="outline"
                            onClick={async () => {
                              try {
                                await api.post('/admin/test-email', { to: testEmailTo })
                                setTestEmailMsg(t('accountPage.testSuccess'))
                              } catch (e: any) {
                                setTestEmailMsg(t('accountPage.testFailed') + ': ' + (e?.response?.data?.error || ''))
                              }
                            }}
                          >
                            {t('accountPage.testEmail')}
                          </Button>
                        </div>
                        {testEmailMsg && <p className="text-sm">{testEmailMsg}</p>}
                      </div>
                    )}

                    {adminStatus.sms && (
                      <div className="grid gap-2 rounded-md border p-3">
                        <Label>{t('accountPage.testSms')}</Label>
                        <div className="flex flex-wrap gap-2">
                          <Input
                            type="tel"
                            value={testSmsTo}
                            onChange={(e) => { setTestSmsTo(e.target.value); setTestSmsMsg('') }}
                            placeholder="+31612345678"
                            className="flex-1 min-w-[180px]"
                          />
                          <Button
                            variant="outline"
                            onClick={async () => {
                              try {
                                await api.post('/admin/test-sms', { to: testSmsTo })
                                setTestSmsMsg(t('accountPage.testSuccess'))
                              } catch (e: any) {
                                setTestSmsMsg(t('accountPage.testFailed') + ': ' + (e?.response?.data?.error || ''))
                              }
                            }}
                          >
                            {t('accountPage.testSms')}
                          </Button>
                        </div>
                        {testSmsMsg && <p className="text-sm">{testSmsMsg}</p>}
                      </div>
                    )}
                  </>
                )}
              </div>
            </TabsContent>
            )}
          </Tabs>
        )}
        </AnimatedHeight>
      </CardContent>
    </Card>
  )
}
