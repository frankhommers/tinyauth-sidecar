import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { api } from '../api/client'

interface Features {
  signupEnabled: boolean
  smsEnabled: boolean
  emailEnabled: boolean
  usernameIsEmail: boolean
  loaded: boolean
}

const FeaturesContext = createContext<Features>({ signupEnabled: false, smsEnabled: false, emailEnabled: false, usernameIsEmail: true, loaded: false })

export function FeaturesProvider({ children }: { children: ReactNode }) {
  const [features, setFeatures] = useState<Features>({ signupEnabled: false, smsEnabled: false, emailEnabled: false, usernameIsEmail: true, loaded: false })

  useEffect(() => {
    api.get('/features').then((res) => {
      setFeatures({
        signupEnabled: res.data.signupEnabled ?? true,
        smsEnabled: res.data.smsEnabled ?? false,
        emailEnabled: res.data.emailEnabled ?? false,
        usernameIsEmail: res.data.usernameIsEmail ?? true,
        loaded: true,
      })
    }).catch(() => {
      setFeatures({ signupEnabled: true, smsEnabled: false, emailEnabled: false, usernameIsEmail: true, loaded: true })
    })
  }, [])

  return <FeaturesContext.Provider value={features}>{children}</FeaturesContext.Provider>
}

export function useFeatures() {
  return useContext(FeaturesContext)
}
