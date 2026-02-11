import { Alert, Button, Divider, Stack, TextField, Typography } from '@mui/material'
import { useState } from 'react'
import { api } from '../api/client'

export default function ResetPasswordPage() {
  const [username, setUsername] = useState('')
  const [token, setToken] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [msg, setMsg] = useState('')

  return <Stack spacing={2} maxWidth={460}>
    <Typography variant='h5'>Password Reset</Typography>
    {msg && <Alert severity='info'>{msg}</Alert>}
    <TextField label='Username/email' value={username} onChange={e => setUsername(e.target.value)} />
    <Button variant='outlined' onClick={async () => {
      await api.post('/password-reset/request', { username })
      setMsg('Als user bestaat is mail verstuurd (of in logs)')
    }}>Request reset</Button>
    <Divider />
    <TextField label='Token' value={token} onChange={e => setToken(e.target.value)} />
    <TextField type='password' label='New password' value={newPassword} onChange={e => setNewPassword(e.target.value)} />
    <Button variant='contained' onClick={async () => {
      try {
        await api.post('/password-reset/confirm', { token, newPassword })
        setMsg('Wachtwoord gewijzigd')
      } catch (e: any) {
        setMsg(e?.response?.data?.error || 'Reset mislukt')
      }
    }}>Reset password</Button>
  </Stack>
}
