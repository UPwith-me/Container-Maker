import { useState, useRef, useEffect, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, Terminal as TerminalIcon, Maximize2, Minimize2, Copy, Check } from 'lucide-react'
import { toast } from 'sonner'

interface TerminalProps {
    instanceId: string
    instanceName: string
    isOpen: boolean
    onClose: () => void
}

interface TerminalLine {
    type: 'input' | 'output' | 'error' | 'system'
    content: string
    timestamp: Date
}

export default function Terminal({ instanceId, instanceName, isOpen, onClose }: TerminalProps) {
    const [lines, setLines] = useState<TerminalLine[]>([
        { type: 'system', content: `Connecting to ${instanceName}...`, timestamp: new Date() }
    ])
    const [input, setInput] = useState('')
    const [isFullscreen, setIsFullscreen] = useState(false)
    const [isConnected, setIsConnected] = useState(false)
    const [copied, setCopied] = useState(false)
    const inputRef = useRef<HTMLInputElement>(null)
    const terminalRef = useRef<HTMLDivElement>(null)
    const wsRef = useRef<WebSocket | null>(null)

    // Connect to WebSocket for terminal
    useEffect(() => {
        if (!isOpen) return

        const token = localStorage.getItem('access_token') || 'cm_demo'
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//${window.location.host}/api/v1/instances/${instanceId}/terminal?token=${token}`

        try {
            const ws = new WebSocket(wsUrl)

            ws.onopen = () => {
                setIsConnected(true)
                addLine('system', `Connected to ${instanceName}`)
                addLine('system', 'Type "help" for available commands')
            }

            ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data)
                    if (data.type === 'output') {
                        addLine('output', data.content)
                    } else if (data.type === 'error') {
                        addLine('error', data.content)
                    }
                } catch {
                    // Plain text output
                    addLine('output', event.data)
                }
            }

            ws.onclose = () => {
                setIsConnected(false)
                addLine('system', 'Connection closed')
            }

            ws.onerror = () => {
                addLine('error', 'Connection error')
            }

            wsRef.current = ws
        } catch (e) {
            // Fallback to simulated terminal if WebSocket fails
            setIsConnected(true)
            addLine('system', '[Simulated Terminal Mode]')
            addLine('system', 'Type commands to interact with the container')
        }

        return () => {
            wsRef.current?.close()
        }
    }, [isOpen, instanceId, instanceName])

    // Auto-scroll to bottom
    useEffect(() => {
        if (terminalRef.current) {
            terminalRef.current.scrollTop = terminalRef.current.scrollHeight
        }
    }, [lines])

    // Focus input when opened
    useEffect(() => {
        if (isOpen) {
            setTimeout(() => inputRef.current?.focus(), 100)
        }
    }, [isOpen])

    const addLine = useCallback((type: TerminalLine['type'], content: string) => {
        setLines(prev => [...prev, { type, content, timestamp: new Date() }])
    }, [])

    const executeCommand = (cmd: string) => {
        if (!cmd.trim()) return

        addLine('input', `$ ${cmd}`)

        // Send to WebSocket if connected
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'command', content: cmd }))
        } else {
            // Simulated responses for demo
            simulateCommand(cmd)
        }

        setInput('')
    }

    const simulateCommand = (cmd: string) => {
        const [command, ...args] = cmd.trim().split(' ')

        switch (command.toLowerCase()) {
            case 'help':
                addLine('output', 'Available commands:')
                addLine('output', '  help     - Show this help')
                addLine('output', '  ls       - List files')
                addLine('output', '  pwd      - Print working directory')
                addLine('output', '  whoami   - Show current user')
                addLine('output', '  date     - Show current date')
                addLine('output', '  env      - Show environment variables')
                addLine('output', '  ps       - List processes')
                addLine('output', '  clear    - Clear terminal')
                addLine('output', '  exit     - Close terminal')
                break
            case 'ls':
                addLine('output', 'app/  data/  config/  logs/')
                break
            case 'pwd':
                addLine('output', '/home/container')
                break
            case 'whoami':
                addLine('output', 'container')
                break
            case 'date':
                addLine('output', new Date().toString())
                break
            case 'env':
                addLine('output', 'NODE_ENV=production')
                addLine('output', 'PORT=3000')
                addLine('output', 'INSTANCE_ID=' + instanceId)
                break
            case 'ps':
                addLine('output', 'PID   CMD')
                addLine('output', '1     /bin/sh')
                addLine('output', '12    node app.js')
                break
            case 'clear':
                setLines([])
                break
            case 'exit':
                onClose()
                break
            case 'echo':
                addLine('output', args.join(' '))
                break
            case 'cat':
                if (args[0]) {
                    addLine('output', `Contents of ${args[0]}...`)
                } else {
                    addLine('error', 'cat: missing file operand')
                }
                break
            default:
                addLine('error', `Command not found: ${command}`)
                addLine('output', 'Type "help" for available commands')
        }
    }

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            executeCommand(input)
        }
    }

    const copyToClipboard = () => {
        const text = lines.map(l => l.content).join('\n')
        navigator.clipboard.writeText(text)
        setCopied(true)
        toast.success('Terminal output copied!')
        setTimeout(() => setCopied(false), 2000)
    }

    const getLineColor = (type: TerminalLine['type']) => {
        switch (type) {
            case 'input': return 'text-emerald-400'
            case 'output': return 'text-gray-300'
            case 'error': return 'text-red-400'
            case 'system': return 'text-amber-400'
        }
    }

    if (!isOpen) return null

    return (
        <AnimatePresence>
            <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className={`fixed z-50 ${isFullscreen ? 'inset-0' : 'inset-4 md:inset-8'}`}
            >
                {/* Backdrop */}
                <div
                    className="absolute inset-0 bg-black/60 backdrop-blur-sm"
                    onClick={onClose}
                />

                {/* Terminal Window */}
                <motion.div
                    initial={{ scale: 0.95, y: 20 }}
                    animate={{ scale: 1, y: 0 }}
                    exit={{ scale: 0.95, y: 20 }}
                    className={`relative ${isFullscreen ? 'h-full' : 'h-[80vh] max-h-[600px]'} bg-[#1e1e2e] border border-[#313244] rounded-xl overflow-hidden shadow-2xl flex flex-col`}
                    onClick={e => e.stopPropagation()}
                >
                    {/* Title Bar */}
                    <div className="flex items-center justify-between px-4 py-3 bg-[#181825] border-b border-[#313244]">
                        <div className="flex items-center gap-3">
                            <TerminalIcon className="h-4 w-4 text-emerald-400" />
                            <span className="text-sm font-medium text-white">{instanceName}</span>
                            <span className={`px-2 py-0.5 rounded text-xs ${isConnected ? 'bg-emerald-500/20 text-emerald-400' : 'bg-red-500/20 text-red-400'}`}>
                                {isConnected ? 'Connected' : 'Disconnected'}
                            </span>
                        </div>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={copyToClipboard}
                                className="p-1.5 hover:bg-white/10 rounded transition-colors"
                                title="Copy output"
                            >
                                {copied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4 text-gray-400" />}
                            </button>
                            <button
                                onClick={() => setIsFullscreen(!isFullscreen)}
                                className="p-1.5 hover:bg-white/10 rounded transition-colors"
                                title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
                            >
                                {isFullscreen ? <Minimize2 className="h-4 w-4 text-gray-400" /> : <Maximize2 className="h-4 w-4 text-gray-400" />}
                            </button>
                            <button
                                onClick={onClose}
                                className="p-1.5 hover:bg-white/10 rounded transition-colors"
                                title="Close"
                            >
                                <X className="h-4 w-4 text-gray-400" />
                            </button>
                        </div>
                    </div>

                    {/* Terminal Content */}
                    <div
                        ref={terminalRef}
                        className="flex-1 overflow-y-auto p-4 font-mono text-sm"
                        onClick={() => inputRef.current?.focus()}
                    >
                        {lines.map((line, i) => (
                            <div key={i} className={`${getLineColor(line.type)} whitespace-pre-wrap`}>
                                {line.content}
                            </div>
                        ))}

                        {/* Input Line */}
                        <div className="flex items-center text-emerald-400 mt-1">
                            <span className="mr-2">$</span>
                            <input
                                ref={inputRef}
                                type="text"
                                value={input}
                                onChange={e => setInput(e.target.value)}
                                onKeyDown={handleKeyDown}
                                className="flex-1 bg-transparent outline-none text-white caret-emerald-400"
                                placeholder="Type a command..."
                                autoFocus
                            />
                        </div>
                    </div>
                </motion.div>
            </motion.div>
        </AnimatePresence>
    )
}
