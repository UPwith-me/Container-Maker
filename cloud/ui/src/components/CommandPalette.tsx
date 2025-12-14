import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import {
    Search,
    Plus,
    Settings,
    CreditCard,
    Server,
    Command,
    ArrowRight
} from 'lucide-react'

interface CommandItem {
    id: string
    title: string
    icon: typeof Search
    action: () => void
    shortcut?: string
}

export default function CommandPalette() {
    const [open, setOpen] = useState(false)
    const [query, setQuery] = useState('')
    const navigate = useNavigate()

    const commands: CommandItem[] = [
        { id: 'new', title: 'New Instance', icon: Plus, action: () => navigate('/instances/new'), shortcut: 'N' },
        { id: 'dashboard', title: 'Go to Dashboard', icon: Server, action: () => navigate('/'), shortcut: 'D' },
        { id: 'settings', title: 'Open Settings', icon: Settings, action: () => navigate('/settings'), shortcut: 'S' },
        { id: 'billing', title: 'View Billing', icon: CreditCard, action: () => navigate('/billing'), shortcut: 'B' },
    ]

    const filteredCommands = commands.filter(cmd =>
        cmd.title.toLowerCase().includes(query.toLowerCase())
    )

    const handleKeyDown = useCallback((e: KeyboardEvent) => {
        // Open with Cmd+K or Ctrl+K
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
            e.preventDefault()
            setOpen(o => !o)
        }
        // Close with Escape
        if (e.key === 'Escape') {
            setOpen(false)
        }
    }, [])

    useEffect(() => {
        document.addEventListener('keydown', handleKeyDown)
        return () => document.removeEventListener('keydown', handleKeyDown)
    }, [handleKeyDown])

    const executeCommand = (cmd: CommandItem) => {
        cmd.action()
        setOpen(false)
        setQuery('')
    }

    return (
        <AnimatePresence>
            {open && (
                <>
                    {/* Backdrop */}
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50"
                        onClick={() => setOpen(false)}
                    />

                    {/* Modal */}
                    <motion.div
                        initial={{ opacity: 0, scale: 0.95, y: -20 }}
                        animate={{ opacity: 1, scale: 1, y: 0 }}
                        exit={{ opacity: 0, scale: 0.95, y: -20 }}
                        className="fixed top-[20%] left-1/2 -translate-x-1/2 w-full max-w-lg z-50"
                    >
                        <div className="bg-card border border-border rounded-xl shadow-2xl overflow-hidden">
                            {/* Search Input */}
                            <div className="flex items-center gap-3 px-4 py-3 border-b border-border">
                                <Search className="h-5 w-5 text-muted-foreground" />
                                <input
                                    type="text"
                                    value={query}
                                    onChange={(e) => setQuery(e.target.value)}
                                    placeholder="Type a command or search..."
                                    className="flex-1 bg-transparent outline-none text-foreground placeholder:text-muted-foreground"
                                    autoFocus
                                />
                                <kbd className="px-2 py-0.5 text-xs bg-muted rounded border border-border text-muted-foreground">
                                    ESC
                                </kbd>
                            </div>

                            {/* Commands */}
                            <div className="max-h-80 overflow-y-auto py-2">
                                {filteredCommands.length === 0 ? (
                                    <div className="px-4 py-8 text-center text-muted-foreground">
                                        No commands found
                                    </div>
                                ) : (
                                    filteredCommands.map(cmd => (
                                        <button
                                            key={cmd.id}
                                            onClick={() => executeCommand(cmd)}
                                            className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-muted/50 transition-colors text-left"
                                        >
                                            <cmd.icon className="h-4 w-4 text-muted-foreground" />
                                            <span className="flex-1">{cmd.title}</span>
                                            {cmd.shortcut && (
                                                <kbd className="px-2 py-0.5 text-xs bg-muted rounded border border-border text-muted-foreground">
                                                    {cmd.shortcut}
                                                </kbd>
                                            )}
                                            <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100" />
                                        </button>
                                    ))
                                )}
                            </div>

                            {/* Footer */}
                            <div className="px-4 py-2 border-t border-border bg-muted/30 flex items-center gap-4 text-xs text-muted-foreground">
                                <span className="flex items-center gap-1">
                                    <Command className="h-3 w-3" />
                                    <span>K to toggle</span>
                                </span>
                                <span>↑↓ to navigate</span>
                                <span>↵ to select</span>
                            </div>
                        </div>
                    </motion.div>
                </>
            )}
        </AnimatePresence>
    )
}
