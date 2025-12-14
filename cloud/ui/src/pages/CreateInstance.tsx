import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { Check, Cpu, Globe, Rocket } from 'lucide-react'
import { api, Provider } from '@/lib/api'
import { cn } from '@/lib/utils'

const instanceTypes = [
    { id: 'cpu-small', name: 'CPU Small', vcpu: 2, ram: '4GB', price: '$0.02/hr', type: 'cpu' },
    { id: 'cpu-medium', name: 'CPU Medium', vcpu: 4, ram: '8GB', price: '$0.04/hr', type: 'cpu' },
    { id: 'gpu-t4', name: 'NVIDIA T4', vcpu: 4, ram: '16GB', gpu: '1x T4', price: '$0.50/hr', type: 'gpu' },
    { id: 'gpu-a10', name: 'NVIDIA A10', vcpu: 8, ram: '32GB', gpu: '1x A10', price: '$1.50/hr', type: 'gpu' },
]

export default function CreateInstance() {
    const navigate = useNavigate()
    const [providers, setProviders] = useState<Provider[]>([])
    const [selectedProvider, setSelectedProvider] = useState('aws')
    const [selectedType, setSelectedType] = useState('cpu-small')
    const [name, setName] = useState('')
    const [isSubmitting, setIsSubmitting] = useState(false)

    useEffect(() => {
        api.getProviders().then(setProviders)
    }, [])

    const handleSubmit = async () => {
        setIsSubmitting(true)
        try {
            await api.createInstance({
                name: name || 'Untitled Instance',
                provider: selectedProvider,
                instance_type: selectedType,
                region: 'us-east-1'
            })
            navigate('/')
        } catch (e) {
            console.error(e)
        } finally {
            setIsSubmitting(false)
        }
    }

    return (
        <div className="max-w-4xl mx-auto space-y-10 pb-20">
            <div>
                <h2 className="text-2xl font-bold mb-2">Create New Instance</h2>
                <p className="text-muted-foreground">Deploy a new development environment in seconds.</p>
            </div>

            {/* Step 1: Provider */}
            <section>
                <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-4">1. Select Cloud Provider</h3>
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
                    {providers.map((p) => (
                        <button
                            key={p.name}
                            onClick={() => setSelectedProvider(p.name)}
                            className={cn(
                                "flex flex-col items-center justify-center gap-3 p-4 rounded-xl border transition-all duration-200 h-28",
                                selectedProvider === p.name
                                    ? "bg-emerald-500/10 border-emerald-500 ring-1 ring-emerald-500"
                                    : "bg-card/50 border-border/40 hover:border-foreground/20 hover:bg-muted/50"
                            )}
                        >
                            <div className="h-8 w-8 rounded bg-foreground/10 flex items-center justify-center text-xs font-bold uppercase">
                                {p.name.substring(0, 2)}
                            </div>
                            <span className="text-xs font-medium text-center">{p.display_name}</span>
                        </button>
                    ))}
                </div>
            </section>

            {/* Step 2: Instance Type */}
            <section>
                <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-4">2. Select Instance Type</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {instanceTypes.map((t) => (
                        <button
                            key={t.id}
                            onClick={() => setSelectedType(t.id)}
                            className={cn(
                                "relative flex items-center gap-4 p-4 rounded-xl border text-left transition-all",
                                selectedType === t.id
                                    ? "bg-emerald-500/5 border-emerald-500"
                                    : "bg-card/50 border-border/40 hover:border-foreground/20"
                            )}
                        >
                            <div className={cn(
                                "h-12 w-12 rounded-lg flex items-center justify-center border",
                                t.type === 'gpu' ? "bg-purple-500/10 border-purple-500/20 text-purple-500" : "bg-blue-500/10 border-blue-500/20 text-blue-500"
                            )}>
                                <Cpu className="h-6 w-6" />
                            </div>
                            <div className="flex-1">
                                <div className="flex items-center justify-between mb-1">
                                    <span className="font-semibold">{t.name}</span>
                                    <span className="text-sm font-mono">{t.price}</span>
                                </div>
                                <div className="text-xs text-muted-foreground">
                                    {t.vcpu} vCPU • {t.ram} RAM {t.gpu && `• ${t.gpu}`}
                                </div>
                            </div>
                            {selectedType === t.id && (
                                <div className="absolute right-4 top-4 text-emerald-500">
                                    <Check className="h-4 w-4" />
                                </div>
                            )}
                        </button>
                    ))}
                </div>
            </section>

            {/* Step 3: Configuration */}
            <section>
                <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-4">3. Configure</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6 p-6 rounded-xl border border-border/40 bg-card/30">
                    <div>
                        <label className="block text-sm font-medium mb-2">Instance Name</label>
                        <input
                            type="text"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            placeholder="e.g., ai-training-cluster"
                            className="w-full px-4 py-2 rounded-md bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all placeholder:text-muted-foreground/50"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-2">Region</label>
                        <div className="flex items-center gap-3 px-4 py-2 rounded-md bg-muted/50 border border-border text-muted-foreground cursor-not-allowed">
                            <Globe className="h-4 w-4" />
                            <span>US East (N. Virginia)</span>
                        </div>
                    </div>
                </div>
            </section>

            <div className="flex justify-end pt-6 border-t border-border/40">
                <button
                    onClick={handleSubmit}
                    disabled={isSubmitting}
                    className="bg-emerald-500 text-white hover:bg-emerald-600 px-8 py-3 rounded-lg font-bold flex items-center gap-2 transition-all shadow-lg shadow-emerald-500/20 hover:shadow-emerald-500/40 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    {isSubmitting ? (
                        'Deploying...'
                    ) : (
                        <>
                            <Rocket className="h-4 w-4" />
                            Deploy Instance
                        </>
                    )}
                </button>
            </div>
        </div>
    )
}
