import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Route, Routes, Navigate } from 'react-router-dom'
import { ThemeProvider } from '@/components/providers/theme-provider'
import { FeaturesProvider } from './context/FeaturesContext'
import { AuthProvider } from './context/AuthContext'
import { Layout } from './components/Layout'
import ResetPasswordPage from './pages/ResetPasswordPage'
import AccountPage from './pages/AccountPage'
import './i18n'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ThemeProvider defaultTheme="system" storageKey="tinyauth-theme">
      <FeaturesProvider>
        <AuthProvider>
          <BrowserRouter basename="/manage">
            <Layout>
            <Routes>
              <Route path='/' element={<Navigate to="/account" replace />} />
              <Route path='/reset-password' element={<ResetPasswordPage />} />
              <Route path='/account' element={<AccountPage />} />
            </Routes>
            </Layout>
          </BrowserRouter>
        </AuthProvider>
      </FeaturesProvider>
    </ThemeProvider>
  </React.StrictMode>,
)
