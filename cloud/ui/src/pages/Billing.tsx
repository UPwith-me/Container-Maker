import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
    CreditCard,
    TrendingUp,
    Calendar,
    Download,
    Server,
    Clock,
    Plus,
    ExternalLink,
    Loader2,
    AlertCircle
} from 'lucide-react'
import { api, type UsageData, type Invoice } from '@/lib/api'
import { toast } from 'sonner'

export default function Billing() {
    const [usage, setUsage] = useState<UsageData | null>(null)
    const [invoices, setInvoices] = useState<Invoice[]>([])
    const [loading, setLoading] = useState(true)
    const [portalLoading, setPortalLoading] = useState(false)
    const [downloadingAll, setDownloadingAll] = useState(false)
    const [downloadingId, setDownloadingId] = useState<string | null>(null)

    useEffect(() => {
        loadData()
    }, [])

    const loadData = async () => {
        setLoading(true)
        try {
            const [usageData, invoiceData] = await Promise.all([
                api.getUsage(),
                api.getInvoices()
            ])
            setUsage(usageData)
            setInvoices(invoiceData || [])
        } catch (e) {
            console.error('Failed to load billing data:', e)
        } finally {
            setLoading(false)
        }
    }

    const openBillingPortal = async () => {
        setPortalLoading(true)
        try {
            const { url } = await api.createBillingPortalSession()
            window.open(url, '_blank')
            toast.success('Opening billing portal...')
        } catch (e: any) {
            toast.error(e.message || 'Failed to open billing portal. Please configure Stripe in Admin settings.')
        } finally {
            setPortalLoading(false)
        }
    }

    const downloadInvoice = async (invoice: Invoice) => {
        setDownloadingId(invoice.id)
        try {
            if (invoice.invoice_url) {
                window.open(invoice.invoice_url, '_blank')
                toast.success('Opening invoice...')
            } else {
                const { url } = await api.getInvoicePdfUrl(invoice.id)
                window.open(url, '_blank')
                toast.success('Downloading invoice...')
            }
        } catch (e: any) {
            toast.error(e.message || 'Failed to download invoice')
        } finally {
            setDownloadingId(null)
        }
    }

    const downloadAllInvoices = async () => {
        if (invoices.length === 0) {
            toast.error('No invoices to download')
            return
        }

        setDownloadingAll(true)
        toast.loading('Downloading all invoices...')

        try {
            for (const invoice of invoices) {
                await downloadInvoice(invoice)
                // Small delay between downloads
                await new Promise(resolve => setTimeout(resolve, 500))
            }
            toast.success('All invoices downloaded!')
        } catch (e) {
            toast.error('Some invoices failed to download')
        } finally {
            setDownloadingAll(false)
        }
    }

    // Get current month/year dynamically
    const now = new Date()
    const currentMonth = now.toLocaleString('en-US', { month: 'long', year: 'numeric' })
    const daysInMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate()
    const daysRemaining = daysInMonth - now.getDate()

    if (loading) {
        return (
            <div className="space-y-6">
                <div className="h-32 bg-muted/20 animate-pulse rounded-xl" />
                <div className="grid grid-cols-3 gap-6">
                    {[1, 2, 3].map(i => (
                        <div key={i} className="h-24 bg-muted/20 animate-pulse rounded-xl" />
                    ))}
                </div>
            </div>
        )
    }

    const data = usage?.current_month || {
        cpu_hours: 0,
        gpu_hours: 0,
        total_cost: 0,
        instances: 0,
        forecast: 0
    }

    const hasNoUsage = data.cpu_hours === 0 && data.gpu_hours === 0 && data.total_cost === 0

    return (
        <div className="space-y-8">
            <div>
                <h2 className="text-2xl font-bold mb-2">Billing & Usage</h2>
                <p className="text-muted-foreground">Monitor your resource consumption and manage payments</p>
            </div>

            {/* Current Month Summary */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                className="p-6 rounded-xl border border-border/40 bg-gradient-to-br from-emerald-500/10 to-emerald-500/5"
            >
                <div className="flex items-center justify-between mb-6">
                    <div>
                        <p className="text-sm text-muted-foreground mb-1">Current Period</p>
                        <p className="text-lg font-semibold">{currentMonth}</p>
                    </div>
                    <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 text-emerald-500 text-sm">
                        <Calendar className="h-4 w-4" />
                        <span>{daysRemaining} days remaining</span>
                    </div>
                </div>

                {hasNoUsage ? (
                    <div className="text-center py-6">
                        <p className="text-muted-foreground mb-2">No usage this month</p>
                        <p className="text-sm text-muted-foreground">Create an instance to start tracking usage</p>
                    </div>
                ) : (
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
                        <div>
                            <p className="text-sm text-muted-foreground mb-1">Total Spend</p>
                            <p className="text-3xl font-bold">${data.total_cost.toFixed(2)}</p>
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground mb-1">Forecasted</p>
                            <p className="text-3xl font-bold text-muted-foreground">${data.forecast.toFixed(2)}</p>
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground mb-1">CPU Hours</p>
                            <p className="text-3xl font-bold">{data.cpu_hours.toFixed(1)}</p>
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground mb-1">GPU Hours</p>
                            <p className="text-3xl font-bold">{data.gpu_hours.toFixed(1)}</p>
                        </div>
                    </div>
                )}
            </motion.div>

            {/* Quick Stats */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="p-5 rounded-xl border border-border/40 bg-card/30">
                    <div className="flex items-center justify-between mb-3">
                        <span className="text-sm text-muted-foreground">Active Instances</span>
                        <Server className="h-4 w-4 text-emerald-500" />
                    </div>
                    <p className="text-2xl font-bold">{data.instances}</p>
                </div>

                <div className="p-5 rounded-xl border border-border/40 bg-card/30">
                    <div className="flex items-center justify-between mb-3">
                        <span className="text-sm text-muted-foreground">Avg. Hourly Cost</span>
                        <TrendingUp className="h-4 w-4 text-amber-500" />
                    </div>
                    <p className="text-2xl font-bold">
                        ${data.cpu_hours + data.gpu_hours > 0
                            ? (data.total_cost / (data.cpu_hours + data.gpu_hours)).toFixed(3)
                            : '0.000'}
                    </p>
                </div>

                <div className="p-5 rounded-xl border border-border/40 bg-card/30">
                    <div className="flex items-center justify-between mb-3">
                        <span className="text-sm text-muted-foreground">Total Runtime</span>
                        <Clock className="h-4 w-4 text-blue-500" />
                    </div>
                    <p className="text-2xl font-bold">{(data.cpu_hours + data.gpu_hours).toFixed(1)}h</p>
                </div>
            </div>

            {/* Payment Method */}
            <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                <div className="flex items-center justify-between mb-6">
                    <h3 className="text-lg font-semibold">Payment Method</h3>
                    <button
                        onClick={openBillingPortal}
                        disabled={portalLoading}
                        className="text-sm text-emerald-500 hover:text-emerald-400 flex items-center gap-1 disabled:opacity-50"
                    >
                        {portalLoading && <Loader2 className="h-3 w-3 animate-spin" />}
                        Manage
                        <ExternalLink className="h-3 w-3" />
                    </button>
                </div>

                {/* Check if Stripe is configured - for now show setup prompt */}
                <div className="space-y-4">
                    <div className="p-4 rounded-lg bg-muted/30 border border-border/40">
                        <div className="flex items-center gap-3">
                            <div className="h-10 w-16 rounded bg-gradient-to-br from-blue-600 to-blue-800 flex items-center justify-center">
                                <CreditCard className="h-5 w-5 text-white" />
                            </div>
                            <div className="flex-1">
                                <p className="font-medium">Add Payment Method</p>
                                <p className="text-sm text-muted-foreground">
                                    Click "Manage" to add a card via Stripe
                                </p>
                            </div>
                            <button
                                onClick={openBillingPortal}
                                disabled={portalLoading}
                                className="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-white rounded-lg text-sm font-medium flex items-center gap-2 disabled:opacity-50"
                            >
                                {portalLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                                Add Card
                            </button>
                        </div>
                    </div>

                    <div className="flex items-start gap-2 text-sm text-muted-foreground">
                        <AlertCircle className="h-4 w-4 flex-shrink-0 mt-0.5" />
                        <span>
                            Payment processing powered by Stripe. Configure Stripe API keys in Settings › Admin to enable.
                        </span>
                    </div>
                </div>
            </div>

            {/* Invoices */}
            <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                <div className="flex items-center justify-between mb-6">
                    <h3 className="text-lg font-semibold">Invoices</h3>
                    {invoices.length > 0 && (
                        <button
                            onClick={downloadAllInvoices}
                            disabled={downloadingAll}
                            className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 disabled:opacity-50"
                        >
                            {downloadingAll ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
                            Download All
                        </button>
                    )}
                </div>

                {invoices.length === 0 ? (
                    <div className="text-center py-8">
                        <CreditCard className="h-10 w-10 text-muted-foreground/50 mx-auto mb-3" />
                        <p className="text-muted-foreground">No invoices yet</p>
                        <p className="text-sm text-muted-foreground">Invoices will appear here after your first billing cycle</p>
                    </div>
                ) : (
                    <div className="space-y-3">
                        {invoices.map(invoice => (
                            <div
                                key={invoice.id}
                                className="flex items-center justify-between p-4 rounded-lg hover:bg-muted/30 transition-colors border border-transparent hover:border-border/40"
                            >
                                <div>
                                    <p className="font-medium">{invoice.id}</p>
                                    <p className="text-sm text-muted-foreground">
                                        {new Date(invoice.created_at).toLocaleDateString('en-US', {
                                            month: 'short',
                                            year: 'numeric'
                                        })}
                                    </p>
                                </div>
                                <div className="flex items-center gap-4">
                                    <span className="text-sm font-medium">
                                        ${invoice.amount.toFixed(2)} {invoice.currency?.toUpperCase()}
                                    </span>
                                    <span className={`px-2 py-1 rounded text-xs font-medium capitalize ${invoice.status === 'paid'
                                            ? 'bg-emerald-500/10 text-emerald-500'
                                            : invoice.status === 'open'
                                                ? 'bg-amber-500/10 text-amber-500'
                                                : 'bg-muted text-muted-foreground'
                                        }`}>
                                        {invoice.status}
                                    </span>
                                    <button
                                        onClick={() => downloadInvoice(invoice)}
                                        disabled={downloadingId === invoice.id}
                                        className="p-2 text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded-lg transition-colors disabled:opacity-50"
                                    >
                                        {downloadingId === invoice.id
                                            ? <Loader2 className="h-4 w-4 animate-spin" />
                                            : <Download className="h-4 w-4" />
                                        }
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    )
}
