import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";

// Admin-only per-server port manager. Shows the server's primary + additional
// ports and lets an admin attach a free port from the server's node or detach
// an additional one. Each change recreates the container on the node.
export function ServerPortsPanel({ serverId }: { serverId: string }) {
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState("");
  const [error, setError] = useState<string | null>(null);

  const queryKey = ["admin", "server-allocations", serverId];
  const { data, isLoading } = useQuery({
    queryKey,
    queryFn: () => adminApi.listServerAllocations(serverId),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey });
    // The node's allocations tab and the servers list both reflect port usage.
    queryClient.invalidateQueries({ queryKey: ["admin", "allocations"] });
  };

  const add = useMutation({
    mutationFn: (allocationId: string) => adminApi.addServerAllocation(serverId, allocationId),
    onSuccess: () => {
      setSelected("");
      setError(null);
      invalidate();
    },
    onError: (err) => setError(err instanceof ApiError ? err.message : "Failed to add port."),
  });
  const remove = useMutation({
    mutationFn: (allocationId: string) => adminApi.removeServerAllocation(serverId, allocationId),
    onSuccess: () => {
      setError(null);
      invalidate();
    },
    onError: (err) => setError(err instanceof ApiError ? err.message : "Failed to remove port."),
  });

  const busy = add.isPending || remove.isPending;
  const hasFree = (data?.free.length ?? 0) > 0;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
      <div style={{ display: "flex", gap: 6, flexWrap: "wrap", alignItems: "center" }}>
        <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>Ports</span>
        {isLoading && <span className="sp-mono" style={{ fontSize: 12 }}>loading…</span>}
        {data?.ports.map((p) => (
          <span
            key={p.id}
            className={"sp-badge" + (p.primary ? " sp-badge--running" : "")}
            style={{ display: "inline-flex", gap: 6, alignItems: "center" }}
          >
            <span className="sp-mono">{p.port}</span>
            {p.primary ? (
              <span style={{ fontSize: 10, opacity: 0.8 }}>primary</span>
            ) : (
              <button
                className="sp-btn sp-btn--sm sp-btn--danger"
                disabled={busy}
                onClick={() => remove.mutate(p.id)}
                title="Remove this port"
                aria-label={`Remove port ${p.port}`}
              >
                ×
              </button>
            )}
          </span>
        ))}
        {data && data.ports.length === 0 && <span className="sp-mono" style={{ fontSize: 12 }}>none</span>}
      </div>

      <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
        <select
          className="sp-select"
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
          style={{ width: 180 }}
          disabled={busy || !hasFree}
        >
          <option value="">{hasFree ? "Add a free port…" : "no free ports on this node"}</option>
          {data?.free.map((f) => (
            <option key={f.id} value={f.id}>
              {f.port}
            </option>
          ))}
        </select>
        <button
          className="sp-btn sp-btn--sm sp-btn--primary"
          disabled={!selected || busy}
          onClick={() => selected && add.mutate(selected)}
        >
          {add.isPending ? "Adding…" : "Add port"}
        </button>
      </div>

      {error && (
        <p className="sp-mono" style={{ fontSize: 12, color: "#ff9b9b", margin: 0 }}>
          {error}
        </p>
      )}
      <p className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", margin: 0 }}>
        Additional ports are published (TCP + UDP) and opened in the node firewall. Adding or removing a
        port recreates the container — the server restarts, but its files are preserved.
      </p>
    </div>
  );
}
