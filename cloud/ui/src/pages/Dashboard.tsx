import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
    Server,
    Cpu,
    Activity,
    Power,
    Trash2,
    Terminal
} from 'lucide-react'
import { api, type Instance } from '@/lib/api'
import { cn } from '@/lib/utils'

export default function Dashboard() {
    const [instances, setInstances] = useState<Instance[]>([])
    const [loading, setLoading] = useState(true)

    const fetchInstances = async () => {
        try {
            const data = await api.getInstances()
            setInstances(data)
        } catch (e) {
            console.error(e)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchInstances()
        const interval = setInterval(fetchInstances, 5000) // Poll every 5s
        return () => clearInterval(interval)
    }, [])

    const handleStop = async (id: string) => {
        await api.stopInstance(id)
        fetchInstances() // Optimistic update would be better
    }

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure?')) return
        await api.deleteInstance(id)
        fetchInstances()
    }

    if (loading && instances.length === 0) {
        return (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {[1, 2, 3].map(i => (
                    <div key={i} className="h-48 bg-muted/20 animate-pulse rounded-lg border border-border/50" />
                ))}
            </div>
        )
    }

    return (
        <div className="space-y-8">
            {/* Stats Row */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="p-6 rounded-xl border border-border/40 bg-card/30 backdrop-blur-sm">
                    <div className="flex items-center justify-between mb-4">
                        <span className="text-sm font-medium text-muted-foreground">Active Instances</span>
                        <Server className="h-4 w-4 text-emerald-500" />
                    </div>
                    <div className="text-3xl font-bold">{instances.filter(i => i.status === 'running').length}</div>
                </div>
                <div className="p-6 rounded-xl border border-border/40 bg-card/30 backdrop-blur-sm">
                    <div className="flex items-center justify-between mb-4">
                        <span className="text-sm font-medium text-muted-foreground">Monthly Spend</span>
                        <span className="text-xs font-mono bg-muted px-2 py-0.5 rounded text-muted-foreground">ESTIMATED</span>
                    </div>
                    <div className="text-3xl font-bold">$45.20</div>
                </div>
                <div className="p-6 rounded-xl border border-border/40 bg-card/30 backdrop-blur-sm">
                    <div className="flex items-center justify-between mb-4">
                        <span className="text-sm font-medium text-muted-foreground">System Health</span>
                        <Activity className="h-4 w-4 text-emerald-500" />
                    </div>
                    <div className="text-3xl font-bold text-emerald-500 flex items-center gap-2">
                        99.9%
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-lg font-semibold mb-6">Running Instances</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
                    <AnimatePresence>
                        {instances.map((instance) => (
                            <InstanceCard
                                key={instance.id}
                                instance={instance}
                                onStop={() => handleStop(instance.id)}
                                onDelete={() => handleDelete(instance.id)}
                            />
                        ))}
                    </AnimatePresence>
                </div>
            </div>
        </div>
    )
}

function InstanceCard({ instance, onStop, onDelete }: {
    instance: Instance,
    onStop: () => void,
    onDelete: () => void
}) {
    const isRunning = instance.status === 'running'

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95 }}
            className="group relative rounded-xl border border-border/40 bg-card/50 backdrop-blur-sm overflow-hidden hover:border-emerald-500/30 transition-colors"
        >
            {/* Top Bar */}
            <div className="p-5 flex items-start justify-between">
                <div className="flex items-start gap-4">
                    <div className={cn(
                        "h-10 w-10 rounded-lg flex items-center justify-center border",
                        isRunning
                            ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-500"
                            : "bg-muted border-border text-muted-foreground"
                    )}>
                        <Cpu className="h-5 w-5" />
                    </div>
                    <div>
                        <h3 className="font-semibold text-lg leading-none mb-1.5">{instance.name}</h3>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground font-mono">
                            <span className="uppercase">{instance.provider}</span>
                            <span>•</span>
                            <span>{instance.instance_type}</span>
                            <span>•</span>
                            <span>{instance.region}</span>
                        </div>
                    </div>
                </div>

                <div className="relative">
                    <div className={cn(
                        "flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border",
                        isRunning
                            ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-500"
                            : "bg-yellow-500/10 border-yellow-500/20 text-yellow-500"
                    )}>
                        <div className={cn("h-1.5 w-1.5 rounded-full", isRunning ? "bg-emerald-500 animate-pulse" : "bg-yellow-500")} />
                        <span className="capitalize">{instance.status}</span>
                    </div>
                </div>
            </div>

            {/* Connection Info */}
            <div className="px-5 py-3 bg-muted/20 border-y border-border/40 flex items-center gap-3">
                <div className="flex-1 flex items-center gap-2 font-mono text-xs text-muted-foreground">
                    <Terminal className="h-3.5 w-3.5" />
                    {instance.public_ip || 'Provisioning IP...'}
                </div>
                <button
                    className="text-xs font-medium text-emerald-500 hover:text-emerald-400 transition-colors disabled:opacity-50"
                    onClick={() => navigator.clipboard.writeText(`ssh root@${instance.public_ip}`)}
                    disabled={!instance.public_ip}
                >
                    Copy SSH
                </button>
            </div>

            {/* Actions */}
            <div className="p-4 flex items-center justify-end gap-2">
                <button
                    onClick={onStop}
                    className="p-2 rounded-md hover:bg-muted/50 text-muted-foreground hover:text-yellow-500 transition-colors"
                    title="Stop Instance"
                >
                    <Power className="h-4 w-4" />
                </button>
                <button
                    onClick={onDelete}
                    className="p-2 rounded-md hover:bg-red-500/10 text-muted-foreground hover:text-red-500 transition-colors"
                    title="Terminate Instance"
                >
                    <Trash2 className="h-4 w-4" />
                </button>
                <button className="ml-2 px-4 py-2 bg-foreground text-background rounded-md text-sm font-medium hover:bg-foreground/90 transition-colors shadow-lg shadow-emerald-500/5">
                    Connect
                </button>
            </div>
        </motion.div>
    )
}
