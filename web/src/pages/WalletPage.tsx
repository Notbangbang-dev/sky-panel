import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { coinsApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { ApiError } from "../lib/api";

export function WalletPage() {
  const queryClient = useQueryClient();
  const updateUser = useAuthStore((s) => s.updateUser);
  const { data: wallet, isLoading } = useQuery({ queryKey: ["wallet"], queryFn: coinsApi.wallet });

  const [giftTo, setGiftTo] = useState("");
  const [giftAmount, setGiftAmount] = useState("");
  const [code, setCode] = useState("");
  const [note, setNote] = useState<{ text: string; ok: boolean } | null>(null);

  const afterBalanceChange = (balance: number) => {
    const cur = useAuthStore.getState().user;
    if (cur) updateUser({ ...cur, coins: balance });
    queryClient.invalidateQueries({ queryKey: ["wallet"] });
  };

  const gift = useMutation({
    mutationFn: () => coinsApi.gift(giftTo.trim(), Number(giftAmount)),
    onSuccess: (res) => {
      afterBalanceChange(res.balance);
      setGiftTo("");
      setGiftAmount("");
      setNote({ text: "Coins sent.", ok: true });
    },
    onError: (err) =>
      setNote({
        text:
          err instanceof ApiError && err.code === "not_found"
            ? "No user with that username."
            : err instanceof ApiError && err.code === "insufficient_balance"
              ? "You don't have enough coins."
              : err instanceof ApiError
                ? err.message
                : "Failed to send coins.",
        ok: false,
      }),
  });

  const redeem = useMutation({
    mutationFn: () => coinsApi.redeem(code.trim()),
    onSuccess: (res) => {
      afterBalanceChange(res.balance);
      setCode("");
      setNote({ text: `Redeemed — +${res.credited} coins.`, ok: true });
    },
    onError: (err) =>
      setNote({
        text: err instanceof ApiError ? err.message : "Failed to redeem code.",
        ok: false,
      }),
  });

  return (
    <div>
      <h1 className="sp-page-title">Wallet</h1>

      <div style={{ display: "flex", gap: 16, flexWrap: "wrap", marginBottom: 20 }}>
        <div className="sp-surface sp-card" style={{ width: 260 }}>
          <p className="sp-label">Balance</p>
          <p style={{ fontFamily: "var(--sp-font-display)", fontSize: 40 }}>{wallet?.balance.toLocaleString() ?? "—"}</p>
        </div>

        <div className="sp-surface sp-card" style={{ flex: 1, minWidth: 280 }}>
          <h2 style={{ fontSize: 15, margin: "0 0 10px" }}>Send coins</h2>
          <form
            style={{ display: "flex", gap: 8, alignItems: "flex-end", flexWrap: "wrap" }}
            onSubmit={(e) => {
              e.preventDefault();
              setNote(null);
              if (giftTo.trim() && Number(giftAmount) > 0) gift.mutate();
            }}
          >
            <div className="sp-field" style={{ margin: 0, flex: 1, minWidth: 140 }}>
              <label className="sp-label">To (username)</label>
              <input className="sp-input" value={giftTo} onChange={(e) => setGiftTo(e.target.value)} />
            </div>
            <div className="sp-field" style={{ margin: 0 }}>
              <label className="sp-label">Amount</label>
              <input className="sp-input sp-mono" type="number" min={1} style={{ width: 110 }} value={giftAmount} onChange={(e) => setGiftAmount(e.target.value)} />
            </div>
            <button className="sp-btn sp-btn--primary sp-btn--sm" type="submit" disabled={gift.isPending}>
              Send
            </button>
          </form>
        </div>

        <div className="sp-surface sp-card" style={{ flex: 1, minWidth: 280 }}>
          <h2 style={{ fontSize: 15, margin: "0 0 10px" }}>Redeem a code</h2>
          <form
            style={{ display: "flex", gap: 8, alignItems: "flex-end", flexWrap: "wrap" }}
            onSubmit={(e) => {
              e.preventDefault();
              setNote(null);
              if (code.trim()) redeem.mutate();
            }}
          >
            <div className="sp-field" style={{ margin: 0, flex: 1, minWidth: 160 }}>
              <label className="sp-label">Code</label>
              <input className="sp-input sp-mono" value={code} onChange={(e) => setCode(e.target.value)} />
            </div>
            <button className="sp-btn sp-btn--primary sp-btn--sm" type="submit" disabled={redeem.isPending}>
              Redeem
            </button>
          </form>
        </div>
      </div>

      {note && (
        <p className="sp-mono" style={{ fontSize: 13, marginBottom: 14, color: note.ok ? "var(--sp-accent)" : "#ff9b9b" }}>
          {note.text}
        </p>
      )}

      <h2 style={{ fontSize: 16, marginBottom: 10 }}>History</h2>
      {isLoading && <p className="sp-mono">loading…</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Amount</th>
            <th>Reason</th>
            <th>Note</th>
            <th>When</th>
          </tr>
        </thead>
        <tbody>
          {wallet?.history.map((entry, i) => (
            <tr key={i}>
              <td className="sp-mono" style={{ color: entry.amount >= 0 ? "var(--sp-accent)" : "#ff9b9b" }}>
                {entry.amount >= 0 ? "+" : ""}
                {entry.amount}
              </td>
              <td>{entry.reason.replace(/_/g, " ")}</td>
              <td className="sp-mono">{entry.metadata}</td>
              <td className="sp-mono">{new Date(entry.created_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
