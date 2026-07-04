import { useQuery } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";

export function AdminAnalyticsTab() {
  const { data, isError } = useQuery({
    queryKey: ["admin", "analytics"],
    queryFn: adminApi.analytics,
    refetchInterval: 30_000,
  });

  if (isError) return <p className="sp-error">failed to load analytics</p>;
  if (!data) return <p className="sp-mono">loading…</p>;

  const stat = (label: string, value: string | number, sub?: string) => (
    <div className="sp-surface sp-card">
      <p className="sp-stat__label">{label}</p>
      <p className="sp-mono" style={{ fontSize: 26, margin: 0, fontVariantNumeric: "tabular-nums" }}>
        {value}
      </p>
      {sub && (
        <p className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", margin: "2px 0 0" }}>
          {sub}
        </p>
      )}
    </div>
  );

  const statusEntries = Object.entries(data.servers_by_status).sort((a, b) => b[1] - a[1]);
  const eggEntries = Object.entries(data.servers_by_egg).sort((a, b) => b[1] - a[1]);

  return (
    <div>
      <div className="sp-grid sp-grid--cards" style={{ marginBottom: 18 }}>
        {stat("Users", data.users, `${data.admins} admin${data.admins === 1 ? "" : "s"}`)}
        {stat("Servers", data.servers, `${data.suspended} suspended`)}
        {stat("Nodes", `${data.nodes_connected}/${data.nodes}`, "connected")}
        {stat("Eggs", data.eggs)}
        {stat("Coins in circulation", data.coins_in_circulation.toLocaleString() + " ⧫")}
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: 16 }}>
        <Breakdown title="Servers by status" entries={statusEntries} total={data.servers} />
        <Breakdown title="Servers by egg" entries={eggEntries} total={data.servers} />
      </div>
    </div>
  );
}

function Breakdown({ title, entries, total }: { title: string; entries: [string, number][]; total: number }) {
  return (
    <div className="sp-surface sp-card">
      <p className="sp-label">{title}</p>
      {entries.length === 0 && <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>no data</p>}
      {entries.map(([label, count]) => {
        const pct = total > 0 ? Math.round((count / total) * 100) : 0;
        return (
          <div key={label} style={{ marginBottom: 8 }}>
            <div style={{ display: "flex", justifyContent: "space-between", fontSize: 12.5, marginBottom: 3 }}>
              <span>{label}</span>
              <span className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
                {count} · {pct}%
              </span>
            </div>
            <div className="sp-gauge">
              <div className="sp-gauge__fill" style={{ width: `${pct}%` }} />
            </div>
          </div>
        );
      })}
    </div>
  );
}
