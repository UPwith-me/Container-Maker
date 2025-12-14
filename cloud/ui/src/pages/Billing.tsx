import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
    CreditCard,
    TrendingUp,
    Calendar,
    Download,
    Server,
    Clock
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface UsageData {
    current_month: {
        cpu_hours: number
        gpu_hours: number
        total_cost: number
        instances: number
        forecast: number
    }
}

export default function Billing() {
    const [usage, setUsage] = useState<UsageData | null>(null)
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        fetch('/api/v1/billing/usage', {
            headers: { 'X-API-Key': 'cm_demo' }
        })
            .then(res => res.json())
            .then(setUsage)
            .finally(() => setLoading(false))
    }, [])

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
        cpu_hours: 124.5,
        gpu_hours: 12.0,
        total_cost: 45.20,
        instances: 3,
        forecast: 85.00
    }

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
                        <p className="text-lg font-semibold">December 2024</p>
                    </div>
                    <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 text-emerald-500 text-sm">
                        <Calendar className="h-4 w-4" />
                        <span>14 days remaining</span>
                    </div>
                </div>

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
                    <p className="text-2xl font-bold">${(data.total_cost / Math.max(data.cpu_hours + data.gpu_hours, 1)).toFixed(3)}</p>
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
                    <button className="text-sm text-emerald-500 hover:text-emerald-400">
                        Update
                    </button>
                </div>

                <div className="flex items-center gap-4 p-4 rounded-lg bg-muted/30 border border-border/40">
                    <div className="h-10 w-16 rounded bg-gradient-to-br from-blue-600 to-blue-800 flex items-center justify-center">
                        <CreditCard className="h-5 w-5 text-white" />
                    </div>
                    <div>
                        <p className="font-medium">•••• •••• •••• 4242</p>
                        <p className="text-sm text-muted-foreground">Expires 12/25</p>
                    </div>
                </div>
            </div>

            {/* Invoices */}
            <div className="p-6 rounded-xl border border-border/40 bg-card/30">
                <div className="flex items-center justify-between mb-6">
                    <h3 className="text-lg font-semibold">Recent Invoices</h3>
                    <button className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1">
                        <Download className="h-4 w-4" />
                        Download All
                    </button>
                </div>

                <div className="space-y-3">
                    {[
                        { id: 'INV-2024-0011', date: 'Nov 2024', amount: 42.50, status: 'paid' },
                        { id: 'INV-2024-0010', date: 'Oct 2024', amount: 38.20, status: 'paid' },
                        { id: 'INV-2024-0009', date: 'Sep 2024', amount: 51.00, status: 'paid' },
                    ].map(invoice => (
                        <div key={invoice.id} className="flex items-center justify-between p-3 rounded-lg hover:bg-muted/30 transition-colors">
                            <div>
                                <p className="font-medium">{invoice.id}</p>
                                <p className="text-sm text-muted-foreground">{invoice.date}</p>
                            </div>
                            <div className="flex items-center gap-4">
                                <span className="text-sm font-medium">${invoice.amount.toFixed(2)}</span>
                                <span className="px-2 py-1 rounded text-xs font-medium bg-emerald-500/10 text-emerald-500 capitalize">
                                    {invoice.status}
                                </span>
                                <button className="text-muted-foreground hover:text-foreground">
                                    <Download className="h-4 w-4" />
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    )
}
