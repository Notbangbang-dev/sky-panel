import { Fragment, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";
import { StatusBadge } from "../StatusBadge";
import { formatBytes } from "../../lib/format";

export function AdminServersTab() {
  const queryClient = useQueryClient();
  const { data: servers } = useQuery({ queryKey: ["admin", "servers"], queryFn: adminApi.listAllServers });
  const { data: users } = useQuery({ queryKey: ["admin", "users"], queryFn: adminApi.listUsers });

  const [transferId, setTransferId] = useState<string | null>(null);
  const [targetOwner, setTargetOwner] = useState("");

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin", "servers"] });
  const transfer = useMutation({
    mutationFn: ({ id, ownerId }: { id: string; ownerId: string }) => adminApi.transferServer(id, ownerId),
    onSuccess: () => {
      setTransferId(null);
      setTargetOwner("");
      invalidate();
    },
  });

  return (
    <table className="sp-table">
      <thead>
        <tr>
          <th>Server</th>
          <th>Owner</th>
          <th>Status</th>
          <th>RAM</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {servers?.map((s) => (
          <Fragment key={s.id}>
            <tr>
              <td>
                <Link to={`/servers/${s.id}`} style={{ color: "var(--sp-text)" }}>
                  {s.name}
                </Link>
              </td>
              <td className="sp-mono">{s.owner_username}</td>
              <td>
                <StatusBadge status={s.status} />
              </td>
              <td className="sp-mono">{formatBytes(s.memory_bytes)}</td>
              <td>
                <button
                  className="sp-btn sp-btn--sm"
                  onClick={() => {
                    setTransferId((cur) => (cur === s.id ? null : s.id));
                    setTargetOwner("");
                  }}
                >
                  Transfer
                </button>
              </td>
            </tr>
            {transferId === s.id && (
              <tr>
                <td colSpan={5} style={{ background: "var(--sp-bg-alt)" }}>
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
          </Fragment>
        ))}
        {servers?.length === 0 && (
          <tr>
            <td colSpan={5} className="sp-mono">
              no servers yet
            </td>
          </tr>
        )}
      </tbody>
    </table>
  );
}
