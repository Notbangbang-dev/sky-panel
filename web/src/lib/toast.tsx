import { create } from "zustand";
import { AnimatePresence, motion } from "framer-motion";

export type ToastKind = "info" | "success" | "error";

export interface Toast {
  id: number;
  message: string;
  kind: ToastKind;
}

interface ToastState {
  toasts: Toast[];
  push: (message: string, kind?: ToastKind) => void;
  dismiss: (id: number) => void;
}

let nextId = 1;

// A tiny global toast store. Any module can fire a toast via the `toast`
// helper below without threading a context through the tree — useful for the
// API layer and react-query's global error handler.
export const useToasts = create<ToastState>((set) => ({
  toasts: [],
  push: (message, kind = "info") => {
    const id = nextId++;
    set((s) => ({ toasts: [...s.toasts, { id, message, kind }] }));
    setTimeout(() => {
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
    }, 5000);
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));

export const toast = {
  info: (m: string) => useToasts.getState().push(m, "info"),
  success: (m: string) => useToasts.getState().push(m, "success"),
  error: (m: string) => useToasts.getState().push(m, "error"),
};

const accentFor: Record<ToastKind, string> = {
  info: "var(--sp-text-muted)",
  success: "var(--sp-accent, #8affc1)",
  error: "#ff9b9b",
};

// ToastHost renders the stack of active toasts. Mounted once at the app root.
export function ToastHost() {
  const toasts = useToasts((s) => s.toasts);
  const dismiss = useToasts((s) => s.dismiss);

  return (
    <div className="sp-toast-host" aria-live="polite" aria-atomic="false">
      <AnimatePresence>
        {toasts.map((t) => (
          <motion.div
            key={t.id}
            className="sp-surface sp-toast"
            role="status"
            initial={{ x: 40, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            exit={{ x: 40, opacity: 0 }}
            style={{ borderLeft: `3px solid ${accentFor[t.kind]}` }}
            onClick={() => dismiss(t.id)}
          >
            <span className="sp-mono sp-toast__kind" style={{ color: accentFor[t.kind] }}>
              {t.kind}
            </span>
            <span className="sp-toast__msg">{t.message}</span>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}
