import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
    Key,
    Cloud,
    User,
    Plus,
    Trash2,
    Copy,
    Check,
    Eye,
    EyeOff,
    AlertCircle
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface APIKey {
    id: string
    name: string
    key_prefix: string
    created_at: string
    last_used_at?: string
}

export default function Settings() {
    const [activeTab, setActiveTab] = useState<'profile' | 'api-keys' | 'credentials'>('profile')
    const [apiKeys, setApiKeys] = useState<APIKey[]>([])
    const [newKeyName, setNewKeyName] = useState('')
    const [createdKey, setCreatedKey] = useState<string | null>(null)
    const [copied, setCopied] = useState(false)

    useEffect(() => {
        fetchAPIKeys()
    }, [])

    const fetchAPIKeys = async () => {
        const res = await fetch('/api/v1/api-keys', {
            headers: { 'X-API-Key': 'cm_demo' }
        })
        if (res.ok) {
            const data = await res.json()
            setApiKeys(data || [])
        }
    }

    const createAPIKey = async () => {
        if (!newKeyName) return

        const res = await fetch('/api/v1/api-keys', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': 'cm_demo'
            },
            body: JSON.stringify({ name: newKeyName })
        })

        if (res.ok) {
            const data = await res.json()
            setCreatedKey(data.key)
            setNewKeyName('')
            fetchAPIKeys()
        }
    }

    const deleteAPIKey = async (id: string) => {
        if (!confirm('Are you sure you want to delete this API key?')) return

        await fetch(`/api/v1/api-keys/${id}`, {
            method: 'DELETE',
            headers: { 'X-API-Key': 'cm_demo' }
        })
        fetchAPIKeys()
    }

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    const tabs = [
        { id: 'profile', label: 'Profile', icon: User },
        { id: 'api-keys', label: 'API Keys', icon: Key },
        { id: 'credentials', label: 'Cloud Credentials', icon: Cloud },
    ] as const

    return (
        <div className="max-w-4xl mx-auto space-y-8">
            <div>
                <h2 className="text-2xl font-bold mb-2">Settings</h2>
                <p className="text-muted-foreground">Manage your account and integrations</p>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 border-b border-border/40 pb-4">
                {tabs.map(tab => (
                    <button
                        key={tab.id}
                        onClick={() => setActiveTab(tab.id)}
                        className={cn(
                            "flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors",
                            activeTab === tab.id
                                ? "bg-emerald-500/10 text-emerald-500"
                                : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                        )}
                    >
                        <tab.icon className="h-4 w-4" />
                        {tab.label}
                    </button>
                ))}
            </div>

            {/* Profile Tab */}
            {activeTab === 'profile' && (
                <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="space-y-6"
                >
                    <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                        <h3 className="text-lg font-semibold mb-6">Profile Information</h3>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium mb-2">Display Name</label>
                                <input
                                    type="text"
                                    defaultValue="Demo User"
                                    className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-2">Email</label>
                                <input
                                    type="email"
                                    defaultValue="demo@container-maker.dev"
                                    className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                                />
                            </div>
                            <button className="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-white rounded-lg font-medium transition-colors">
                                Save Changes
                            </button>
                        </div>
                    </div>
                </motion.div>
            )}

            {/* API Keys Tab */}
            {activeTab === 'api-keys' && (
                <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="space-y-6"
                >
                    {/* Create New Key */}
                    <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                        <h3 className="text-lg font-semibold mb-4">Create API Key</h3>
                        <div className="flex gap-4">
                            <input
                                type="text"
                                value={newKeyName}
                                onChange={(e) => setNewKeyName(e.target.value)}
                                placeholder="Key name (e.g., CI/CD Pipeline)"
                                className="flex-1 px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                            />
                            <button
                                onClick={createAPIKey}
                                disabled={!newKeyName}
                                className="px-4 py-2.5 bg-emerald-500 hover:bg-emerald-600 text-white rounded-lg font-medium transition-colors flex items-center gap-2 disabled:opacity-50"
                            >
                                <Plus className="h-4 w-4" />
                                Create
                            </button>
                        </div>

                        {createdKey && (
                            <div className="mt-4 p-4 rounded-lg bg-amber-500/10 border border-amber-500/20">
                                <div className="flex items-start gap-3">
                                    <AlertCircle className="h-5 w-5 text-amber-500 flex-shrink-0 mt-0.5" />
                                    <div className="flex-1">
                                        <p className="text-sm font-medium text-amber-500 mb-2">
                                            Save this key now! It won't be shown again.
                                        </p>
                                        <div className="flex items-center gap-2">
                                            <code className="flex-1 px-3 py-2 rounded bg-background font-mono text-sm break-all">
                                                {createdKey}
                                            </code>
                                            <button
                                                onClick={() => copyToClipboard(createdKey)}
                                                className="p-2 hover:bg-muted/50 rounded-lg transition-colors"
                                            >
                                                {copied ? <Check className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Existing Keys */}
                    <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                        <h3 className="text-lg font-semibold mb-4">Your API Keys</h3>
                        {apiKeys.length === 0 ? (
                            <p className="text-muted-foreground text-sm">No API keys yet. Create one above.</p>
                        ) : (
                            <div className="space-y-3">
                                {apiKeys.map(key => (
                                    <div key={key.id} className="flex items-center justify-between p-4 rounded-lg bg-muted/20 border border-border/40">
                                        <div>
                                            <p className="font-medium">{key.name}</p>
                                            <p className="text-sm text-muted-foreground font-mono">{key.key_prefix}••••••••</p>
                                        </div>
                                        <button
                                            onClick={() => deleteAPIKey(key.id)}
                                            className="p-2 hover:bg-red-500/10 text-muted-foreground hover:text-red-500 rounded-lg transition-colors"
                                        >
                                            <Trash2 className="h-4 w-4" />
                                        </button>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </motion.div>
            )}

            {/* Cloud Credentials Tab */}
            {activeTab === 'credentials' && (
                <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="space-y-6"
                >
                    <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                        <h3 className="text-lg font-semibold mb-4">Cloud Provider Credentials</h3>
                        <p className="text-muted-foreground text-sm mb-6">
                            Add your cloud provider credentials to deploy instances across multiple platforms.
                        </p>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            {[
                                { name: 'AWS', icon: '🔶', configured: false },
                                { name: 'Google Cloud', icon: '🔵', configured: false },
                                { name: 'Azure', icon: '🔷', configured: false },
                                { name: 'DigitalOcean', icon: '💧', configured: false },
                                { name: 'Hetzner', icon: '🔴', configured: false },
                                { name: 'Vultr', icon: '🟣', configured: false },
                            ].map(provider => (
                                <div
                                    key={provider.name}
                                    className="flex items-center justify-between p-4 rounded-lg border border-border/40 bg-muted/20 hover:bg-muted/30 transition-colors"
                                >
                                    <div className="flex items-center gap-3">
                                        <span className="text-2xl">{provider.icon}</span>
                                        <span className="font-medium">{provider.name}</span>
                                    </div>
                                    <button className="text-sm text-emerald-500 hover:text-emerald-400">
                                        Configure
                                    </button>
                                </div>
                            ))}
                        </div>
                    </div>
                </motion.div>
            )}
        </div>
    )
}
