import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { eggsApi, nodesApi, serversApi } from "../lib/endpoints";
import { StatusBadge } from "../components/StatusBadge";
import { ApiError } from "../lib/api";

export function ServersListPage() {
  const queryClient = useQueryClient();
  const { data: servers } = useQuery({ queryKey: ["servers"], queryFn: serversApi.list });
  const { data: eggs } = useQuery({ queryKey: ["eggs"], queryFn: eggsApi.list });
  const { data: nodes } = useQuery({ queryKey: ["nodes"], queryFn: nodesApi.list });

  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [eggId, setEggId] = useState("");
  const [memoryMb, setMemoryMb] = useState(1024);
  const [cpuLimit, setCpuLimit] = useState(0);
  const [variables, setVariables] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  const selectedEgg = useMemo(() => eggs?.find((e) => e.id === eggId), [eggs, eggId]);
  const editableVariables = selectedEgg?.variables.filter((v) => v.user_editable) ?? [];

  function selectEgg(id: string) {
    setEggId(id);
    const egg = eggs?.find((e) => e.id === id);
    const defaults: Record<string, string> = {};
    for (const v of egg?.variables ?? []) {
      if (v.user_editable) defaults[v.env] = v.default;
    }
    setVariables(defaults);
  }

  const createServer = useMutation({
    mutationFn: () =>
      serversApi.create({
        node_id: nodeId,
        egg_id: eggId,
        name,
        memory_bytes: memoryMb * 1024 * 1024,
        cpu_limit: cpuLimit,
        variables,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["servers"] });
      setShowForm(false);
      setName("");
    },
    onError: (err) => setError(err instanceof ApiError ? err.message : "Failed to create server"),
  });

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 18 }}>
        <h1 className="sp-page-title" style={{ marginBottom: 0 }}>
          Servers
        </h1>
        <button className="sp-btn sp-btn--primary" onClick={() => setShowForm((v) => !v)}>
          + New server
        </button>
      </div>

      {showForm && (
        <form
          className="sp-surface sp-card"
          style={{ marginBottom: 20 }}
          onSubmit={(e) => {
            e.preventDefault();
            setError(null);
            createServer.mutate();
          }}
        >
          <div className="sp-field">
            <label className="sp-label">Name</label>
            <input className="sp-input" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="sp-field">
            <label className="sp-label">Egg</label>
            <select className="sp-select" value={eggId} onChange={(e) => selectEgg(e.target.value)} required>
              <option value="" disabled>
                Select an egg
              </option>
              {eggs?.map((egg) => (
                <option key={egg.id} value={egg.id}>
                  {egg.category ? `${egg.category} — ${egg.name}` : egg.name}
                </option>
              ))}
            </select>
            {selectedEgg?.description && (
              <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 6 }}>
                {selectedEgg.description}
              </p>
            )}
          </div>
          <div className="sp-field">
            <label className="sp-label">Node</label>
            <select className="sp-select" value={nodeId} onChange={(e) => setNodeId(e.target.value)} required>
              <option value="" disabled>
                Select a node
              </option>
              {nodes?.map((node) => (
                <option key={node.id} value={node.id} disabled={!node.connected}>
                  {node.name} ({node.address}) {node.connected ? "" : "— offline"}
                </option>
              ))}
            </select>
          </div>
          <div className="sp-field">
            <label className="sp-label">Memory (MB)</label>
            <input
              className="sp-input"
              type="number"
              value={memoryMb}
              onChange={(e) => setMemoryMb(Number(e.target.value))}
              min={128}
              step={1}
            />
          </div>
          <div className="sp-field">
            <label className="sp-label">CPU limit (% of one core)</label>
            <input
              className="sp-input"
              type="number"
              value={cpuLimit}
              onChange={(e) => setCpuLimit(Number(e.target.value))}
              min={0}
              step={1}
            />
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 6 }}>
              0 = unlimited · 100 = one full core · 200 = two cores
            </p>
          </div>

          {editableVariables.length > 0 && (
            <div className="sp-field">
              <label className="sp-label">{selectedEgg?.name} options</label>
              {editableVariables.map((v) => (
                <div key={v.env} style={{ marginBottom: 8 }}>
                  <label className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
                    {v.name}
                  </label>
                  <input
                    className="sp-input sp-mono"
                    value={variables[v.env] ?? v.default}
                    onChange={(e) => setVariables((prev) => ({ ...prev, [v.env]: e.target.value }))}
                  />
                </div>
              ))}
            </div>
          )}

          {error && <p className="sp-error">{error}</p>}
          <button className="sp-btn sp-btn--primary" type="submit" disabled={createServer.isPending}>
            {createServer.isPending ? "Creating…" : "Create"}
          </button>
        </form>
      )}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Status</th>
            <th>Port</th>
            <th>Memory</th>
          </tr>
        </thead>
        <tbody>
          {servers?.map((server) => (
            <tr key={server.id}>
              <td>
                <Link to={`/servers/${server.id}`}>{server.name}</Link>
              </td>
              <td>
                <StatusBadge status={server.status} />
              </td>
              <td className="sp-mono">{server.primary_port}</td>
              <td className="sp-mono">{(server.memory_bytes / 1024 / 1024).toFixed(0)}MB</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
