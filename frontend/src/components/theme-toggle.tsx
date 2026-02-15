import { Moon, Sun, Monitor } from 'lucide-react'
import { useTheme } from '@/components/providers/theme-provider'
import { cn } from '@/lib/utils'

const modes = [
  { value: 'system', Icon: Monitor },
  { value: 'light', Icon: Sun },
  { value: 'dark', Icon: Moon },
] as const

export function ThemeToggle() {
  const { theme, setTheme } = useTheme()

  return (
    <div
      className={cn(
        'inline-flex h-9 items-center rounded-full border p-1 gap-0.5',
        'bg-card/75 border-border backdrop-blur-md'
      )}
    >
      {modes.map(({ value, Icon }) => (
        <button
          key={value}
          type="button"
          aria-label={value}
          aria-pressed={theme === value}
          onClick={() => setTheme(value)}
          className={cn(
            'inline-flex h-7 w-7 items-center justify-center rounded-full transition-all duration-200',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            theme === value
              ? 'bg-primary text-primary-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          <Icon className="h-3.5 w-3.5" />
        </button>
      ))}
    </div>
  )
}
