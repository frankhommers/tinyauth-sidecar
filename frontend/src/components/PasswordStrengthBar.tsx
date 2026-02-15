import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { zxcvbn, zxcvbnOptions } from '@zxcvbn-ts/core'
import * as zxcvbnCommonPackage from '@zxcvbn-ts/language-common'
import * as zxcvbnEnPackage from '@zxcvbn-ts/language-en'

zxcvbnOptions.setOptions({
  graphs: zxcvbnCommonPackage.adjacencyGraphs,
  dictionary: {
    ...zxcvbnCommonPackage.dictionary,
    ...zxcvbnEnPackage.dictionary,
  },
})

const colors = ['bg-red-500', 'bg-orange-500', 'bg-amber-500', 'bg-yellow-400', 'bg-green-500']

export function PasswordStrengthBar({ password }: { password: string }) {
  const { t } = useTranslation()

  const result = useMemo(() => {
    if (!password) return null
    return zxcvbn(password)
  }, [password])

  if (!result) return null

  const score = result.score // 0-4
  const label = t(`password.strength${score}`)

  return (
    <div className="space-y-1">
      <div className="flex gap-1 h-1.5">
        {[0, 1, 2, 3].map((i) => (
          <div
            key={i}
            className={`flex-1 rounded-full transition-colors ${
              i <= score ? colors[score] : 'bg-muted'
            }`}
          />
        ))}
      </div>
      <p className="text-xs text-muted-foreground">{label}</p>
    </div>
  )
}
