import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import {
    ArrowLeft,
    Terminal as TerminalIcon,
    FileText,
    Power,
    Trash2,
    RefreshCw,
    Globe,
    Cpu,
    Clock,
    DollarSign,
    Copy,
    Check,
    Loader2
} from 'lucide-react'
import { api, type Instance } from '@/lib/api'
import { toast } from 'sonner'
import Terminal from '@/components/Terminal'
import LogViewer from '@/components/LogViewer'

export default function InstanceDetail() {
    const { id } = useParams<{ id: string }>()
    const navigate = useNavigate()
    const [instance, setInstance] = useState<Instance | null>(null)
    const [loading, setLoading] = useState(true)
    const [showTerminal, setShowTerminal] = useState(false)
    const [showLogs, setShowLogs] = useState(false)
    const [copied, setCopied] = useState(false)
    const [actionLoading, setActionLoading] = useState<string | null>(null)

    useEffect(() => {
        if (id) {
            fetchInstance()
            const interval = setInterval(fetchInstance, 5000)
            return () => clearInterval(interval)
        }
    }, [id])

    const fetchInstance = async () => {
        if (!id) return
        try {
            const data = await api.getInstance(id)
            setInstance(data)
        } catch (e: any) {
            toast.error('Failed to load instance')
        } finally {
            setLoading(false)
        }
    }

    const handleAction = async (action: 'start' | 'stop' | 'delete') => {
        if (!id) return
        setActionLoading(action)
        try {
            if (action === 'start') {
                await api.startInstance(id)
                toast.success('Instance starting...')
            } else if (action === 'stop') {
                await api.stopInstance(id)
                toast.success('Instance stopping...')
            } else if (action === 'delete') {
                if (!confirm('Are you sure you want to delete this instance?')) {
                    setActionLoading(null)
                    return
                }
                await api.deleteInstance(id)
                toast.success('Instance deleted')
                navigate('/')
                return
            }
            fetchInstance()
        } catch (e: any) {
            toast.error(e.message || `Failed to ${action} instance`)
        } finally {
            setActionLoading(null)
        }
    }

    const copyIP = () => {
        if (instance?.public_ip) {
            navigator.clipboard.writeText(instance.public_ip)
            setCopied(true)
            toast.success('IP copied!')
            setTimeout(() => setCopied(false), 2000)
        }
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'running': return 'bg-emerald-500'
            case 'stopped': return 'bg-gray-500'
            case 'pending': return 'bg-amber-500'
            case 'error': return 'bg-red-500'
            default: return 'bg-gray-500'
        }
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
        )
    }

    if (!instance) {
        return (
            <div className="text-center py-12">
                <p className="text-muted-foreground">Instance not found</p>
                <Link to="/" className="text-emerald-500 hover:underline mt-2 inline-block">
                    Back to Dashboard
                </Link>
            </div>
        )
    }

    return (
        <div className="space-y-8">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Link to="/" className="p-2 hover:bg-muted/50 rounded-lg transition-colors">
                        <ArrowLeft className="h-5 w-5" />
                    </Link>
                    <div>
                        <div className="flex items-center gap-3">
                            <h1 className="text-2xl font-bold">{instance.name}</h1>
                            <span className={`h-2.5 w-2.5 rounded-full ${getStatusColor(instance.status)}`} />
                            <span className="text-sm text-muted-foreground capitalize">{instance.status}</span>
                        </div>
                        <p className="text-sm text-muted-foreground mt-1">
                            {instance.provider} • {instance.region} • {instance.instance_type}
                        </p>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => setShowTerminal(true)}
                        disabled={instance.status !== 'running'}
                        className="flex items-center gap-2 px-4 py-2 rounded-lg bg-muted/50 hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        <TerminalIcon className="h-4 w-4" />
                        Terminal
                    </button>
                    <button
                        onClick={() => setShowLogs(true)}
                        className="flex items-center gap-2 px-4 py-2 rounded-lg bg-muted/50 hover:bg-muted transition-colors"
                    >
                        <FileText className="h-4 w-4" />
                        Logs
                    </button>
                </div>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="p-4 rounded-xl border border-border/40 bg-card/30"
                >
                    <div className="flex items-center gap-2 text-muted-foreground mb-2">
                        <Globe className="h-4 w-4" />
                        <span className="text-sm">Public IP</span>
                    </div>
                    {instance.public_ip ? (
                        <button onClick={copyIP} className="flex items-center gap-2 font-mono text-sm hover:text-emerald-500 transition-colors">
                            {instance.public_ip}
                            {copied ? <Check className="h-3 w-3 text-emerald-500" /> : <Copy className="h-3 w-3" />}
                        </button>
                    ) : (
                        <span className="text-muted-foreground text-sm">Not assigned</span>
                    )}
                </motion.div>

                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.1 }}
                    className="p-4 rounded-xl border border-border/40 bg-card/30"
                >
                    <div className="flex items-center gap-2 text-muted-foreground mb-2">
                        <Cpu className="h-4 w-4" />
                        <span className="text-sm">Instance Type</span>
                    </div>
                    <span className="font-medium">{instance.instance_type}</span>
                </motion.div>

                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.2 }}
                    className="p-4 rounded-xl border border-border/40 bg-card/30"
                >
                    <div className="flex items-center gap-2 text-muted-foreground mb-2">
                        <Clock className="h-4 w-4" />
                        <span className="text-sm">Created</span>
                    </div>
                    <span className="font-medium text-sm">
                        {new Date(instance.created_at).toLocaleDateString()}
                    </span>
                </motion.div>

                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.3 }}
                    className="p-4 rounded-xl border border-border/40 bg-card/30"
                >
                    <div className="flex items-center gap-2 text-muted-foreground mb-2">
                        <DollarSign className="h-4 w-4" />
                        <span className="text-sm">Hourly Rate</span>
                    </div>
                    <span className="font-medium">
                        ${instance.hourly_rate?.toFixed(2) || '0.00'}/hr
                    </span>
                </motion.div>
            </div>

            {/* Actions */}
            <div className="flex items-center gap-4 p-6 rounded-xl border border-border/40 bg-card/30">
                <h3 className="font-medium mr-4">Actions</h3>
                {instance.status === 'running' ? (
                    <button
                        onClick={() => handleAction('stop')}
                        disabled={!!actionLoading}
                        className="flex items-center gap-2 px-4 py-2 rounded-lg bg-amber-500/10 text-amber-500 hover:bg-amber-500/20 transition-colors disabled:opacity-50"
                    >
                        {actionLoading === 'stop' ? <Loader2 className="h-4 w-4 animate-spin" /> : <Power className="h-4 w-4" />}
                        Stop
                    </button>
                ) : (
                    <button
                        onClick={() => handleAction('start')}
                        disabled={!!actionLoading || instance.status === 'pending'}
                        className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500/10 text-emerald-500 hover:bg-emerald-500/20 transition-colors disabled:opacity-50"
                    >
                        {actionLoading === 'start' ? <Loader2 className="h-4 w-4 animate-spin" /> : <Power className="h-4 w-4" />}
                        Start
                    </button>
                )}
                <button
                    onClick={() => fetchInstance()}
                    className="flex items-center gap-2 px-4 py-2 rounded-lg bg-muted/50 hover:bg-muted transition-colors"
                >
                    <RefreshCw className="h-4 w-4" />
                    Refresh
                </button>
                <button
                    onClick={() => handleAction('delete')}
                    disabled={!!actionLoading}
                    className="flex items-center gap-2 px-4 py-2 rounded-lg bg-red-500/10 text-red-500 hover:bg-red-500/20 transition-colors disabled:opacity-50 ml-auto"
                >
                    {actionLoading === 'delete' ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                    Delete
                </button>
            </div>

            {/* Terminal Modal */}
            {showTerminal && (
                <Terminal
                    instanceId={instance.id}
                    instanceName={instance.name}
                    isOpen={showTerminal}
                    onClose={() => setShowTerminal(false)}
                />
            )}

            {/* Log Viewer Modal */}
            {showLogs && (
                <LogViewer
                    instanceId={instance.id}
                    instanceName={instance.name}
                    isOpen={showLogs}
                    onClose={() => setShowLogs(false)}
                />
            )}
        </div>
    )
}
