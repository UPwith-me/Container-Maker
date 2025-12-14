export interface Instance {
    id: string
    name: string
    instance_type: string
    status: 'provisioning' | 'running' | 'stopped' | 'terminated'
    provider: string
    public_ip: string
    region: string
    created_at: string
}

export interface Provider {
    name: string
    display_name: string
    status: string
}

const API_BASE = '/api/v1'

// Mock Auth wrapper
const headers = {
    'Content-Type': 'application/json',
    'X-API-Key': 'cm_demo_key' // Auto-login for demo
}

export const api = {
    getInstances: async (): Promise<Instance[]> => {
        const res = await fetch(`${API_BASE}/instances`, { headers })
        return res.json()
    },

    createInstance: async (data: any): Promise<Instance> => {
        const res = await fetch(`${API_BASE}/instances`, {
            method: 'POST',
            headers,
            body: JSON.stringify(data)
        })
        if (!res.ok) throw new Error('Failed to create instance')
        return res.json()
    },

    stopInstance: async (id: string) => {
        await fetch(`${API_BASE}/instances/${id}/stop`, { method: 'POST', headers })
    },

    deleteInstance: async (id: string) => {
        await fetch(`${API_BASE}/instances/${id}`, { method: 'DELETE', headers })
    },

    getProviders: async (): Promise<Provider[]> => {
        const res = await fetch(`${API_BASE}/providers`, { headers })
        return res.json()
    }
}
