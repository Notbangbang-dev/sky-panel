import { useQuery } from "@tanstack/react-query";
import { coinsApi } from "../lib/endpoints";

export function WalletPage() {
  const { data: wallet, isLoading } = useQuery({ queryKey: ["wallet"], queryFn: coinsApi.wallet });

  return (
    <div>
      <h1 className="sp-page-title">Wallet</h1>

      <div className="sp-surface sp-card" style={{ marginBottom: 20, width: 260 }}>
        <p className="sp-label">Balance</p>
        <p style={{ fontFamily: "var(--sp-font-display)", fontSize: 40 }}>{wallet?.balance.toLocaleString() ?? "—"}</p>
      </div>

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
