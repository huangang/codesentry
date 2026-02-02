import { useEffect, useRef, useCallback } from 'react';

export interface ReviewEvent {
    id: number;
    project_id: number;
    commit_sha: string;
    status: 'pending' | 'analyzing' | 'completed' | 'failed';
    score?: number;
    error?: string;
}

interface UseReviewSSEOptions {
    enabled?: boolean;
    onEvent?: (event: ReviewEvent) => void;
    onError?: (error: Event) => void;
    onConnect?: () => void;
}

/**
 * Hook for subscribing to real-time review status updates via SSE
 */
export function useReviewSSE(options: UseReviewSSEOptions = {}) {
    const { enabled = true, onEvent, onError, onConnect } = options;
    const eventSourceRef = useRef<EventSource | null>(null);
    const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const reconnectAttemptsRef = useRef(0);

    const connect = useCallback(() => {
        if (!enabled) return;

        const token = localStorage.getItem('token');
        if (!token) {
            console.warn('[SSE] No token found, skipping connection');
            return;
        }

        // Close existing connection
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
        }

        const url = `/api/events/reviews?token=${encodeURIComponent(token)}`;
        const eventSource = new EventSource(url);
        eventSourceRef.current = eventSource;

        eventSource.onopen = () => {
            console.log('[SSE] Connected');
            reconnectAttemptsRef.current = 0;
            onConnect?.();
        };

        eventSource.onmessage = (e) => {
            try {
                const event = JSON.parse(e.data) as ReviewEvent;
                onEvent?.(event);
            } catch (err) {
                console.error('[SSE] Failed to parse event:', err);
            }
        };

        eventSource.onerror = (e) => {
            console.error('[SSE] Error:', e);
            onError?.(e);
            eventSource.close();

            // Reconnect with exponential backoff
            const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
            reconnectAttemptsRef.current++;

            reconnectTimeoutRef.current = setTimeout(() => {
                console.log(`[SSE] Reconnecting (attempt ${reconnectAttemptsRef.current})...`);
                connect();
            }, delay);
        };
    }, [enabled, onEvent, onError, onConnect]);

    useEffect(() => {
        connect();

        return () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close();
                eventSourceRef.current = null;
            }
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
                reconnectTimeoutRef.current = null;
            }
        };
    }, [connect]);

    return {
        reconnect: connect,
        disconnect: () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close();
                eventSourceRef.current = null;
            }
        },
    };
}
