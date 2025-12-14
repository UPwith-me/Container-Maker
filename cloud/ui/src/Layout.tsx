import { Outlet, Link, useLocation } from 'react-router-dom'
import { motion } from 'framer-motion'
import {
    Box,
    LayoutDashboard,
    CreditCard,
    Settings,
    Plus,
    LogOut,
    Server,
    Cloud
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { ThemeToggle } from '@/components/ThemeToggle'
import CommandPalette from '@/components/CommandPalette'

export default function Layout() {
    const location = useLocation()

    const navItems = [
        { icon: LayoutDashboard, label: 'Overview', path: '/' },
        { icon: Server, label: 'Instances', path: '/instances' },
        { icon: CreditCard, label: 'Billing', path: '/billing' },
        { icon: Settings, label: 'Settings', path: '/settings' },
    ]

    return (
        <>
            <CommandPalette />
            <div className="flex min-h-screen bg-background text-foreground font-sans selection:bg-emerald-500/30">
                {/* Sidebar */}
                <aside className="w-64 border-r border-border/40 bg-card/50 backdrop-blur-xl hidden md:flex flex-col fixed h-full z-10">
                    <div className="p-6 flex items-center gap-2 mb-6">
                        <div className="h-8 w-8 bg-emerald-500 rounded-lg flex items-center justify-center shadow-[0_0_15px_rgba(16,185,129,0.3)]">
                            <Box className="text-white h-5 w-5" />
                        </div>
                        <span className="font-bold text-lg tracking-tight">Container Maker</span>
                    </div>

                    <nav className="flex-1 px-4 space-y-1">
                        {navItems.map((item) => {
                            const isActive = location.pathname === item.path
                            return (
                                <Link
                                    key={item.path}
                                    to={item.path}
                                    className={cn(
                                        "flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-all duration-200 group relative",
                                        isActive
                                            ? "text-emerald-500 bg-emerald-500/10"
                                            : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                                    )}
                                >
                                    <item.icon className={cn("h-4 w-4", isActive ? "text-emerald-500" : "text-muted-foreground group-hover:text-foreground")} />
                                    {item.label}
                                    {isActive && (
                                        <motion.div
                                            layoutId="activeNav"
                                            className="absolute left-0 w-1 h-6 rounded-r-full bg-emerald-500"
                                            initial={{ opacity: 0 }}
                                            animate={{ opacity: 1 }}
                                            transition={{ duration: 0.2 }}
                                        />
                                    )}
                                </Link>
                            )
                        })}
                    </nav>

                    <div className="p-4 border-t border-border/40">
                        <div className="flex items-center gap-3 p-2 rounded-md bg-muted/30 mb-4">
                            <div className="h-8 w-8 rounded-full bg-gradient-to-tr from-purple-500 to-indigo-500 flex items-center justify-center text-xs font-bold text-white">
                                DU
                            </div>
                            <div className="flex-1 overflow-hidden">
                                <p className="text-sm font-medium truncate">Demo User</p>
                                <p className="text-xs text-muted-foreground truncate">demo@container-maker.dev</p>
                            </div>
                        </div>
                        <button className="flex items-center gap-2 text-sm text-muted-foreground hover:text-red-400 w-full px-2 py-1.5 transition-colors">
                            <LogOut className="h-4 w-4" />
                            <span>Sign out</span>
                        </button>
                    </div>
                </aside>

                {/* Main Content */}
                <main className="flex-1 md:ml-64 min-h-screen relative">
                    <header className="h-16 border-b border-border/40 bg-background/80 backdrop-blur-md sticky top-0 z-20 flex items-center justify-between px-8">
                        <h1 className="text-sm font-medium text-muted-foreground">
                            Cloud Control Plane <span className="mx-2 text-muted-foreground/30">/</span> <span className="text-foreground">{navItems.find(i => i.path === location.pathname)?.label || 'Dashboard'}</span>
                        </h1>

                        <div className="flex items-center gap-4">
                            <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 text-xs font-medium">
                                <Cloud className="h-3 w-3" />
                                <span>US East (N. Virginia)</span>
                            </div>
                            <ThemeToggle />
                            <Link to="/instances/new">
                                <button className="bg-foreground text-background hover:bg-foreground/90 px-4 py-2 rounded-md text-sm font-medium flex items-center gap-2 transition-colors">
                                    <Plus className="h-4 w-4" />
                                    New Instance
                                </button>
                            </Link>
                        </div>
                    </header>

                    <div className="p-8 max-w-7xl mx-auto">
                        <Outlet />
                    </div>
                </main>
            </div>
        </>
    )
}
