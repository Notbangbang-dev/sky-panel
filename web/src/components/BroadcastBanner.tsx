import { useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { useTopic } from "../lib/useTopic";
import { useAuthStore } from "../lib/authStore";

interface BroadcastMessage {
  message: string;
  created_at: string;
}

export function BroadcastBanner() {
  const isAuthed = useAuthStore((s) => !!s.user);
  const [banner, setBanner] = useState<BroadcastMessage | null>(null);

  useTopic<BroadcastMessage>(isAuthed ? "broadcast" : null, setBanner);

  return (
    <div className="sp-broadcast-slot">
      <AnimatePresence>
        {banner && (
          <motion.div
            className="sp-surface sp-broadcast"
            initial={{ y: -40, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -40, opacity: 0 }}
            onClick={() => setBanner(null)}
          >
            <span className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", marginRight: 8 }}>
              announcement
            </span>
            {banner.message}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
