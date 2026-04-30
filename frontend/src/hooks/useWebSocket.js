import { useEffect, useRef, useState } from "react";

const INITIAL_BACKOFF_MS = 1000;
const MAX_BACKOFF_MS = 30000;

export function useWebSocket(url, { onMessage, enabled = true } = {}) {
  const [status, setStatus] = useState("idle");
  const socketRef = useRef(null);
  const backoffRef = useRef(INITIAL_BACKOFF_MS);
  const reconnectTimerRef = useRef(null);
  const mountedRef = useRef(true);
  const onMessageRef = useRef(onMessage);

  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    mountedRef.current = true;

    if (!enabled || !url) {
      setStatus("idle");
      return () => {
        mountedRef.current = false;
      };
    }

    const connect = () => {
      if (!mountedRef.current) return;
      setStatus("connecting");

      let socket;
      try {
        socket = new WebSocket(url);
      } catch {
        scheduleReconnect();
        return;
      }
      socketRef.current = socket;

      socket.onopen = () => {
        if (!mountedRef.current) return;
        backoffRef.current = INITIAL_BACKOFF_MS;
        setStatus("open");
      };

      socket.onmessage = (event) => {
        if (!mountedRef.current) return;
        try {
          const parsed = JSON.parse(event.data);
          if (onMessageRef.current) onMessageRef.current(parsed);
        } catch (err) {
          console.warn("[useWebSocket] failed to parse message", err);
        }
      };

      socket.onerror = () => {
        if (!mountedRef.current) return;
        setStatus("error");
      };

      socket.onclose = () => {
        if (!mountedRef.current) return;
        setStatus("closed");
        scheduleReconnect();
      };
    };

    const scheduleReconnect = () => {
      if (!mountedRef.current) return;
      const delay = backoffRef.current;
      backoffRef.current = Math.min(backoffRef.current * 2, MAX_BACKOFF_MS);
      reconnectTimerRef.current = window.setTimeout(() => {
        reconnectTimerRef.current = null;
        connect();
      }, delay);
    };

    connect();

    return () => {
      mountedRef.current = false;
      if (reconnectTimerRef.current !== null) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
      const socket = socketRef.current;
      socketRef.current = null;
      if (socket && socket.readyState !== WebSocket.CLOSED) {
        try {
          socket.close(1000, "unmount");
        } catch {
          // ignore
        }
      }
    };
  }, [url, enabled]);

  return { status };
}

export default useWebSocket;
