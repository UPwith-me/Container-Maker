// WebSocket hook for real-time updates
import { useEffect, useRef, useCallback, useState } from 'react'

interface WSMessage {
    type: string
    payload: any
}

interface UseWebSocketOptions {
    onMessage?: (msg: WSMessage) => void
    onConnect?: () => void
    onDisconnect?: () => void
    autoReconnect?: boolean
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
    const { onMessage, onConnect, onDisconnect, autoReconnect = true } = options
    const wsRef = useRef<WebSocket | null>(null)
    const [connected, setConnected] = useState(false)
    const [lastMessage, setLastMessage] = useState<WSMessage | null>(null)
    const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

    const connect = useCallback(() => {
        // Get auth token
        const token = localStorage.getItem('access_token') || 'cm_demo'

        // Build WebSocket URL
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const host = window.location.host
        const wsUrl = `${protocol}//${host}/api/v1/ws?token=${token}`

        try {
            const ws = new WebSocket(wsUrl)

            ws.onopen = () => {
                console.log('WebSocket connected')
                setConnected(true)
                onConnect?.()
            }

            ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data)
                    setLastMessage(data)
                    onMessage?.(data)
                } catch (e) {
                    console.error('Failed to parse WebSocket message:', e)
                }
            }

            ws.onclose = () => {
                console.log('WebSocket disconnected')
                setConnected(false)
                onDisconnect?.()

                // Auto-reconnect after 3 seconds
                if (autoReconnect) {
                    reconnectTimeoutRef.current = setTimeout(() => {
                        console.log('WebSocket reconnecting...')
                        connect()
                    }, 3000)
                }
            }

            ws.onerror = (error) => {
                console.error('WebSocket error:', error)
                ws.close()
            }

            wsRef.current = ws
        } catch (e) {
            console.error('Failed to create WebSocket:', e)
        }
    }, [onMessage, onConnect, onDisconnect, autoReconnect])

    const disconnect = useCallback(() => {
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current)
        }
        if (wsRef.current) {
            wsRef.current.close()
            wsRef.current = null
        }
    }, [])

    const send = useCallback((msg: WSMessage) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify(msg))
        }
    }, [])

    // Connect on mount, disconnect on unmount
    useEffect(() => {
        connect()
        return () => disconnect()
    }, [connect, disconnect])

    return {
        connected,
        lastMessage,
        send,
        disconnect,
        reconnect: connect,
    }
}

// Hook for subscribing to instance updates
export function useInstanceUpdates(
    instanceId: string | null,
    onUpdate: (status: string, details: any) => void
) {
    const { lastMessage } = useWebSocket()

    useEffect(() => {
        if (!lastMessage || !instanceId) return

        if (
            lastMessage.type === 'instance_update' &&
            lastMessage.payload?.instance_id === instanceId
        ) {
            onUpdate(lastMessage.payload.status, lastMessage.payload)
        }
    }, [lastMessage, instanceId, onUpdate])
}

// Hook for all instance updates (dashboard)
export function useDashboardUpdates(
    onInstanceUpdate: (instanceId: string, status: string, details: any) => void
) {
    const { connected, lastMessage } = useWebSocket()

    useEffect(() => {
        if (!lastMessage) return

        if (lastMessage.type === 'instance_update') {
            const { instance_id, status, ...details } = lastMessage.payload
            onInstanceUpdate(instance_id, status, details)
        }
    }, [lastMessage, onInstanceUpdate])

    return { connected }
}
