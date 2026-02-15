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
  const activeIndex = modes.findIndex((m) => m.value === theme)

  return (
    <div
      className={cn(
        'relative inline-flex h-9 items-center rounded-full border p-1',
        'bg-card/75 border-border backdrop-blur-md'
      )}
    >
      {/* Sliding indicator */}
      <div
        className="absolute h-7 w-7 rounded-full bg-primary shadow-sm transition-transform duration-300 ease-in-out"
        style={{ transform: `translateX(${activeIndex * 30}px)` }}
      />

      {modes.map(({ value, Icon }) => (
        <button
          key={value}
          type="button"
          aria-label={value}
          aria-pressed={theme === value}
          onClick={() => setTheme(value)}
          className={cn(
            'relative z-10 inline-flex h-7 w-7 items-center justify-center rounded-full transition-colors duration-300',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            theme === value
              ? 'text-primary-foreground'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          <Icon className="h-3.5 w-3.5" />
        </button>
      ))}
    </div>
  )
}
