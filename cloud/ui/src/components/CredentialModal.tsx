import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, Loader2, CheckCircle, AlertCircle, Eye, EyeOff } from 'lucide-react'
import { api } from '@/lib/api'
import { toast } from 'sonner'

interface CredentialField {
    key: string
    label: string
    type: 'text' | 'password' | 'textarea'
    placeholder?: string
    required?: boolean
}

interface ProviderConfig {
    name: string
    displayName: string
    icon: string
    fields: CredentialField[]
    helpUrl?: string
}

const PROVIDER_CONFIGS: Record<string, ProviderConfig> = {
    aws: {
        name: 'aws',
        displayName: 'Amazon Web Services',
        icon: '🔶',
        helpUrl: 'https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html',
        fields: [
            { key: 'access_key_id', label: 'Access Key ID', type: 'text', placeholder: 'AKIAIOSFODNN7EXAMPLE', required: true },
            { key: 'secret_access_key', label: 'Secret Access Key', type: 'password', placeholder: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCY...', required: true },
            { key: 'region', label: 'Default Region', type: 'text', placeholder: 'us-east-1' }
        ]
    },
    gcp: {
        name: 'gcp',
        displayName: 'Google Cloud Platform',
        icon: '🔵',
        helpUrl: 'https://cloud.google.com/iam/docs/creating-managing-service-account-keys',
        fields: [
            { key: 'project_id', label: 'Project ID', type: 'text', placeholder: 'my-project-123456', required: true },
            { key: 'service_account_json', label: 'Service Account JSON', type: 'textarea', placeholder: '{"type": "service_account", ...}', required: true }
        ]
    },
    azure: {
        name: 'azure',
        displayName: 'Microsoft Azure',
        icon: '🔷',
        helpUrl: 'https://docs.microsoft.com/en-us/azure/active-directory/develop/howto-create-service-principal-portal',
        fields: [
            { key: 'tenant_id', label: 'Tenant ID', type: 'text', required: true },
            { key: 'client_id', label: 'Client ID', type: 'text', required: true },
            { key: 'client_secret', label: 'Client Secret', type: 'password', required: true },
            { key: 'subscription_id', label: 'Subscription ID', type: 'text', required: true }
        ]
    },
    digitalocean: {
        name: 'digitalocean',
        displayName: 'DigitalOcean',
        icon: '💧',
        helpUrl: 'https://docs.digitalocean.com/reference/api/create-personal-access-token/',
        fields: [
            { key: 'api_token', label: 'API Token', type: 'password', required: true }
        ]
    },
    hetzner: {
        name: 'hetzner',
        displayName: 'Hetzner Cloud',
        icon: '🔴',
        helpUrl: 'https://docs.hetzner.cloud/#getting-started',
        fields: [
            { key: 'api_token', label: 'API Token', type: 'password', required: true }
        ]
    },
    vultr: {
        name: 'vultr',
        displayName: 'Vultr',
        icon: '🟣',
        helpUrl: 'https://my.vultr.com/settings/#settingsapi',
        fields: [
            { key: 'api_key', label: 'API Key', type: 'password', required: true }
        ]
    },
    linode: {
        name: 'linode',
        displayName: 'Linode (Akamai)',
        icon: '🟢',
        helpUrl: 'https://www.linode.com/docs/products/tools/api/get-started/',
        fields: [
            { key: 'api_token', label: 'Personal Access Token', type: 'password', required: true }
        ]
    },
    oci: {
        name: 'oci',
        displayName: 'Oracle Cloud',
        icon: '🔵',
        helpUrl: 'https://docs.oracle.com/en-us/iaas/Content/API/Concepts/apisigningkey.htm',
        fields: [
            { key: 'tenancy_ocid', label: 'Tenancy OCID', type: 'text', required: true },
            { key: 'user_ocid', label: 'User OCID', type: 'text', required: true },
            { key: 'fingerprint', label: 'Key Fingerprint', type: 'text', required: true },
            { key: 'private_key', label: 'Private Key (PEM)', type: 'textarea', required: true }
        ]
    },
    lambdalabs: {
        name: 'lambdalabs',
        displayName: 'Lambda Labs',
        icon: '🧪',
        helpUrl: 'https://cloud.lambdalabs.com/api-keys',
        fields: [
            { key: 'api_key', label: 'API Key', type: 'password', required: true }
        ]
    },
    runpod: {
        name: 'runpod',
        displayName: 'RunPod',
        icon: '🚀',
        helpUrl: 'https://www.runpod.io/console/user/settings',
        fields: [
            { key: 'api_key', label: 'API Key', type: 'password', required: true }
        ]
    },
    vast: {
        name: 'vast',
        displayName: 'Vast.ai',
        icon: '🌊',
        helpUrl: 'https://vast.ai/console/account/',
        fields: [
            { key: 'api_key', label: 'API Key', type: 'password', required: true }
        ]
    }
}

interface Props {
    provider: string
    isOpen: boolean
    onClose: () => void
    onSuccess: () => void
    existingCredential?: { id: string; name: string }
}

export default function CredentialModal({ provider, isOpen, onClose, onSuccess, existingCredential }: Props) {
    const config = PROVIDER_CONFIGS[provider]
    const [formData, setFormData] = useState<Record<string, string>>({})
    const [credentialName, setCredentialName] = useState(existingCredential?.name || '')
    const [loading, setLoading] = useState(false)
    const [testing, setTesting] = useState(false)
    const [testResult, setTestResult] = useState<'success' | 'error' | null>(null)
    const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({})

    if (!config) return null

    const handleChange = (key: string, value: string) => {
        setFormData(prev => ({ ...prev, [key]: value }))
        setTestResult(null)
    }

    const handleTest = async () => {
        setTesting(true)
        setTestResult(null)

        try {
            // Create a temporary credential to test
            const tempCred = await api.addCredential({
                provider: config.name,
                name: `_test_${Date.now()}`,
                data: formData
            })

            // Verify it
            const result = await api.verifyCredential(tempCred.id)

            // Delete the temp credential
            await api.deleteCredential(tempCred.id)

            setTestResult(result.verified ? 'success' : 'error')
            if (result.verified) {
                toast.success('Credentials verified successfully!')
            } else {
                toast.error('Credential verification failed')
            }
        } catch (e: any) {
            setTestResult('error')
            toast.error(e.message || 'Failed to test credentials')
        } finally {
            setTesting(false)
        }
    }

    const handleSave = async () => {
        if (!credentialName.trim()) {
            toast.error('Please enter a name for this credential')
            return
        }

        const missingFields = config.fields
            .filter(f => f.required && !formData[f.key])
            .map(f => f.label)

        if (missingFields.length > 0) {
            toast.error(`Missing required fields: ${missingFields.join(', ')}`)
            return
        }

        setLoading(true)
        try {
            await api.addCredential({
                provider: config.name,
                name: credentialName,
                data: formData
            })

            toast.success(`${config.displayName} credentials saved!`)
            onSuccess()
            onClose()
        } catch (e: any) {
            toast.error(e.message || 'Failed to save credentials')
        } finally {
            setLoading(false)
        }
    }

    return (
        <AnimatePresence>
            {isOpen && (
                <>
                    {/* Backdrop */}
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50"
                        onClick={onClose}
                    />

                    {/* Modal */}
                    <motion.div
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.95 }}
                        className="fixed inset-0 z-50 flex items-center justify-center p-4"
                        onClick={e => e.stopPropagation()}
                    >
                        <div className="bg-background border border-border rounded-xl shadow-2xl max-w-lg w-full max-h-[90vh] overflow-hidden">
                            {/* Header */}
                            <div className="flex items-center justify-between p-6 border-b border-border">
                                <div className="flex items-center gap-3">
                                    <span className="text-2xl">{config.icon}</span>
                                    <div>
                                        <h2 className="text-lg font-semibold">Configure {config.displayName}</h2>
                                        <p className="text-sm text-muted-foreground">
                                            {existingCredential ? 'Update credentials' : 'Add your credentials'}
                                        </p>
                                    </div>
                                </div>
                                <button onClick={onClose} className="p-2 hover:bg-muted rounded-lg">
                                    <X className="h-5 w-5" />
                                </button>
                            </div>

                            {/* Body */}
                            <div className="p-6 space-y-4 overflow-y-auto max-h-[60vh]">
                                {/* Credential Name */}
                                <div>
                                    <label className="block text-sm font-medium mb-2">Credential Name</label>
                                    <input
                                        type="text"
                                        value={credentialName}
                                        onChange={e => setCredentialName(e.target.value)}
                                        placeholder="My AWS Account"
                                        className="w-full px-4 py-2.5 rounded-lg bg-muted/50 border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                                    />
                                </div>

                                {/* Fields */}
                                {config.fields.map(field => (
                                    <div key={field.key}>
                                        <label className="block text-sm font-medium mb-2">
                                            {field.label}
                                            {field.required && <span className="text-red-500 ml-1">*</span>}
                                        </label>
                                        <div className="relative">
                                            {field.type === 'textarea' ? (
                                                <textarea
                                                    value={formData[field.key] || ''}
                                                    onChange={e => handleChange(field.key, e.target.value)}
                                                    placeholder={field.placeholder}
                                                    rows={4}
                                                    className="w-full px-4 py-2.5 rounded-lg bg-muted/50 border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 font-mono text-sm"
                                                />
                                            ) : (
                                                <>
                                                    <input
                                                        type={field.type === 'password' && !showSecrets[field.key] ? 'password' : 'text'}
                                                        value={formData[field.key] || ''}
                                                        onChange={e => handleChange(field.key, e.target.value)}
                                                        placeholder={field.placeholder}
                                                        className="w-full px-4 py-2.5 rounded-lg bg-muted/50 border border-border focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 pr-10"
                                                    />
                                                    {field.type === 'password' && (
                                                        <button
                                                            type="button"
                                                            onClick={() => setShowSecrets(prev => ({ ...prev, [field.key]: !prev[field.key] }))}
                                                            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                                                        >
                                                            {showSecrets[field.key] ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                                                        </button>
                                                    )}
                                                </>
                                            )}
                                        </div>
                                    </div>
                                ))}

                                {/* Help Link */}
                                {config.helpUrl && (
                                    <a
                                        href={config.helpUrl}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="text-sm text-emerald-500 hover:text-emerald-400 flex items-center gap-1"
                                    >
                                        How to get these credentials →
                                    </a>
                                )}

                                {/* Test Result */}
                                {testResult && (
                                    <div className={`p-3 rounded-lg flex items-center gap-2 ${testResult === 'success'
                                            ? 'bg-emerald-500/10 text-emerald-500'
                                            : 'bg-red-500/10 text-red-500'
                                        }`}>
                                        {testResult === 'success' ? (
                                            <CheckCircle className="h-5 w-5" />
                                        ) : (
                                            <AlertCircle className="h-5 w-5" />
                                        )}
                                        <span className="text-sm font-medium">
                                            {testResult === 'success' ? 'Credentials verified!' : 'Verification failed'}
                                        </span>
                                    </div>
                                )}
                            </div>

                            {/* Footer */}
                            <div className="flex items-center justify-between p-6 border-t border-border bg-muted/20">
                                <button
                                    onClick={handleTest}
                                    disabled={testing || Object.keys(formData).length === 0}
                                    className="px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground disabled:opacity-50 flex items-center gap-2"
                                >
                                    {testing && <Loader2 className="h-4 w-4 animate-spin" />}
                                    Test Connection
                                </button>
                                <div className="flex gap-3">
                                    <button
                                        onClick={onClose}
                                        className="px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        onClick={handleSave}
                                        disabled={loading}
                                        className="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-white rounded-lg text-sm font-medium disabled:opacity-50 flex items-center gap-2"
                                    >
                                        {loading && <Loader2 className="h-4 w-4 animate-spin" />}
                                        Save Credentials
                                    </button>
                                </div>
                            </div>
                        </div>
                    </motion.div>
                </>
            )}
        </AnimatePresence>
    )
}

export { PROVIDER_CONFIGS }
