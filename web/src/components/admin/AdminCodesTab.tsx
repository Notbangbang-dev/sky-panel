import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";

export function AdminCodesTab() {
  const queryClient = useQueryClient();
  const { data: codes } = useQuery({ queryKey: ["admin", "codes"], queryFn: adminApi.listRedeemCodes });

  const [code, setCode] = useState("");
  const [coins, setCoins] = useState("");
  const [maxUses, setMaxUses] = useState("0");
  const [error, setError] = useState<string | null>(null);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin", "codes"] });

  const create = useMutation({
    mutationFn: () => adminApi.createRedeemCode(code.trim(), Number(coins), Number(maxUses || 0)),
    onSuccess: () => {
      setCode("");
      setCoins("");
      setMaxUses("0");
      setError(null);
      invalidate();
    },
    onError: (err) =>
      setError(err instanceof ApiError && err.code === "already_exists" ? "That code already exists." : "Failed to create code."),
  });
  const remove = useMutation({ mutationFn: (id: string) => adminApi.deleteRedeemCode(id), onSuccess: invalidate });

  const canCreate = code.trim() !== "" && Number(coins) > 0 && !create.isPending;

  return (
    <div>
      <div className="sp-surface sp-card" style={{ marginBottom: 18 }}>
        <h2 style={{ fontSize: 16, margin: "0 0 4px" }}>Mint a redeem code</h2>
        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", margin: "0 0 14px" }}>
          Users enter a code on their Wallet to claim coins. Each user can redeem a given code once; max uses of 0 means
          unlimited total redemptions.
        </p>
        <div style={{ display: "flex", gap: 8, alignItems: "flex-end", flexWrap: "wrap" }}>
          <div className="sp-field" style={{ margin: 0 }}>
            <label className="sp-label">Code</label>
            <input className="sp-input sp-mono" placeholder="SUMMER2026" value={code} onChange={(e) => setCode(e.target.value)} />
          </div>
          <div className="sp-field" style={{ margin: 0 }}>
            <label className="sp-label">Coins</label>
            <input className="sp-input sp-mono" type="number" min={1} style={{ width: 110 }} value={coins} onChange={(e) => setCoins(e.target.value)} />
          </div>
          <div className="sp-field" style={{ margin: 0 }}>
            <label className="sp-label">Max uses (0 = ∞)</label>
            <input className="sp-input sp-mono" type="number" min={0} style={{ width: 120 }} value={maxUses} onChange={(e) => setMaxUses(e.target.value)} />
          </div>
          <button className="sp-btn sp-btn--primary sp-btn--sm" disabled={!canCreate} onClick={() => create.mutate()}>
            {create.isPending ? "Creating…" : "Create code"}
          </button>
        </div>
        {error && <p className="sp-error" style={{ marginTop: 10 }}>{error}</p>}
      </div>

      <table className="sp-table">
        <thead>
          <tr>
            <th>Code</th>
            <th>Coins</th>
            <th>Uses</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {codes?.map((c) => (
            <tr key={c.id}>
              <td className="sp-mono">{c.code}</td>
              <td className="sp-mono">{c.coins.toLocaleString()}</td>
              <td className="sp-mono">
                {c.uses}
                {c.max_uses > 0 ? ` / ${c.max_uses}` : " / ∞"}
              </td>
              <td style={{ textAlign: "right" }}>
                <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => remove.mutate(c.id)} disabled={remove.isPending}>
                  Delete
                </button>
              </td>
            </tr>
          ))}
          {codes?.length === 0 && (
            <tr>
              <td colSpan={4} className="sp-mono">
                no codes yet
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
