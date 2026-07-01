import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";

export function AdminBroadcastTab() {
  const [message, setMessage] = useState("");
  const send = useMutation({
    mutationFn: () => adminApi.broadcast(message),
    onSuccess: () => setMessage(""),
  });

  return (
    <form
      className="sp-surface sp-card"
      style={{ maxWidth: 480 }}
      onSubmit={(e) => {
        e.preventDefault();
        send.mutate();
      }}
    >
      <div className="sp-field">
        <label className="sp-label">Message to all connected users</label>
        <textarea className="sp-textarea" value={message} onChange={(e) => setMessage(e.target.value)} required />
      </div>
      <button className="sp-btn sp-btn--primary" type="submit" disabled={send.isPending}>
        {send.isPending ? "Sending…" : "Broadcast"}
      </button>
      {send.isSuccess && <p className="sp-mono" style={{ marginTop: 10, fontSize: 12 }}>Sent.</p>}
    </form>
  );
}
