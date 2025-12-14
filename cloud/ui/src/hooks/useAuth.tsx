import { useState, useEffect, createContext, useContext, type ReactNode } from 'react'

interface User {
    id: string
    email: string
    name: string
    avatar_url?: string
}

interface AuthContextType {
    user: User | null
    isAuthenticated: boolean
    isLoading: boolean
    login: (email: string, password: string) => Promise<void>
    register: (email: string, password: string, name: string) => Promise<void>
    logout: () => void
    refreshToken: () => Promise<boolean>
    getAuthHeaders: () => Record<string, string>
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null)
    const [isLoading, setIsLoading] = useState(true)

    useEffect(() => {
        // Check for existing token on mount
        checkAuth()
    }, [])

    const checkAuth = async () => {
        const token = localStorage.getItem('access_token')
        if (!token) {
            setIsLoading(false)
            return
        }

        try {
            const res = await fetch('/api/v1/user', {
                headers: { Authorization: `Bearer ${token}` }
            })
            if (res.ok) {
                const userData = await res.json()
                setUser(userData)
            } else if (res.status === 401) {
                // Try to refresh
                const refreshed = await refreshToken()
                if (!refreshed) {
                    logout()
                }
            }
        } catch (e) {
            console.error('Auth check failed:', e)
        } finally {
            setIsLoading(false)
        }
    }

    const login = async (email: string, password: string) => {
        const res = await fetch('/api/v1/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        })

        if (!res.ok) {
            const data = await res.json()
            throw new Error(data.message || 'Login failed')
        }

        const data = await res.json()
        localStorage.setItem('access_token', data.access_token)
        localStorage.setItem('refresh_token', data.refresh_token)

        // Fetch user data
        await checkAuth()
    }

    const register = async (email: string, password: string, name: string) => {
        const res = await fetch('/api/v1/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password, name })
        })

        if (!res.ok) {
            const data = await res.json()
            throw new Error(data.message || 'Registration failed')
        }

        // Auto-login after registration
        await login(email, password)
    }

    const logout = () => {
        const token = localStorage.getItem('refresh_token')
        if (token) {
            fetch('/api/v1/auth/logout', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ refresh_token: token })
            }).catch(() => { })
        }

        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        setUser(null)
    }

    const refreshToken = async (): Promise<boolean> => {
        const token = localStorage.getItem('refresh_token')
        if (!token) return false

        try {
            const res = await fetch('/api/v1/auth/refresh', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ refresh_token: token })
            })

            if (res.ok) {
                const data = await res.json()
                localStorage.setItem('access_token', data.access_token)
                localStorage.setItem('refresh_token', data.refresh_token)
                return true
            }
        } catch (e) {
            console.error('Token refresh failed:', e)
        }
        return false
    }

    const getAuthHeaders = (): Record<string, string> => {
        const token = localStorage.getItem('access_token')
        if (token) {
            return { Authorization: `Bearer ${token}` }
        }
        return {}
    }

    return (
        <AuthContext.Provider value={{
            user,
            isAuthenticated: !!user,
            isLoading,
            login,
            register,
            logout,
            refreshToken,
            getAuthHeaders
        }}>
            {children}
        </AuthContext.Provider>
    )
}

export function useAuth() {
    const context = useContext(AuthContext)
    if (!context) {
        throw new Error('useAuth must be used within AuthProvider')
    }
    return context
}

// For demo mode when not authenticated
export function useDemoAuth() {
    return {
        getAuthHeaders: (): Record<string, string> => {
            const token = localStorage.getItem('access_token')
            if (token) {
                return { Authorization: `Bearer ${token}` }
            }
            // Fallback to demo key if not logged in
            return { 'X-API-Key': 'cm_demo' }
        }
    }
}
