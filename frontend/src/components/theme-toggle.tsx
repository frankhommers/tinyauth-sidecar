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

  // Each button is h-7 w-7 (28px), gap-0.5 (2px), container has p-1 (4px)
  // Position = 4px padding + index * (28px button + 2px gap)
  const offset = 4 + activeIndex * 30

  return (
    <div
      className={cn(
        'relative inline-flex h-9 items-center rounded-full border p-1 gap-0.5',
        'bg-card/75 border-border backdrop-blur-md'
      )}
    >
      {/* Sliding indicator */}
      <div
        className="absolute h-7 w-7 rounded-full bg-primary shadow-sm transition-all duration-300 ease-in-out"
        style={{ left: `${offset}px` }}
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
