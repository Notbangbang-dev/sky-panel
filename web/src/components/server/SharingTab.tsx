import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { subusersApi } from "../../lib/endpoints";
import { PERMISSIONS, type Permission } from "../../types/api";

export function SharingTab({ serverId }: { serverId: string }) {
  const queryClient = useQueryClient();
  const { data: subusers, isError } = useQuery({
    queryKey: ["servers", serverId, "subusers"],
    queryFn: () => subusersApi.list(serverId),
  });

  const [username, setUsername] = useState("");
  const [permissions, setPermissions] = useState<Permission[]>([]);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["servers", serverId, "subusers"] });

  const add = useMutation({
    mutationFn: () => subusersApi.add(serverId, username, permissions),
    onSuccess: () => {
      setUsername("");
      setPermissions([]);
      invalidate();
    },
  });

  const remove = useMutation({ mutationFn: (userId: string) => subusersApi.remove(serverId, userId), onSuccess: invalidate });

  function togglePermission(p: Permission) {
    setPermissions((prev) => (prev.includes(p) ? prev.filter((x) => x !== p) : [...prev, p]));
  }

  return (
    <div>
      <form
        className="sp-surface sp-card"
        style={{ marginBottom: 20, maxWidth: 460 }}
        onSubmit={(e) => {
          e.preventDefault();
          add.mutate();
        }}
      >
        <div className="sp-field">
          <label className="sp-label">Username</label>
          <input className="sp-input" value={username} onChange={(e) => setUsername(e.target.value)} required />
        </div>
        <div className="sp-field">
          <label className="sp-label">Permissions</label>
          <div style={{ display: "flex", gap: 14, flexWrap: "wrap" }}>
            {PERMISSIONS.map((p) => (
              <label key={p} className="sp-mono" style={{ display: "flex", gap: 6, alignItems: "center", cursor: "pointer" }}>
                <input type="checkbox" checked={permissions.includes(p)} onChange={() => togglePermission(p)} />
                {p}
              </label>
            ))}
          </div>
        </div>
        {add.isError && <p className="sp-mono">{(add.error as Error).message}</p>}
        <button className="sp-btn sp-btn--primary" type="submit">
          Grant access
        </button>
      </form>

      {isError && <p className="sp-mono">failed to load subusers</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th>User</th>
            <th>Permissions</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {subusers?.map((s) => (
            <tr key={s.user_id}>
              <td className="sp-mono" title={s.user_id}>
                {s.user_id.slice(0, 8)}…
              </td>
              <td className="sp-mono">{s.permissions.join(", ")}</td>
              <td>
                <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => remove.mutate(s.user_id)}>
                  Revoke
                </button>
              </td>
            </tr>
          ))}
          {subusers?.length === 0 && (
            <tr>
              <td colSpan={3} className="sp-mono">
                no one else has access to this server
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
