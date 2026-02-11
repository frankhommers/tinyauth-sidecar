import { Alert, Button, Card, CardContent, Divider, Stack, TextField, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import { api } from '../api/client'

export default function AccountPage() {
  const [profile, setProfile] = useState<any>(null)
  const [msg, setMsg] = useState('')
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [totpSecret, setTotpSecret] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const [qrPng, setQrPng] = useState('')
  const [disablePassword, setDisablePassword] = useState('')

  const load = async () => {
    try { setProfile((await api.get('/account/profile')).data) } catch { setMsg('Niet ingelogd') }
  }
  useEffect(() => { load() }, [])

  return <Stack spacing={2}>
    <Typography variant='h5'>Account</Typography>
    {msg && <Alert severity='info'>{msg}</Alert>}
    {profile && <Card><CardContent>
      <Typography>Username: {profile.username}</Typography>
      <Typography>TOTP enabled: {String(profile.totpEnabled)}</Typography>
    </CardContent></Card>}

    <Divider />
    <Typography variant='h6'>Wachtwoord wijzigen</Typography>
    <Stack direction='row' spacing={2}>
      <TextField type='password' label='Old' value={oldPassword} onChange={e => setOldPassword(e.target.value)} />
      <TextField type='password' label='New' value={newPassword} onChange={e => setNewPassword(e.target.value)} />
      <Button variant='contained' onClick={async () => {
        await api.post('/account/change-password', { oldPassword, newPassword })
        setMsg('Wachtwoord gewijzigd')
      }}>Change</Button>
    </Stack>

    <Divider />
    <Typography variant='h6'>TOTP setup</Typography>
    <Button variant='outlined' onClick={async () => {
      const data = (await api.post('/account/totp/setup')).data
      setTotpSecret(data.secret); setQrPng(data.qrPng)
    }}>Generate secret</Button>
    {qrPng && <img src={qrPng} width={220} />}
    {totpSecret && <Typography>Secret: {totpSecret}</Typography>}
    <Stack direction='row' spacing={2}>
      <TextField label='Code' value={totpCode} onChange={e => setTotpCode(e.target.value)} />
      <Button variant='contained' onClick={async () => {
        await api.post('/account/totp/enable', { secret: totpSecret, code: totpCode })
        setMsg('TOTP enabled')
        load()
      }}>Enable</Button>
    </Stack>

    <Stack direction='row' spacing={2}>
      <TextField type='password' label='Password' value={disablePassword} onChange={e => setDisablePassword(e.target.value)} />
      <Button variant='outlined' onClick={async () => {
        await api.post('/account/totp/disable', { password: disablePassword })
        setMsg('TOTP disabled')
        load()
      }}>Disable</Button>
    </Stack>
  </Stack>
}
