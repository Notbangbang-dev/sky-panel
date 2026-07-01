import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";

export function AdminUsersTab() {
  const queryClient = useQueryClient();
  const { data: users } = useQuery({ queryKey: ["admin", "users"], queryFn: adminApi.listUsers });
  const [adjustAmount, setAdjustAmount] = useState<Record<string, string>>({});

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin", "users"] });

  const setRole = useMutation({
    mutationFn: ({ id, role }: { id: string; role: "admin" | "user" }) => adminApi.setUserRole(id, role),
    onSuccess: invalidate,
  });
  const deleteUser = useMutation({ mutationFn: (id: string) => adminApi.deleteUser(id), onSuccess: invalidate });
  const adjust = useMutation({
    mutationFn: ({ id, amount }: { id: string; amount: number }) => adminApi.adjustCoins(id, amount),
    onSuccess: invalidate,
  });

  return (
    <table className="sp-table">
      <thead>
        <tr>
          <th>Username</th>
          <th>Email</th>
          <th>Role</th>
          <th>Coins</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {users?.map((u) => (
          <tr key={u.id}>
            <td>{u.username}</td>
            <td className="sp-mono">{u.email}</td>
            <td>{u.role}</td>
            <td className="sp-mono">{u.coins}</td>
            <td>
              <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                <button
                  className="sp-btn sp-btn--sm"
                  onClick={() => setRole.mutate({ id: u.id, role: u.role === "admin" ? "user" : "admin" })}
                >
                  Make {u.role === "admin" ? "user" : "admin"}
                </button>
                <input
                  className="sp-input sp-mono"
                  style={{ width: 70 }}
                  placeholder="±coins"
                  value={adjustAmount[u.id] ?? ""}
                  onChange={(e) => setAdjustAmount((prev) => ({ ...prev, [u.id]: e.target.value }))}
                />
                <button
                  className="sp-btn sp-btn--sm"
                  onClick={() => {
                    const amount = Number(adjustAmount[u.id]);
                    if (amount) adjust.mutate({ id: u.id, amount });
                  }}
                >
                  Adjust
                </button>
                <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => deleteUser.mutate(u.id)}>
                  Delete
                </button>
              </div>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
