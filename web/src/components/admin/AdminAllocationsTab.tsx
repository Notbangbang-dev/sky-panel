import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";

type Mode = "range" | "single";

export function AdminAllocationsTab() {
  const queryClient = useQueryClient();
  const { data: nodes } = useQuery({ queryKey: ["admin", "nodes"], queryFn: adminApi.listNodes });

  const [nodeId, setNodeId] = useState("");
  const [mode, setMode] = useState<Mode>("range");
  const [port, setPort] = useState(25565);
  const [portStart, setPortStart] = useState(25565);
  const [portEnd, setPortEnd] = useState(25614);
  const [message, setMessage] = useState<{ text: string; ok: boolean } | null>(null);

  // Default to the first node once the list loads.
  useEffect(() => {
    if (!nodeId && nodes && nodes.length > 0) setNodeId(nodes[0].id);
  }, [nodes, nodeId]);

  const { data: allocations } = useQuery({
    queryKey: ["admin", "allocations", nodeId],
    queryFn: () => adminApi.listAllocations(nodeId),
    enabled: !!nodeId,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin", "allocations", nodeId] });

  const create = useMutation({
    mutationFn: () =>
      adminApi.createAllocations(nodeId, mode === "single" ? { port } : { port_start: portStart, port_end: portEnd }),
    onSuccess: (result) => {
      setMessage({ text: `Added ${result.created} new port${result.created === 1 ? "" : "s"}.`, ok: true });
      invalidate();
    },
    onError: (err) => setMessage({ text: err instanceof ApiError ? err.message : "Failed to add allocations.", ok: false }),
  });

  const remove = useMutation({
    mutationFn: (id: string) => adminApi.deleteAllocation(id),
    onSuccess: invalidate,
    onError: (err) => setMessage({ text: err instanceof ApiError ? err.message : "Failed to delete.", ok: false }),
  });

  const total = allocations?.length ?? 0;
  const free = allocations?.filter((a) => !a.server_id).length ?? 0;

  return (
    <div>
      <div className="sp-field" style={{ maxWidth: 420 }}>
        <label className="sp-label">Node</label>
        <select className="sp-select" value={nodeId} onChange={(e) => setNodeId(e.target.value)}>
          <option value="" disabled>
            Select a node
          </option>
          {nodes?.map((n) => (
            <option key={n.id} value={n.id}>
              {n.name} ({n.address})
            </option>
          ))}
        </select>
      </div>

      {nodeId && (
        <>
          <form
            className="sp-surface sp-card"
            style={{ marginBottom: 20, maxWidth: 520 }}
            onSubmit={(e) => {
              e.preventDefault();
              setMessage(null);
              create.mutate();
            }}
          >
            <div className="sp-field">
              <label className="sp-label">Add ports</label>
              <div style={{ display: "flex", gap: 6, marginBottom: 10 }}>
                <button
                  type="button"
                  className="sp-btn sp-btn--sm"
                  style={mode === "range" ? { background: "var(--sp-accent)", color: "var(--sp-accent-text)" } : undefined}
                  onClick={() => setMode("range")}
                >
                  Range
                </button>
                <button
                  type="button"
                  className="sp-btn sp-btn--sm"
                  style={mode === "single" ? { background: "var(--sp-accent)", color: "var(--sp-accent-text)" } : undefined}
                  onClick={() => setMode("single")}
                >
                  Single
                </button>
              </div>
              {mode === "single" ? (
                <input
                  className="sp-input sp-mono"
                  type="number"
                  min={1}
                  max={65535}
                  value={port}
                  onChange={(e) => setPort(Number(e.target.value))}
                />
              ) : (
                <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                  <input
                    className="sp-input sp-mono"
                    type="number"
                    min={1}
                    max={65535}
                    value={portStart}
                    onChange={(e) => setPortStart(Number(e.target.value))}
                  />
                  <span className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
                    to
                  </span>
                  <input
                    className="sp-input sp-mono"
                    type="number"
                    min={1}
                    max={65535}
                    value={portEnd}
                    onChange={(e) => setPortEnd(Number(e.target.value))}
                  />
                </div>
              )}
            </div>
            <button className="sp-btn sp-btn--primary" type="submit" disabled={create.isPending}>
              {create.isPending ? "Adding…" : "Add allocations"}
            </button>
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 10 }}>
              Existing ports are skipped. New nodes are auto-seeded with {25565}–{25614}. The daemon publishes each
              allocated port on the node's host (TCP + UDP) when a server claims it.
            </p>
          </form>

          {message && (
            <p className="sp-mono" style={{ fontSize: 13, color: message.ok ? "var(--sp-accent)" : "#ff9b9b", marginBottom: 12 }}>
              {message.text}
            </p>
          )}

          <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginBottom: 10 }}>
            {total} port{total === 1 ? "" : "s"} · {free} free · {total - free} in use
          </p>

          <table className="sp-table">
            <thead>
              <tr>
                <th>Port</th>
                <th>Status</th>
                <th>Server</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {allocations?.map((a) => (
                <tr key={a.id}>
                  <td className="sp-mono">{a.port}</td>
                  <td>
                    <span className={"sp-badge" + (a.server_id ? " sp-badge--running" : "")}>
                      {a.server_id ? "in use" : "free"}
                    </span>
                  </td>
                  <td className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
                    {a.server_name || (a.server_id ? a.server_id.slice(0, 8) + "…" : "—")}
                  </td>
                  <td>
                    {!a.server_id && (
                      <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => remove.mutate(a.id)}>
                        Delete
                      </button>
                    )}
                  </td>
                </tr>
              ))}
              {total === 0 && (
                <tr>
                  <td colSpan={4} className="sp-mono">
                    no allocations on this node yet — add a range above
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </>
      )}
    </div>
  );
}
