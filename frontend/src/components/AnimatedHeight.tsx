import { useRef, useEffect, useState, type ReactNode } from 'react'

export function AnimatedHeight({ children }: { children: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null)
  const [height, setHeight] = useState<number | 'auto'>('auto')

  useEffect(() => {
    if (!ref.current) return

    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setHeight(entry.contentRect.height)
      }
    })

    observer.observe(ref.current)
    return () => observer.disconnect()
  }, [])

  return (
    <div
      style={{
        height: height === 'auto' ? 'auto' : `${height}px`,
        overflow: 'hidden',
        transition: 'height 300ms ease-in-out',
      }}
    >
      <div ref={ref}>{children}</div>
    </div>
  )
}
