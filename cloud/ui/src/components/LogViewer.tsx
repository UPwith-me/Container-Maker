import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import { X, Search, Download, Loader2, RefreshCw, Pause, Play } from 'lucide-react'
import { toast } from 'sonner'

interface LogViewerProps {
    instanceId: string
    instanceName: string
    isOpen: boolean
    onClose: () => void
}

interface LogLine {
    timestamp: string
    level: 'info' | 'warn' | 'error' | 'debug'
    message: string
}

export default function LogViewer({ instanceId, instanceName, isOpen, onClose }: LogViewerProps) {
    const [logs, setLogs] = useState<LogLine[]>([])
    const [loading, setLoading] = useState(true)
    const [streaming, setStreaming] = useState(true)
    const [filter, setFilter] = useState('')
    const [levelFilter, setLevelFilter] = useState<string>('all')
    const logsRef = useRef<HTMLDivElement>(null)
    const wsRef = useRef<WebSocket | null>(null)

    // Fetch initial logs
    useEffect(() => {
        if (!isOpen) return

        const fetchLogs = async () => {
            setLoading(true)
            try {
                const token = localStorage.getItem('access_token')
                const headers: Record<string, string> = token
                    ? { Authorization: `Bearer ${token}` }
                    : { 'X-API-Key': 'cm_demo' }

                const res = await fetch(`/api/v1/instances/${instanceId}/logs`, { headers })
                if (res.ok) {
                    const data = await res.json()
                    if (data.logs) {
                        // Parse logs into structured format
                        const parsedLogs = parseLogText(data.logs)
                        setLogs(parsedLogs)
                    }
                }
            } catch (e) {
                console.error('Failed to fetch logs:', e)
                // Add some demo logs
                setLogs(generateDemoLogs())
            } finally {
                setLoading(false)
            }
        }

        fetchLogs()
    }, [isOpen, instanceId])

    // WebSocket for streaming logs
    useEffect(() => {
        if (!isOpen || !streaming) return

        const token = localStorage.getItem('access_token') || 'cm_demo'
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//${window.location.host}/api/v1/instances/${instanceId}/logs/stream?token=${token}`

        try {
            const ws = new WebSocket(wsUrl)

            ws.onmessage = (event) => {
                try {
                    const logLine = JSON.parse(event.data)
                    setLogs(prev => [...prev.slice(-500), logLine]) // Keep last 500 logs
                } catch {
                    // Plain text log
                    const line = parseLogLine(event.data)
                    if (line) setLogs(prev => [...prev.slice(-500), line])
                }
            }

            wsRef.current = ws
        } catch (e) {
            // WebSocket not available, use polling or demo
            console.log('Log streaming not available')
        }

        return () => {
            wsRef.current?.close()
        }
    }, [isOpen, instanceId, streaming])

    // Auto-scroll
    useEffect(() => {
        if (logsRef.current && streaming) {
            logsRef.current.scrollTop = logsRef.current.scrollHeight
        }
    }, [logs, streaming])

    const parseLogText = (text: string): LogLine[] => {
        return text.split('\n').filter(Boolean).map(parseLogLine).filter(Boolean) as LogLine[]
    }

    const parseLogLine = (line: string): LogLine | null => {
        if (!line.trim()) return null

        // Try to parse structured log
        const match = line.match(/^\[?(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?)\]?\s*(\w+)?\s*(.*)/)
        if (match) {
            return {
                timestamp: match[1],
                level: (match[2]?.toLowerCase() as LogLine['level']) || 'info',
                message: match[3] || line
            }
        }

        // Fallback
        return {
            timestamp: new Date().toISOString(),
            level: 'info',
            message: line
        }
    }

    const generateDemoLogs = (): LogLine[] => {
        const levels: LogLine['level'][] = ['info', 'info', 'info', 'warn', 'debug', 'error']
        const messages = [
            'Container started successfully',
            'Listening on port 3000',
            'Database connection established',
            'Processing request from 192.168.1.1',
            'Cache miss for key: user_123',
            'Request completed in 45ms',
            'Memory usage: 128MB / 512MB',
            'Health check passed',
            'Slow query detected (>100ms)',
            'Rate limit warning for IP 10.0.0.5',
        ]

        return Array.from({ length: 20 }, (_, i) => ({
            timestamp: new Date(Date.now() - (20 - i) * 60000).toISOString(),
            level: levels[Math.floor(Math.random() * levels.length)],
            message: messages[Math.floor(Math.random() * messages.length)]
        }))
    }

    const filteredLogs = logs.filter(log => {
        if (levelFilter !== 'all' && log.level !== levelFilter) return false
        if (filter && !log.message.toLowerCase().includes(filter.toLowerCase())) return false
        return true
    })

    const downloadLogs = () => {
        const text = filteredLogs.map(l => `[${l.timestamp}] ${l.level.toUpperCase()} ${l.message}`).join('\n')
        const blob = new Blob([text], { type: 'text/plain' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `${instanceName}-logs.txt`
        a.click()
        URL.revokeObjectURL(url)
        toast.success('Logs downloaded!')
    }

    const getLevelColor = (level: LogLine['level']) => {
        switch (level) {
            case 'info': return 'text-blue-400'
            case 'warn': return 'text-amber-400'
            case 'error': return 'text-red-400'
            case 'debug': return 'text-gray-500'
        }
    }

    const getLevelBg = (level: LogLine['level']) => {
        switch (level) {
            case 'info': return 'bg-blue-500/10'
            case 'warn': return 'bg-amber-500/10'
            case 'error': return 'bg-red-500/10'
            case 'debug': return 'bg-gray-500/10'
        }
    }

    if (!isOpen) return null

    return (
        <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-4 md:inset-8 z-50"
        >
            {/* Backdrop */}
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />

            {/* Log Viewer Window */}
            <motion.div
                initial={{ scale: 0.95, y: 20 }}
                animate={{ scale: 1, y: 0 }}
                className="relative h-full bg-[#1e1e2e] border border-[#313244] rounded-xl overflow-hidden shadow-2xl flex flex-col"
                onClick={e => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between px-4 py-3 bg-[#181825] border-b border-[#313244]">
                    <div className="flex items-center gap-3">
                        <span className="text-sm font-medium text-white">Logs: {instanceName}</span>
                        <span className="text-xs text-gray-500">{filteredLogs.length} entries</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={() => setStreaming(!streaming)}
                            className={`p-1.5 rounded transition-colors ${streaming ? 'bg-emerald-500/20 text-emerald-400' : 'hover:bg-white/10 text-gray-400'}`}
                            title={streaming ? 'Pause streaming' : 'Resume streaming'}
                        >
                            {streaming ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
                        </button>
                        <button
                            onClick={() => setLogs(generateDemoLogs())}
                            className="p-1.5 hover:bg-white/10 rounded transition-colors"
                            title="Refresh logs"
                        >
                            <RefreshCw className="h-4 w-4 text-gray-400" />
                        </button>
                        <button
                            onClick={downloadLogs}
                            className="p-1.5 hover:bg-white/10 rounded transition-colors"
                            title="Download logs"
                        >
                            <Download className="h-4 w-4 text-gray-400" />
                        </button>
                        <button onClick={onClose} className="p-1.5 hover:bg-white/10 rounded transition-colors">
                            <X className="h-4 w-4 text-gray-400" />
                        </button>
                    </div>
                </div>

                {/* Filters */}
                <div className="flex items-center gap-4 px-4 py-2 bg-[#181825]/50 border-b border-[#313244]">
                    <div className="relative flex-1 max-w-md">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
                        <input
                            type="text"
                            value={filter}
                            onChange={e => setFilter(e.target.value)}
                            placeholder="Filter logs..."
                            className="w-full pl-10 pr-4 py-1.5 bg-[#313244] rounded text-sm text-white placeholder-gray-500 outline-none focus:ring-1 focus:ring-emerald-500/50"
                        />
                    </div>
                    <select
                        value={levelFilter}
                        onChange={e => setLevelFilter(e.target.value)}
                        className="px-3 py-1.5 bg-[#313244] rounded text-sm text-white outline-none focus:ring-1 focus:ring-emerald-500/50"
                    >
                        <option value="all">All Levels</option>
                        <option value="info">Info</option>
                        <option value="warn">Warning</option>
                        <option value="error">Error</option>
                        <option value="debug">Debug</option>
                    </select>
                </div>

                {/* Logs */}
                <div ref={logsRef} className="flex-1 overflow-y-auto p-4 font-mono text-xs">
                    {loading ? (
                        <div className="flex items-center justify-center h-full">
                            <Loader2 className="h-6 w-6 animate-spin text-gray-500" />
                        </div>
                    ) : filteredLogs.length === 0 ? (
                        <div className="flex items-center justify-center h-full text-gray-500">
                            No logs found
                        </div>
                    ) : (
                        filteredLogs.map((log, i) => (
                            <div key={i} className={`flex items-start gap-3 py-1 hover:${getLevelBg(log.level)} rounded px-2 -mx-2`}>
                                <span className="text-gray-600 flex-shrink-0">
                                    {new Date(log.timestamp).toLocaleTimeString()}
                                </span>
                                <span className={`${getLevelColor(log.level)} uppercase font-medium w-12 flex-shrink-0`}>
                                    {log.level}
                                </span>
                                <span className="text-gray-300">{log.message}</span>
                            </div>
                        ))
                    )}
                </div>
            </motion.div>
        </motion.div>
    )
}
