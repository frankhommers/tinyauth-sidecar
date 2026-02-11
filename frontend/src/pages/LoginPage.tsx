import { Alert, Button, Stack, TextField, Typography } from '@mui/material'
import { useState } from 'react'
import { api } from '../api/client'

export default function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [msg, setMsg] = useState('')

  const submit = async () => {
    try {
      await api.post('/auth/login', { username, password })
      setMsg('Ingelogd')
    } catch (e: any) {
      setMsg(e?.response?.data?.error || 'Login mislukt')
    }
  }

  return <Stack spacing={2} maxWidth={420}>
    <Typography variant="h5">Login</Typography>
    {msg && <Alert severity="info">{msg}</Alert>}
    <TextField label="Username" value={username} onChange={e => setUsername(e.target.value)} />
    <TextField type="password" label="Password" value={password} onChange={e => setPassword(e.target.value)} />
    <Button variant="contained" onClick={submit}>Login</Button>
  </Stack>
}
