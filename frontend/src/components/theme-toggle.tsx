import { Moon, Sun, Monitor } from 'lucide-react'
import { useTheme } from '@/components/providers/theme-provider'
import { cn } from '@/lib/utils'

const modes = ['light', 'dark', 'system'] as const

export function ThemeToggle() {
  const { theme, setTheme } = useTheme()

  const currentIndex = modes.indexOf(theme)
  const next = () => setTheme(modes[(currentIndex + 1) % modes.length])

  // Resolve effective appearance for styling
  const effectiveDark =
    theme === 'dark' ||
    (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)

  const Icon = theme === 'system' ? Monitor : theme === 'dark' ? Moon : Sun
  const label = theme === 'system' ? 'System' : theme === 'dark' ? 'Dark' : 'Light'

  return (
    <button
      type="button"
      aria-label={`Theme: ${label}. Click to change.`}
      onClick={next}
      className={cn(
        'inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-xs font-medium transition-colors',
        'bg-card/75 border-border backdrop-blur-md',
        'hover:bg-accent hover:text-accent-foreground',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        effectiveDark ? 'text-foreground' : 'text-foreground'
      )}
    >
      <Icon className="h-3.5 w-3.5" />
      <span className="hidden sm:inline">{label}</span>
    </button>
  )
}
