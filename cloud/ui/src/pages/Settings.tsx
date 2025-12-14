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
    AlertCircle,
    Loader2,
    CheckCircle,
    XCircle,
    Settings2
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { api, type APIKey, type CloudCredential, type User as UserType } from '@/lib/api'
import { toast } from 'sonner'
import CredentialModal, { PROVIDER_CONFIGS } from '@/components/CredentialModal'
import AdminTab from '@/components/AdminTab'

export default function Settings() {
    const [activeTab, setActiveTab] = useState<'profile' | 'api-keys' | 'credentials' | 'admin'>('profile')

    // Profile state
    const [user, setUser] = useState<Partial<UserType>>({ name: '', email: '' })
    const [profileLoading, setProfileLoading] = useState(false)
    const [profileSaving, setProfileSaving] = useState(false)

    // API Keys state
    const [apiKeys, setApiKeys] = useState<APIKey[]>([])
    const [newKeyName, setNewKeyName] = useState('')
    const [createdKey, setCreatedKey] = useState<string | null>(null)
    const [keyLoading, setKeyLoading] = useState(false)
    const [copied, setCopied] = useState(false)

    // Credentials state
    const [credentials, setCredentials] = useState<CloudCredential[]>([])
    const [credLoading, setCredLoading] = useState(false)
    const [modalProvider, setModalProvider] = useState<string | null>(null)

    // Load data on mount
    useEffect(() => {
        loadProfile()
        loadAPIKeys()
        loadCredentials()
    }, [])

    const loadProfile = async () => {
        setProfileLoading(true)
        try {
            const data = await api.getCurrentUser()
            setUser(data)
        } catch (e) {
            // Use defaults if not logged in
        } finally {
            setProfileLoading(false)
        }
    }

    const loadAPIKeys = async () => {
        try {
            const data = await api.getAPIKeys()
            setApiKeys(data || [])
        } catch (e) {
            console.error('Failed to load API keys:', e)
        }
    }

    const loadCredentials = async () => {
        setCredLoading(true)
        try {
            const data = await api.getCredentials()
            setCredentials(data || [])
        } catch (e) {
            console.error('Failed to load credentials:', e)
        } finally {
            setCredLoading(false)
        }
    }

    const saveProfile = async () => {
        setProfileSaving(true)
        try {
            await api.updateUser(user)
            toast.success('Profile updated successfully!')
        } catch (e: any) {
            toast.error(e.message || 'Failed to update profile')
        } finally {
            setProfileSaving(false)
        }
    }

    const createAPIKey = async () => {
        if (!newKeyName.trim()) {
            toast.error('Please enter a name for the API key')
            return
        }

        setKeyLoading(true)
        try {
            const data = await api.createAPIKey(newKeyName)
            setCreatedKey(data.key)
            setNewKeyName('')
            loadAPIKeys()
            toast.success('API key created!')
        } catch (e: any) {
            toast.error(e.message || 'Failed to create API key')
        } finally {
            setKeyLoading(false)
        }
    }

    const deleteAPIKey = async (id: string) => {
        if (!confirm('Are you sure you want to delete this API key?')) return

        try {
            await api.deleteAPIKey(id)
            toast.success('API key deleted')
            loadAPIKeys()
        } catch (e: any) {
            toast.error(e.message || 'Failed to delete API key')
        }
    }

    const deleteCredential = async (id: string) => {
        if (!confirm('Are you sure you want to delete this credential?')) return

        try {
            await api.deleteCredential(id)
            toast.success('Credential deleted')
            loadCredentials()
        } catch (e: any) {
            toast.error(e.message || 'Failed to delete credential')
        }
    }

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        setCopied(true)
        toast.success('Copied to clipboard!')
        setTimeout(() => setCopied(false), 2000)
    }

    const getCredentialForProvider = (providerName: string) => {
        return credentials.find(c => c.provider === providerName)
    }

    const tabs = [
        { id: 'profile', label: 'Profile', icon: User },
        { id: 'api-keys', label: 'API Keys', icon: Key },
        { id: 'credentials', label: 'Cloud Credentials', icon: Cloud },
        { id: 'admin', label: 'Admin', icon: Settings2 },
    ] as const

    const providerList = [
        { name: 'aws', display: 'Amazon Web Services', icon: '🔶' },
        { name: 'gcp', display: 'Google Cloud', icon: '🔵' },
        { name: 'azure', display: 'Microsoft Azure', icon: '🔷' },
        { name: 'digitalocean', display: 'DigitalOcean', icon: '💧' },
        { name: 'hetzner', display: 'Hetzner Cloud', icon: '🔴' },
        { name: 'vultr', display: 'Vultr', icon: '🟣' },
        { name: 'linode', display: 'Linode (Akamai)', icon: '🟢' },
        { name: 'oci', display: 'Oracle Cloud', icon: '🔵' },
        { name: 'lambdalabs', display: 'Lambda Labs', icon: '🧪' },
        { name: 'runpod', display: 'RunPod', icon: '🚀' },
        { name: 'vast', display: 'Vast.ai', icon: '🌊' },
    ]

    return (
        <div className="max-w-4xl mx-auto space-y-8">
            <div>
                <h2 className="text-2xl font-bold mb-2">Settings</h2>
                <p className="text-muted-foreground">Manage your account and integrations</p>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 border-b border-border/40 pb-4 overflow-x-auto">
                {tabs.map(tab => (
                    <button
                        key={tab.id}
                        onClick={() => setActiveTab(tab.id)}
                        className={cn(
                            "flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors whitespace-nowrap",
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
                <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
                    <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                        <h3 className="text-lg font-semibold mb-6">Profile Information</h3>
                        {profileLoading ? (
                            <div className="flex items-center justify-center py-8">
                                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium mb-2">Display Name</label>
                                    <input
                                        type="text"
                                        value={user.name || ''}
                                        onChange={e => setUser(prev => ({ ...prev, name: e.target.value }))}
                                        className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium mb-2">Email</label>
                                    <input
                                        type="email"
                                        value={user.email || ''}
                                        onChange={e => setUser(prev => ({ ...prev, email: e.target.value }))}
                                        className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                                    />
                                </div>
                                <button
                                    onClick={saveProfile}
                                    disabled={profileSaving}
                                    className="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-white rounded-lg font-medium transition-colors flex items-center gap-2 disabled:opacity-50"
                                >
                                    {profileSaving && <Loader2 className="h-4 w-4 animate-spin" />}
                                    Save Changes
                                </button>
                            </div>
                        )}
                    </div>
                </motion.div>
            )}

            {/* API Keys Tab */}
            {activeTab === 'api-keys' && (
                <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
                    {/* Create New Key */}
                    <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                        <h3 className="text-lg font-semibold mb-4">Create API Key</h3>
                        <div className="flex gap-4">
                            <input
                                type="text"
                                value={newKeyName}
                                onChange={e => setNewKeyName(e.target.value)}
                                placeholder="Key name (e.g., CI/CD Pipeline)"
                                className="flex-1 px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                                onKeyDown={e => e.key === 'Enter' && createAPIKey()}
                            />
                            <button
                                onClick={createAPIKey}
                                disabled={!newKeyName.trim() || keyLoading}
                                className="px-4 py-2.5 bg-emerald-500 hover:bg-emerald-600 text-white rounded-lg font-medium transition-colors flex items-center gap-2 disabled:opacity-50"
                            >
                                {keyLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
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
                <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
                    <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                        <h3 className="text-lg font-semibold mb-4">Cloud Provider Credentials</h3>
                        <p className="text-muted-foreground text-sm mb-6">
                            Add your cloud provider credentials to deploy instances across multiple platforms.
                        </p>

                        {credLoading ? (
                            <div className="flex items-center justify-center py-8">
                                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                            </div>
                        ) : (
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                {providerList.map(provider => {
                                    const cred = getCredentialForProvider(provider.name)
                                    return (
                                        <div
                                            key={provider.name}
                                            className="flex items-center justify-between p-4 rounded-lg border border-border/40 bg-muted/20 hover:bg-muted/30 transition-colors"
                                        >
                                            <div className="flex items-center gap-3">
                                                <span className="text-2xl">{provider.icon}</span>
                                                <div>
                                                    <span className="font-medium">{provider.display}</span>
                                                    {cred && (
                                                        <div className="flex items-center gap-1 mt-0.5">
                                                            {cred.is_valid ? (
                                                                <CheckCircle className="h-3 w-3 text-emerald-500" />
                                                            ) : (
                                                                <XCircle className="h-3 w-3 text-red-500" />
                                                            )}
                                                            <span className="text-xs text-muted-foreground">{cred.name}</span>
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                            <div className="flex items-center gap-2">
                                                {cred && (
                                                    <button
                                                        onClick={() => deleteCredential(cred.id)}
                                                        className="p-1.5 hover:bg-red-500/10 text-muted-foreground hover:text-red-500 rounded transition-colors"
                                                    >
                                                        <Trash2 className="h-4 w-4" />
                                                    </button>
                                                )}
                                                <button
                                                    onClick={() => setModalProvider(provider.name)}
                                                    className="text-sm text-emerald-500 hover:text-emerald-400 font-medium"
                                                >
                                                    {cred ? 'Update' : 'Configure'}
                                                </button>
                                            </div>
                                        </div>
                                    )
                                })}
                            </div>
                        )}
                    </div>

                    {/* Docker - Free Local */}
                    <div className="p-6 rounded-xl border border-emerald-500/40 bg-emerald-500/5">
                        <div className="flex items-center gap-3 mb-2">
                            <span className="text-2xl">🐳</span>
                            <h3 className="text-lg font-semibold">Docker (Local)</h3>
                            <span className="px-2 py-0.5 bg-emerald-500/20 text-emerald-500 text-xs font-medium rounded">FREE</span>
                        </div>
                        <p className="text-muted-foreground text-sm">
                            Run development environments locally using Docker. No configuration needed - works out of the box!
                        </p>
                    </div>
                </motion.div>
            )}

            {/* Admin Tab */}
            {activeTab === 'admin' && (
                <AdminTab />
            )}

            {/* Credential Modal */}
            {modalProvider && (
                <CredentialModal
                    provider={modalProvider}
                    isOpen={!!modalProvider}
                    onClose={() => setModalProvider(null)}
                    onSuccess={loadCredentials}
                    existingCredential={getCredentialForProvider(modalProvider) ? {
                        id: getCredentialForProvider(modalProvider)!.id,
                        name: getCredentialForProvider(modalProvider)!.name
                    } : undefined}
                />
            )}
        </div>
    )
}
