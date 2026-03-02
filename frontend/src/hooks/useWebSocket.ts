import { useCallback, useEffect, useRef, useState } from 'react';
import type { WSMessage, MessageType } from '../types';

interface UseWebSocketOptions {
  slug: string;
  username: string;
  onMessage: (msg: WSMessage) => void;
}

export function useWebSocket({ slug, username, onMessage }: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const onMessageRef = useRef(onMessage);
  const [connected, setConnected] = useState(false);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>();
  const reconnectAttempt = useRef(0);
  const mountedRef = useRef(false);

  onMessageRef.current = onMessage;

  const connect = useCallback(() => {
    // Don't connect if unmounted or already open
    if (!mountedRef.current) return;
    if (wsRef.current?.readyState === WebSocket.OPEN ||
        wsRef.current?.readyState === WebSocket.CONNECTING) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const ws = new WebSocket(
      `${protocol}//${host}/ws/${slug}?username=${encodeURIComponent(username)}`
    );

    ws.onopen = () => {
      if (!mountedRef.current) {
        ws.close();
        return;
      }
      setConnected(true);
      reconnectAttempt.current = 0;
      ws.send(JSON.stringify({ type: 'room:join', payload: { username } }));
    };

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);
        onMessageRef.current(msg);
      } catch (e) {
        console.error('Failed to parse WS message:', e);
      }
    };

    ws.onclose = () => {
      setConnected(false);
      wsRef.current = null;
      // Only reconnect if still mounted
      if (!mountedRef.current) return;
      const delay = Math.min(1000 * 2 ** reconnectAttempt.current, 30000);
      reconnectAttempt.current++;
      reconnectTimer.current = setTimeout(connect, delay);
    };

    ws.onerror = (err) => {
      console.error('WebSocket error:', err);
      ws.close();
    };

    wsRef.current = ws;
  }, [slug, username]);

  useEffect(() => {
    mountedRef.current = true;
    connect();
    return () => {
      mountedRef.current = false;
      clearTimeout(reconnectTimer.current);
      if (wsRef.current) {
        wsRef.current.onclose = null; // prevent reconnect from close handler
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect]);

  const send = useCallback((type: MessageType, payload?: unknown, to?: string) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) return;
    const msg: WSMessage = { type, payload, to };
    wsRef.current.send(JSON.stringify(msg));
  }, []);

  return { send, connected };
}
