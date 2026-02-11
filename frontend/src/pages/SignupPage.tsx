import { Alert, Button, Stack, TextField, Typography } from '@mui/material'
import { useState } from 'react'
import { api } from '../api/client'

export default function SignupPage() {
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [msg, setMsg] = useState('')
  const submit = async () => {
    try {
      const res = await api.post('/signup', { username, email, password })
      setMsg(`Signup status: ${res.data.status}`)
    } catch (e: any) {
      setMsg(e?.response?.data?.error || 'Signup mislukt')
    }
  }
  return <Stack spacing={2} maxWidth={420}>
    <Typography variant='h5'>Signup</Typography>
    {msg && <Alert severity='info'>{msg}</Alert>}
    <TextField label='Username' value={username} onChange={e => setUsername(e.target.value)} />
    <TextField label='Email' value={email} onChange={e => setEmail(e.target.value)} />
    <TextField type='password' label='Password' value={password} onChange={e => setPassword(e.target.value)} />
    <Button variant='contained' onClick={submit}>Signup</Button>
  </Stack>
}
