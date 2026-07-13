import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import { eggsApi, favoritesApi, nodesApi, quotaApi, serversApi } from "../lib/endpoints";
import { StatusBadge } from "../components/StatusBadge";
import { QuotaMeters } from "../components/QuotaMeters";
import { bytesPerMB } from "../lib/format";
import { ApiError } from "../lib/api";

export function ServersListPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { data: servers } = useQuery({
    queryKey: ["servers"],
    queryFn: serversApi.list,
    // While a server is provisioning (async, may pull an image), poll so it
    // flips from "installing" to "running" in the list without a manual reload.
    refetchInterval: (query) =>
      query.state.data?.some((s) => s.status === "installing" || s.status === "stopping") ? 3000 : false,
  });
  const { data: eggs } = useQuery({ queryKey: ["eggs"], queryFn: eggsApi.list });
  const { data: nodes } = useQuery({ queryKey: ["nodes"], queryFn: nodesApi.list });
  const { data: quota } = useQuery({ queryKey: ["quota"], queryFn: quotaApi.mine });
  const { data: favorites } = useQuery({ queryKey: ["favorites"], queryFn: favoritesApi.list });
  const allowUnlimitedCpu = quota?.allow_unlimited_cpu ?? true;

  const favoriteSet = useMemo(() => new Set(favorites ?? []), [favorites]);

  const toggleFavorite = useMutation({
    mutationFn: ({ id, on }: { id: string; on: boolean }) =>
      on ? serversApi.favorite(id) : serversApi.unfavorite(id),
    // Optimistic: flip the star instantly, roll back on error.
    onMutate: async ({ id, on }) => {
      await queryClient.cancelQueries({ queryKey: ["favorites"] });
      const prev = queryClient.getQueryData<string[]>(["favorites"]) ?? [];
      queryClient.setQueryData<string[]>(["favorites"], on ? [...prev, id] : prev.filter((x) => x !== id));
      return { prev };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(["favorites"], ctx.prev);
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["favorites"] }),
  });

  const cloneServer = useMutation({
    mutationFn: (id: string) => serversApi.clone(id),
    onSuccess: (server) => {
      queryClient.invalidateQueries({ queryKey: ["servers"] });
      queryClient.invalidateQueries({ queryKey: ["quota"] });
      navigate(`/servers/${server.id}`);
    },
    onError: (err) => setError(err instanceof ApiError ? err.message : "Failed to clone server"),
  });

  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const sortedServers = useMemo(() => {
    if (!servers) return servers;
    const q = search.trim().toLowerCase();
    return [...servers]
      .filter((s) => (q === "" ? true : s.name.toLowerCase().includes(q)))
      .filter((s) => (statusFilter === "all" ? true : s.status === statusFilter))
      .sort((a, b) => {
        const fa = favoriteSet.has(a.id) ? 0 : 1;
        const fb = favoriteSet.has(b.id) ? 0 : 1;
        if (fa !== fb) return fa - fb;
        return a.name.localeCompare(b.name);
      });
  }, [servers, favoriteSet, search, statusFilter]);

  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [eggId, setEggId] = useState("");
  const [memoryMb, setMemoryMb] = useState(1024);
  const [cpuLimit, setCpuLimit] = useState(100);
  const [diskMb, setDiskMb] = useState(5120);
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
        memory_bytes: memoryMb * bytesPerMB,
        cpu_limit: cpuLimit,
        disk_bytes: diskMb * bytesPerMB,
        variables,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["servers"] });
      queryClient.invalidateQueries({ queryKey: ["quota"] });
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
              min={allowUnlimitedCpu ? 0 : 1}
              step={1}
            />
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 6 }}>
              {allowUnlimitedCpu
                ? "0 = unlimited · 100 = one full core · 200 = two cores"
                : "100 = one full core · 200 = two cores — a CPU limit is required"}
            </p>
          </div>
          <div className="sp-field">
            <label className="sp-label">Disk (MB)</label>
            <input
              className="sp-input"
              type="number"
              value={diskMb}
              onChange={(e) => setDiskMb(Number(e.target.value))}
              min={0}
              step={1}
            />
          </div>

          <div className="sp-field">
            <label className="sp-label">Your quota</label>
            <div className="sp-surface" style={{ padding: 14 }}>
              <QuotaMeters compact />
            </div>
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 6 }}>
              Need more? Buy quota upgrades in the Store with coins earned on the AFK page.
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

      {(servers?.length ?? 0) > 0 && (
        <div style={{ display: "flex", gap: 10, marginBottom: 14, flexWrap: "wrap" }}>
          <input
            className="sp-input"
            style={{ flex: "1 1 220px", maxWidth: 360 }}
            placeholder="Search servers…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-label="Search servers by name"
          />
          <select
            className="sp-select"
            style={{ width: 160 }}
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            aria-label="Filter by status"
          >
            <option value="all">All statuses</option>
            <option value="running">Running</option>
            <option value="offline">Offline</option>
            <option value="installing">Installing</option>
            <option value="errored">Errored</option>
          </select>
        </div>
      )}

      <table className="sp-table">
        <thead>
          <tr>
            <th style={{ width: 36 }}></th>
            <th>Name</th>
            <th>Status</th>
            <th>Port</th>
            <th>Memory</th>
            <th style={{ width: 90, textAlign: "right" }}></th>
          </tr>
        </thead>
        <tbody>
          {sortedServers?.length === 0 && (
            <tr>
              <td colSpan={6} className="sp-mono" style={{ color: "var(--sp-text-muted)", textAlign: "center", padding: "24px 0" }}>
                {servers && servers.length > 0 ? "No servers match your filters." : "No servers yet — create one to get started."}
              </td>
            </tr>
          )}
          {sortedServers?.map((server) => {
            const starred = favoriteSet.has(server.id);
            return (
              <tr key={server.id}>
                <td>
                  <button
                    type="button"
                    className="sp-star"
                    aria-label={starred ? "Unfavorite" : "Favorite"}
                    aria-pressed={starred}
                    title={starred ? "Unfavorite" : "Favorite"}
                    data-on={starred ? "1" : undefined}
                    onClick={() => toggleFavorite.mutate({ id: server.id, on: !starred })}
                  >
                    {starred ? "★" : "☆"}
                  </button>
                </td>
                <td>
                  <Link to={`/servers/${server.id}`}>{server.name}</Link>
                </td>
                <td>
                  <StatusBadge status={server.status} />
                  {server.suspended && (
                    <span className="sp-badge" style={{ marginLeft: 6, color: "#ff9b9b", borderColor: "#ff9b9b" }}>
                      suspended
                    </span>
                  )}
                </td>
                <td className="sp-mono">{server.primary_port}</td>
                <td className="sp-mono">{(server.memory_bytes / 1024 / 1024).toFixed(0)}MB</td>
                <td style={{ textAlign: "right" }}>
                  <button
                    type="button"
                    className="sp-btn sp-btn--ghost sp-btn--sm"
                    title="Clone this server's configuration into a new server"
                    disabled={cloneServer.isPending}
                    onClick={() => {
                      setError(null);
                      cloneServer.mutate(server.id);
                    }}
                  >
                    Clone
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
