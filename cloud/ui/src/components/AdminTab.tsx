import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { AlertCircle, Loader2, CheckCircle, Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'

interface AdminConfig {
    github_client_id: string
    github_configured: boolean
    google_client_id: string
    google_configured: boolean
    stripe_publishable_key: string
    stripe_configured: boolean
}

export default function AdminTab() {
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [config, setConfig] = useState<AdminConfig | null>(null)

    // Form state
    const [githubClientId, setGithubClientId] = useState('')
    const [githubClientSecret, setGithubClientSecret] = useState('')
    const [googleClientId, setGoogleClientId] = useState('')
    const [googleClientSecret, setGoogleClientSecret] = useState('')
    const [stripePublishable, setStripePublishable] = useState('')
    const [stripeSecret, setStripeSecret] = useState('')
    const [stripeWebhook, setStripeWebhook] = useState('')

    // Show/hide secrets
    const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({})

    useEffect(() => {
        loadConfig()
    }, [])

    const loadConfig = async () => {
        try {
            const token = localStorage.getItem('access_token')
            const headers: Record<string, string> = token
                ? { Authorization: `Bearer ${token}` }
                : { 'X-API-Key': 'cm_demo' }

            const res = await fetch('/api/v1/admin/config', { headers })
            if (res.ok) {
                const data = await res.json()
                setConfig(data)
                // Pre-fill editable fields
                setGithubClientId(data.github_client_id || '')
                setGoogleClientId(data.google_client_id || '')
                setStripePublishable(data.stripe_publishable_key || '')
            }
        } catch (e) {
            console.error('Failed to load config:', e)
        } finally {
            setLoading(false)
        }
    }

    const saveConfig = async () => {
        setSaving(true)
        try {
            const token = localStorage.getItem('access_token')
            const headers: Record<string, string> = {
                'Content-Type': 'application/json',
                ...(token ? { Authorization: `Bearer ${token}` } : { 'X-API-Key': 'cm_demo' })
            }

            const res = await fetch('/api/v1/admin/config', {
                method: 'PUT',
                headers,
                body: JSON.stringify({
                    github_client_id: githubClientId,
                    github_client_secret: githubClientSecret,
                    google_client_id: googleClientId,
                    google_client_secret: googleClientSecret,
                    stripe_publishable_key: stripePublishable,
                    stripe_secret_key: stripeSecret,
                    stripe_webhook_secret: stripeWebhook,
                })
            })

            if (res.ok) {
                toast.success('Admin settings saved! Config updated in database.')
                // Clear secrets after save (they shouldn't be kept in state)
                setGithubClientSecret('')
                setGoogleClientSecret('')
                setStripeSecret('')
                setStripeWebhook('')
                // Reload to get updated status
                loadConfig()
            } else {
                const data = await res.json()
                toast.error(data.message || 'Failed to save settings')
            }
        } catch (e: any) {
            toast.error(e.message || 'Failed to save settings')
        } finally {
            setSaving(false)
        }
    }

    const toggleSecret = (key: string) => {
        setShowSecrets(prev => ({ ...prev, [key]: !prev[key] }))
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
        )
    }

    return (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
            <div className="p-6 rounded-xl border border-amber-500/40 bg-amber-500/5">
                <div className="flex items-center gap-2 mb-4">
                    <AlertCircle className="h-5 w-5 text-amber-500" />
                    <h3 className="text-lg font-semibold">Admin Configuration</h3>
                </div>
                <p className="text-muted-foreground text-sm mb-6">
                    These settings configure integrations for all users. Changes are saved to the database and persist across server restarts.
                </p>

                {/* OAuth Configuration */}
                <div className="space-y-4 mb-8">
                    <div className="flex items-center gap-2">
                        <h4 className="font-medium">OAuth Providers</h4>
                        {config?.github_configured && (
                            <span className="flex items-center gap-1 text-xs text-emerald-500">
                                <CheckCircle className="h-3 w-3" /> GitHub configured
                            </span>
                        )}
                        {config?.google_configured && (
                            <span className="flex items-center gap-1 text-xs text-emerald-500">
                                <CheckCircle className="h-3 w-3" /> Google configured
                            </span>
                        )}
                    </div>

                    <div className="space-y-3">
                        <div>
                            <label className="block text-sm font-medium mb-1">GitHub Client ID</label>
                            <input
                                type="text"
                                value={githubClientId}
                                onChange={e => setGithubClientId(e.target.value)}
                                placeholder="Ov23lixxxxxxxxx"
                                className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                            />
                        </div>
                        <div className="relative">
                            <label className="block text-sm font-medium mb-1">GitHub Client Secret</label>
                            <input
                                type={showSecrets.github ? 'text' : 'password'}
                                value={githubClientSecret}
                                onChange={e => setGithubClientSecret(e.target.value)}
                                placeholder={config?.github_configured ? '••••••••••••••• (configured)' : 'Enter secret'}
                                className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 pr-10"
                            />
                            <button
                                type="button"
                                onClick={() => toggleSecret('github')}
                                className="absolute right-3 top-8 text-muted-foreground hover:text-foreground"
                            >
                                {showSecrets.github ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                            </button>
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1">Google Client ID</label>
                            <input
                                type="text"
                                value={googleClientId}
                                onChange={e => setGoogleClientId(e.target.value)}
                                placeholder="xxxxx.apps.googleusercontent.com"
                                className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                            />
                        </div>
                        <div className="relative">
                            <label className="block text-sm font-medium mb-1">Google Client Secret</label>
                            <input
                                type={showSecrets.google ? 'text' : 'password'}
                                value={googleClientSecret}
                                onChange={e => setGoogleClientSecret(e.target.value)}
                                placeholder={config?.google_configured ? '••••••••••••••• (configured)' : 'Enter secret'}
                                className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 pr-10"
                            />
                            <button
                                type="button"
                                onClick={() => toggleSecret('google')}
                                className="absolute right-3 top-8 text-muted-foreground hover:text-foreground"
                            >
                                {showSecrets.google ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                            </button>
                        </div>
                    </div>
                </div>

                {/* Stripe Configuration */}
                <div className="space-y-4 mb-6">
                    <div className="flex items-center gap-2">
                        <h4 className="font-medium">Stripe Billing</h4>
                        {config?.stripe_configured && (
                            <span className="flex items-center gap-1 text-xs text-emerald-500">
                                <CheckCircle className="h-3 w-3" /> Configured
                            </span>
                        )}
                    </div>

                    <div className="space-y-3">
                        <div>
                            <label className="block text-sm font-medium mb-1">Publishable Key</label>
                            <input
                                type="text"
                                value={stripePublishable}
                                onChange={e => setStripePublishable(e.target.value)}
                                placeholder="pk_live_xxxxx"
                                className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                            />
                        </div>
                        <div className="relative">
                            <label className="block text-sm font-medium mb-1">Secret Key</label>
                            <input
                                type={showSecrets.stripe ? 'text' : 'password'}
                                value={stripeSecret}
                                onChange={e => setStripeSecret(e.target.value)}
                                placeholder={config?.stripe_configured ? '••••••••••••••• (configured)' : 'sk_live_xxxxx'}
                                className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 pr-10"
                            />
                            <button
                                type="button"
                                onClick={() => toggleSecret('stripe')}
                                className="absolute right-3 top-8 text-muted-foreground hover:text-foreground"
                            >
                                {showSecrets.stripe ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                            </button>
                        </div>
                        <div className="relative">
                            <label className="block text-sm font-medium mb-1">Webhook Secret</label>
                            <input
                                type={showSecrets.webhook ? 'text' : 'password'}
                                value={stripeWebhook}
                                onChange={e => setStripeWebhook(e.target.value)}
                                placeholder="whsec_xxxxx"
                                className="w-full px-4 py-2.5 rounded-lg bg-background border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 pr-10"
                            />
                            <button
                                type="button"
                                onClick={() => toggleSecret('webhook')}
                                className="absolute right-3 top-8 text-muted-foreground hover:text-foreground"
                            >
                                {showSecrets.webhook ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                            </button>
                        </div>
                    </div>
                </div>

                <button
                    onClick={saveConfig}
                    disabled={saving}
                    className="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-white rounded-lg font-medium transition-colors flex items-center gap-2 disabled:opacity-50"
                >
                    {saving && <Loader2 className="h-4 w-4 animate-spin" />}
                    Save Admin Settings
                </button>
            </div>
        </motion.div>
    )
}
