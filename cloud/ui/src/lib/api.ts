// API Client with real authentication support

const API_BASE = '/api/v1'

// Get auth headers from localStorage
function getAuthHeaders(): Record<string, string> {
    const token = localStorage.getItem('access_token')
    if (token) {
        return {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        }
    }
    // Fallback for demo mode (when not logged in)
    return {
        'X-API-Key': 'cm_demo',
        'Content-Type': 'application/json'
    }
}

// Generic fetch wrapper with error handling
async function request<T>(
    endpoint: string,
    options: RequestInit = {}
): Promise<T> {
    const headers = {
        ...getAuthHeaders(),
        ...options.headers
    }

    const res = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers
    })

    if (res.status === 401) {
        // Try to refresh token
        const refreshToken = localStorage.getItem('refresh_token')
        if (refreshToken) {
            const refreshRes = await fetch(`${API_BASE}/auth/refresh`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ refresh_token: refreshToken })
            })

            if (refreshRes.ok) {
                const data = await refreshRes.json()
                localStorage.setItem('access_token', data.access_token)
                localStorage.setItem('refresh_token', data.refresh_token)

                // Retry original request
                return request(endpoint, options)
            }
        }

        // Redirect to login if refresh fails
        window.location.href = '/login'
        throw new Error('Session expired')
    }

    if (!res.ok) {
        const errorData = await res.json().catch(() => ({}))
        throw new Error(errorData.message || `Request failed: ${res.status}`)
    }

    // Handle empty responses
    const text = await res.text()
    if (!text) return {} as T

    return JSON.parse(text)
}

// TypeScript interfaces
export interface Instance {
    id: string
    name: string
    instance_type: string
    status: string
    provider: string
    region: string
    public_ip?: string
    ssh_port?: number
    hourly_rate?: number
    created_at: string
    updated_at?: string
    status_reason?: string
}

export interface Provider {
    name: string
    display_name: string
    description: string
    website: string
    status: string
    features: string[]
    required_credentials: string[]
}

export interface Region {
    id: string
    name: string
    country: string
    available: boolean
    gpu_available: boolean
}

export interface InstanceType {
    type: string
    hourly_rate: number
    vcpu: number
    memory_gb: number
    gpu_type?: string
    gpu_memory_gb?: number
}

export interface APIKey {
    id: string
    name: string
    key_prefix: string
    created_at: string
    last_used_at?: string
}

export interface CloudCredential {
    id: string
    provider: string
    name: string
    is_valid: boolean
    created_at: string
    updated_at: string
}

export interface Invoice {
    id: string
    stripe_invoice_id?: string
    amount: number
    currency: string
    status: string
    period_start: string
    period_end: string
    invoice_url?: string
    created_at: string
}

export interface UsageData {
    current_month: {
        cpu_hours: number
        gpu_hours: number
        total_cost: number
        instances: number
        forecast: number
    }
}

export interface User {
    id: string
    email: string
    name: string
    avatar_url?: string
}

// API Methods
export const api = {
    // Auth
    login: (email: string, password: string) =>
        request<{ access_token: string; refresh_token: string }>('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password })
        }),

    register: (email: string, password: string, name: string) =>
        request<{ message: string }>('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ email, password, name })
        }),

    // User
    getCurrentUser: () => request<User>('/user'),

    updateUser: (data: Partial<User>) =>
        request<User>('/user', {
            method: 'PUT',
            body: JSON.stringify(data)
        }),

    // Instances
    getInstances: () => request<Instance[]>('/instances'),

    getInstance: (id: string) => request<Instance>(`/instances/${id}`),

    createInstance: (data: {
        name: string
        provider: string
        instance_type: string
        region: string
        image?: string
    }) => request<Instance>('/instances', {
        method: 'POST',
        body: JSON.stringify(data)
    }),

    startInstance: (id: string) =>
        request<Instance>(`/instances/${id}/start`, { method: 'POST' }),

    stopInstance: (id: string) =>
        request<Instance>(`/instances/${id}/stop`, { method: 'POST' }),

    deleteInstance: (id: string) =>
        request<void>(`/instances/${id}`, { method: 'DELETE' }),

    getInstanceLogs: (id: string) =>
        request<{ logs: string }>(`/instances/${id}/logs`),

    getSSHConfig: (id: string) =>
        request<{ host: string; port: number; user: string }>(`/instances/${id}/ssh`),

    // Providers
    getProviders: () => request<Provider[]>('/providers'),

    getProviderRegions: (name: string) => request<Region[]>(`/providers/${name}/regions`),

    getProviderInstanceTypes: (name: string) =>
        request<InstanceType[]>(`/providers/${name}/types`),

    // API Keys
    getAPIKeys: () => request<APIKey[]>('/api-keys'),

    createAPIKey: (name: string) =>
        request<{ key: string; id: string }>('/api-keys', {
            method: 'POST',
            body: JSON.stringify({ name })
        }),

    deleteAPIKey: (id: string) =>
        request<void>(`/api-keys/${id}`, { method: 'DELETE' }),

    // Cloud Credentials
    getCredentials: () => request<CloudCredential[]>('/credentials'),

    addCredential: (data: {
        provider: string
        name: string
        data: Record<string, string>
    }) => request<CloudCredential>('/credentials', {
        method: 'POST',
        body: JSON.stringify(data)
    }),

    deleteCredential: (id: string) =>
        request<void>(`/credentials/${id}`, { method: 'DELETE' }),

    verifyCredential: (id: string) =>
        request<{ verified: boolean; message: string }>(`/credentials/${id}/verify`, {
            method: 'POST'
        }),

    // Billing
    getUsage: () => request<UsageData>('/billing/usage'),

    getInvoices: () => request<Invoice[]>('/billing/invoices'),

    getInvoicePdfUrl: (id: string) =>
        request<{ url: string }>(`/billing/invoices/${id}/pdf`),

    createBillingPortalSession: () =>
        request<{ url: string }>('/billing/portal', { method: 'POST' }),

    createSetupIntent: () =>
        request<{ client_secret: string }>('/billing/setup-intent', { method: 'POST' }),
}
