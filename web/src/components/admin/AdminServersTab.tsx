import { Fragment, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";
import { StatusBadge } from "../StatusBadge";
import { formatBytes } from "../../lib/format";
import { ServerPortsPanel } from "./ServerPortsPanel";

export function AdminServersTab() {
  const queryClient = useQueryClient();
  const { data: servers } = useQuery({ queryKey: ["admin", "servers"], queryFn: adminApi.listAllServers });
  const { data: users } = useQuery({ queryKey: ["admin", "users"], queryFn: adminApi.listUsers });

  const [transferId, setTransferId] = useState<string | null>(null);
  const [portsId, setPortsId] = useState<string | null>(null);
  const [targetOwner, setTargetOwner] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState("");
  const [note, setNote] = useState<string | null>(null);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin", "servers"] });

  const transfer = useMutation({
    mutationFn: ({ id, ownerId }: { id: string; ownerId: string }) => adminApi.transferServer(id, ownerId),
    onSuccess: () => {
      setTransferId(null);
      setTargetOwner("");
      invalidate();
    },
  });
  const remove = useMutation({
    mutationFn: (id: string) => adminApi.deleteServer(id),
    onSuccess: (_res, id) => {
      // Drop the deleted id from the selection so bulk counts/actions stay accurate.
      setSelected((prev) => {
        if (!prev.has(id)) return prev;
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      invalidate();
    },
  });
  const setSuspended = useMutation({
    mutationFn: ({ id, on }: { id: string; on: boolean }) =>
      on ? adminApi.suspendServer(id) : adminApi.unsuspendServer(id),
    onSuccess: invalidate,
  });
  const purge = useMutation({
    mutationFn: (ids: string[]) => adminApi.purgeServers(ids),
    onSuccess: (res) => {
      setSelected(new Set());
      setNote(`Purged ${res.deleted} server${res.deleted === 1 ? "" : "s"}${res.failed.length ? `, ${res.failed.length} failed` : ""}.`);
      setTimeout(() => setNote(null), 4000);
      invalidate();
    },
    onError: (err) => {
      setNote(`Purge failed: ${err instanceof ApiError ? err.message : "unknown error"}`);
      setTimeout(() => setNote(null), 6000);
    },
  });
  const bulkSuspend = useMutation({
    mutationFn: async ({ ids, on }: { ids: string[]; on: boolean }) => {
      const failed: string[] = [];
      for (const id of ids) {
        try {
          await (on ? adminApi.suspendServer(id) : adminApi.unsuspendServer(id));
        } catch {
          failed.push(id);
        }
      }
      return { total: ids.length, failed, on };
    },
    onSuccess: (res) => {
      setSelected(new Set());
      const ok = res.total - res.failed.length;
      const verb = res.on ? "Suspended" : "Unsuspended";
      setNote(`${verb} ${ok} server${ok === 1 ? "" : "s"}${res.failed.length ? `, ${res.failed.length} failed` : ""}.`);
      setTimeout(() => setNote(null), 4000);
      invalidate();
    },
  });

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase();
    const list = servers ?? [];
    if (!q) return list;
    return list.filter(
      (s) => s.name.toLowerCase().includes(q) || s.owner_username.toLowerCase().includes(q),
    );
  }, [servers, filter]);

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  const allShownSelected = filtered.length > 0 && filtered.every((s) => selected.has(s.id));
  const toggleAll = () =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (allShownSelected) filtered.forEach((s) => next.delete(s.id));
      else filtered.forEach((s) => next.add(s.id));
      return next;
    });

  // Bulk actions only ever touch rows the admin can currently see — a filter
  // change must never leave hidden servers silently queued for a purge.
  const filteredIds = useMemo(() => new Set(filtered.map((s) => s.id)), [filtered]);
  const selectedIds = [...selected].filter((id) => filteredIds.has(id));
  const busy = purge.isPending || bulkSuspend.isPending;

  return (
    <div>
      <div style={{ display: "flex", gap: 8, alignItems: "center", marginBottom: 12, flexWrap: "wrap" }}>
        <input
          className="sp-input"
          placeholder="Filter by server or owner…"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          style={{ maxWidth: 260 }}
        />
        <span style={{ flex: 1 }} />
        {selectedIds.length > 0 && (
          <>
            <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
              {selectedIds.length} selected
            </span>
            <button className="sp-btn sp-btn--sm sp-btn--ghost" disabled={busy} onClick={() => bulkSuspend.mutate({ ids: selectedIds, on: true })}>
              Suspend
            </button>
            <button className="sp-btn sp-btn--sm sp-btn--ghost" disabled={busy} onClick={() => bulkSuspend.mutate({ ids: selectedIds, on: false })}>
              Unsuspend
            </button>
            <button
              className="sp-btn sp-btn--sm sp-btn--danger"
              disabled={busy}
              onClick={() => {
                if (
                  window.confirm(
                    `Purge ${selectedIds.length} server(s)? This permanently deletes the containers and frees their ports. Files are lost. This cannot be undone.`,
                  )
                ) {
                  purge.mutate(selectedIds);
                }
              }}
            >
              {purge.isPending ? "Purging…" : `Purge ${selectedIds.length}`}
            </button>
          </>
        )}
      </div>
      {note && <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-accent)", marginTop: 0 }}>{note}</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th style={{ width: 30 }}>
              <input type="checkbox" checked={allShownSelected} onChange={toggleAll} aria-label="Select all" />
            </th>
            <th>Server</th>
            <th>Owner</th>
            <th>Status</th>
            <th>RAM</th>
            <th style={{ textAlign: "right" }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {filtered.map((s) => (
            <Fragment key={s.id}>
              <tr>
                <td>
                  <input type="checkbox" checked={selected.has(s.id)} onChange={() => toggle(s.id)} aria-label={`Select ${s.name}`} />
                </td>
                <td>
                  <Link to={`/servers/${s.id}`} style={{ color: "var(--sp-text)" }}>
                    {s.name}
                  </Link>
                </td>
                <td className="sp-mono">{s.owner_username}</td>
                <td>
                  <StatusBadge status={s.status} />
                  {s.suspended && (
                    <span className="sp-badge" style={{ marginLeft: 6, color: "#ff9b9b", borderColor: "#ff9b9b" }}>
                      suspended
                    </span>
                  )}
                </td>
                <td className="sp-mono">{formatBytes(s.memory_bytes)}</td>
                <td style={{ textAlign: "right", display: "flex", gap: 6, justifyContent: "flex-end", flexWrap: "wrap" }}>
                  <button
                    className="sp-btn sp-btn--sm sp-btn--ghost"
                    disabled={setSuspended.isPending}
                    onClick={() => setSuspended.mutate({ id: s.id, on: !s.suspended })}
                  >
                    {s.suspended ? "Unsuspend" : "Suspend"}
                  </button>
                  <button
                    className="sp-btn sp-btn--sm"
                    onClick={() => setPortsId((cur) => (cur === s.id ? null : s.id))}
                  >
                    Ports
                  </button>
                  <button
                    className="sp-btn sp-btn--sm"
                    onClick={() => {
                      setTransferId((cur) => (cur === s.id ? null : s.id));
                      setTargetOwner("");
                    }}
                  >
                    Transfer
                  </button>
                  <button
                    className="sp-btn sp-btn--sm sp-btn--danger"
                    disabled={remove.isPending}
                    onClick={() => {
                      if (window.confirm(`Delete “${s.name}” (owner ${s.owner_username})? Container + files are removed. This cannot be undone.`)) {
                        remove.mutate(s.id);
                      }
                    }}
                  >
                    Delete
                  </button>
                </td>
              </tr>
              {transferId === s.id && (
                <tr>
                  <td colSpan={6} style={{ background: "var(--sp-bg-alt)" }}>
                    <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
                      <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
                        Reassign “{s.name}” to
                      </span>
                      <select className="sp-select" value={targetOwner} onChange={(e) => setTargetOwner(e.target.value)} style={{ width: 200 }}>
                        <option value="">Choose a user…</option>
                        {users
                          ?.filter((u) => u.id !== s.owner_id)
                          .map((u) => (
                            <option key={u.id} value={u.id}>
                              {u.username}
                            </option>
                          ))}
                      </select>
                      <button
                        className="sp-btn sp-btn--sm sp-btn--primary"
                        disabled={!targetOwner || transfer.isPending}
                        onClick={() => transfer.mutate({ id: s.id, ownerId: targetOwner })}
                      >
                        Transfer
                      </button>
                    </div>
                  </td>
                </tr>
              )}
              {portsId === s.id && (
                <tr>
                  <td colSpan={6} style={{ background: "var(--sp-bg-alt)" }}>
                    <ServerPortsPanel serverId={s.id} />
                  </td>
                </tr>
              )}
            </Fragment>
          ))}
          {filtered.length === 0 && (
            <tr>
              <td colSpan={6} className="sp-mono">
                no servers
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
