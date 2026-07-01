import { useEffect, useRef } from "react";
import { API_BASE } from "./api";
import { useAuthStore } from "./authStore";

/**
 * Subscribes to a single real-time topic (e.g. "server:<id>:stats") over its
 * own dedicated WebSocket connection. One connection per topic keeps the
 * wire format simple (no need to multiplex/disambiguate several topics'
 * differently-shaped payloads over one socket) at the cost of an extra
 * connection per concurrently-watched topic, which is fine at this scale.
 *
 * Reconnects with capped exponential backoff if the connection drops, and
 * tears everything down cleanly on unmount or when `topic` changes/becomes
 * null.
 */
export function useTopic<T>(topic: string | null, onMessage: (data: T) => void) {
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;

  useEffect(() => {
    if (!topic) return;

    let socket: WebSocket | null = null;
    let closedByUs = false;
    let backoff = 1000;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;

    function connect() {
      const token = useAuthStore.getState().accessToken;
      const wsBase = API_BASE.replace(/^http/, "ws");
      const url = `${wsBase}/ws?topics=${encodeURIComponent(topic!)}&access_token=${encodeURIComponent(token ?? "")}`;

      socket = new WebSocket(url);
      socket.onmessage = (event) => {
        try {
          onMessageRef.current(JSON.parse(event.data));
        } catch {
          // ignore malformed frames
        }
      };
      socket.onclose = () => {
        if (closedByUs) return;
        reconnectTimer = setTimeout(connect, backoff);
        backoff = Math.min(backoff * 2, 15000);
      };
      socket.onopen = () => {
        backoff = 1000;
      };
    }

    connect();

    return () => {
      closedByUs = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [topic]);
}
