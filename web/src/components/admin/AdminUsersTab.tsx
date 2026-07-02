import { Fragment, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";
import { bytesPerMB } from "../../lib/format";

interface QuotaDraft {
  memoryMb: string;
  cpuPercent: string;
  diskMb: string;
}

const EMPTY_DRAFT: QuotaDraft = { memoryMb: "", cpuPercent: "", diskMb: "" };

export function AdminUsersTab() {
  const queryClient = useQueryClient();
  const { data: users } = useQuery({ queryKey: ["admin", "users"], queryFn: adminApi.listUsers });
  const [adjustAmount, setAdjustAmount] = useState<Record<string, string>>({});
  const [quotaOpen, setQuotaOpen] = useState<string | null>(null);
  const [quotaDraft, setQuotaDraft] = useState<QuotaDraft>(EMPTY_DRAFT);
  const [quotaMsg, setQuotaMsg] = useState<string | null>(null);

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
  const setQuota = useMutation({
    mutationFn: ({ id, draft }: { id: string; draft: QuotaDraft }) =>
      adminApi.setUserQuota(id, {
        memory_bytes: Number(draft.memoryMb || 0) * bytesPerMB,
        cpu_percent: Number(draft.cpuPercent || 0),
        disk_bytes: Number(draft.diskMb || 0) * bytesPerMB,
      }),
    onSuccess: () => setQuotaMsg("Bonus quota saved."),
    onError: () => setQuotaMsg("Failed to save quota."),
  });

  function toggleQuota(id: string) {
    setQuotaMsg(null);
    setQuotaDraft(EMPTY_DRAFT);
    setQuotaOpen((cur) => (cur === id ? null : id));
  }

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
          <Fragment key={u.id}>
            <tr>
              <td>{u.username}</td>
              <td className="sp-mono">{u.email}</td>
              <td>{u.role}</td>
              <td className="sp-mono">{u.coins}</td>
              <td>
                <div style={{ display: "flex", gap: 6, alignItems: "center", flexWrap: "wrap" }}>
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
                  <button className="sp-btn sp-btn--sm" onClick={() => toggleQuota(u.id)}>
                    Quota
                  </button>
                  <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => deleteUser.mutate(u.id)}>
                    Delete
                  </button>
                </div>
              </td>
            </tr>
            {quotaOpen === u.id && (
              <tr>
                <td colSpan={5} style={{ background: "var(--sp-bg-alt)" }}>
                  <div style={{ display: "flex", gap: 10, alignItems: "flex-end", flexWrap: "wrap" }}>
                    <QuotaInput
                      label="Bonus RAM (MB)"
                      value={quotaDraft.memoryMb}
                      onChange={(v) => setQuotaDraft((d) => ({ ...d, memoryMb: v }))}
                    />
                    <QuotaInput
                      label="Bonus CPU (%)"
                      value={quotaDraft.cpuPercent}
                      onChange={(v) => setQuotaDraft((d) => ({ ...d, cpuPercent: v }))}
                    />
                    <QuotaInput
                      label="Bonus disk (MB)"
                      value={quotaDraft.diskMb}
                      onChange={(v) => setQuotaDraft((d) => ({ ...d, diskMb: v }))}
                    />
                    <button className="sp-btn sp-btn--sm sp-btn--primary" onClick={() => setQuota.mutate({ id: u.id, draft: quotaDraft })}>
                      Set bonus
                    </button>
                    <span className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)" }}>
                      Absolute bonus on top of the global default quota. {quotaMsg}
                    </span>
                  </div>
                </td>
              </tr>
            )}
          </Fragment>
        ))}
      </tbody>
    </table>
  );
}

function QuotaInput({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", display: "block", marginBottom: 4 }}>
        {label}
      </label>
      <input
        className="sp-input sp-mono"
        style={{ width: 120 }}
        type="number"
        min={0}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}
