import { useState, useEffect, createContext, useContext, type ReactNode } from 'react'
import { Moon, Sun, Monitor } from 'lucide-react'

type Theme = 'dark' | 'light' | 'system'

interface ThemeContextType {
    theme: Theme
    setTheme: (theme: Theme) => void
    resolvedTheme: 'dark' | 'light'
}

const ThemeContext = createContext<ThemeContextType | null>(null)

export function ThemeProvider({ children }: { children: ReactNode }) {
    const [theme, setTheme] = useState<Theme>(() => {
        const stored = localStorage.getItem('theme')
        return (stored as Theme) || 'system'
    })

    const [resolvedTheme, setResolvedTheme] = useState<'dark' | 'light'>('dark')

    useEffect(() => {
        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

        const updateTheme = () => {
            let resolved: 'dark' | 'light'
            if (theme === 'system') {
                resolved = mediaQuery.matches ? 'dark' : 'light'
            } else {
                resolved = theme
            }
            setResolvedTheme(resolved)

            document.documentElement.classList.remove('dark', 'light')
            document.documentElement.classList.add(resolved)
        }

        updateTheme()
        localStorage.setItem('theme', theme)

        mediaQuery.addEventListener('change', updateTheme)
        return () => mediaQuery.removeEventListener('change', updateTheme)
    }, [theme])

    return (
        <ThemeContext.Provider value={{ theme, setTheme, resolvedTheme }}>
            {children}
        </ThemeContext.Provider>
    )
}

export function useTheme() {
    const context = useContext(ThemeContext)
    if (!context) {
        throw new Error('useTheme must be used within ThemeProvider')
    }
    return context
}

export function ThemeToggle() {
    const { theme, setTheme, resolvedTheme } = useTheme()
    const [open, setOpen] = useState(false)

    const themes: { value: Theme; label: string; icon: typeof Sun }[] = [
        { value: 'light', label: 'Light', icon: Sun },
        { value: 'dark', label: 'Dark', icon: Moon },
        { value: 'system', label: 'System', icon: Monitor },
    ]

    return (
        <div className="relative">
            <button
                onClick={() => setOpen(!open)}
                className="p-2 rounded-lg hover:bg-muted/50 transition-colors"
                title="Toggle theme"
            >
                {resolvedTheme === 'dark' ? (
                    <Moon className="h-4 w-4" />
                ) : (
                    <Sun className="h-4 w-4" />
                )}
            </button>

            {open && (
                <>
                    <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
                    <div className="absolute right-0 mt-2 w-36 py-1 rounded-lg border border-border bg-card shadow-lg z-50">
                        {themes.map(t => (
                            <button
                                key={t.value}
                                onClick={() => {
                                    setTheme(t.value)
                                    setOpen(false)
                                }}
                                className={`w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-muted/50 transition-colors ${theme === t.value ? 'text-emerald-500' : ''
                                    }`}
                            >
                                <t.icon className="h-4 w-4" />
                                {t.label}
                            </button>
                        ))}
                    </div>
                </>
            )}
        </div>
    )
}
