import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { api } from '../api/client'

interface Features {
  smsEnabled: boolean
  emailEnabled: boolean
  usernameIsEmail: boolean
  backgroundImage: string
  title: string
  loaded: boolean
}

const defaults: Features = { smsEnabled: false, emailEnabled: false, usernameIsEmail: true, backgroundImage: '', title: '', loaded: false }
const FeaturesContext = createContext<Features>(defaults)

export function FeaturesProvider({ children }: { children: ReactNode }) {
  const [features, setFeatures] = useState<Features>(defaults)

  useEffect(() => {
    api.get('/features').then((res) => {
      setFeatures({
        smsEnabled: res.data.smsEnabled ?? false,
        emailEnabled: res.data.emailEnabled ?? false,
        usernameIsEmail: res.data.usernameIsEmail ?? true,
        backgroundImage: res.data.backgroundImage ?? '',
        title: res.data.title ?? '',
        loaded: true,
      })
    }).catch(() => {
      setFeatures({ ...defaults, loaded: true })
    })
  }, [])

  return <FeaturesContext.Provider value={features}>{children}</FeaturesContext.Provider>
}

export function useFeatures() {
  return useContext(FeaturesContext)
}
