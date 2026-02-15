import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { api } from '../api/client'

interface Features {
  signupEnabled: boolean
  smsEnabled: boolean
  emailEnabled: boolean
  usernameIsEmail: boolean
  backgroundImage: string
  loaded: boolean
}

const defaults: Features = { signupEnabled: false, smsEnabled: false, emailEnabled: false, usernameIsEmail: true, backgroundImage: '', loaded: false }
const FeaturesContext = createContext<Features>(defaults)

export function FeaturesProvider({ children }: { children: ReactNode }) {
  const [features, setFeatures] = useState<Features>(defaults)

  useEffect(() => {
    api.get('/features').then((res) => {
      setFeatures({
        signupEnabled: res.data.signupEnabled ?? true,
        smsEnabled: res.data.smsEnabled ?? false,
        emailEnabled: res.data.emailEnabled ?? false,
        usernameIsEmail: res.data.usernameIsEmail ?? true,
        backgroundImage: res.data.backgroundImage ?? '',
        loaded: true,
      })
    }).catch(() => {
      setFeatures({ ...defaults, signupEnabled: true, loaded: true })
    })
  }, [])

  return <FeaturesContext.Provider value={features}>{children}</FeaturesContext.Provider>
}

export function useFeatures() {
  return useContext(FeaturesContext)
}
