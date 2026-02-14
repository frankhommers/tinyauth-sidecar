import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { api } from '../api/client'

interface Features {
  signupEnabled: boolean
  smsEnabled: boolean
  loaded: boolean
}

const FeaturesContext = createContext<Features>({ signupEnabled: false, smsEnabled: false, loaded: false })

export function FeaturesProvider({ children }: { children: ReactNode }) {
  const [features, setFeatures] = useState<Features>({ signupEnabled: false, smsEnabled: false, loaded: false })

  useEffect(() => {
    api.get('/features').then((res) => {
      setFeatures({
        signupEnabled: res.data.signupEnabled ?? true,
        smsEnabled: res.data.smsEnabled ?? false,
        loaded: true,
      })
    }).catch(() => {
      setFeatures({ signupEnabled: true, smsEnabled: false, loaded: true })
    })
  }, [])

  return <FeaturesContext.Provider value={features}>{children}</FeaturesContext.Provider>
}

export function useFeatures() {
  return useContext(FeaturesContext)
}
