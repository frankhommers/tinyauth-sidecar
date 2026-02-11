import { AppBar, Box, Button, Container, Toolbar, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import type { ReactNode } from 'react'

export function Layout({ children }: { children: ReactNode }) {
  return (
    <Box>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" sx={{ flexGrow: 1 }}>tinyauth-usermanagement</Typography>
          <Button color="inherit" component={RouterLink} to="/">Login</Button>
          <Button color="inherit" component={RouterLink} to="/signup">Signup</Button>
          <Button color="inherit" component={RouterLink} to="/reset-password">Reset</Button>
          <Button color="inherit" component={RouterLink} to="/account">Account</Button>
        </Toolbar>
      </AppBar>
      <Container sx={{ mt: 4 }}>{children}</Container>
    </Box>
  )
}
