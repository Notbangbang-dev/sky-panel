import { useQuery } from "@tanstack/react-query";
import { quotaApi } from "../lib/endpoints";
import { formatBytes } from "../lib/format";
import type { QuotaInfo } from "../types/api";

// A trio of usage bars (memory, CPU, disk) driven by GET /me/quota. Reused on
// the servers list, the create form, and the store so the user always sees how
// much of their quota is spoken for.
export function QuotaMeters({ compact = false }: { compact?: boolean }) {
  const { data } = useQuery({ queryKey: ["quota"], queryFn: quotaApi.mine });
  if (!data) return null;
  return <QuotaMetersView quota={data} compact={compact} />;
}

export function QuotaMetersView({ quota, compact = false }: { quota: QuotaInfo; compact?: boolean }) {
  const { usage, limit } = quota;
  return (
    <div style={{ display: "grid", gap: compact ? 10 : 14 }}>
      <Meter label="Memory" used={usage.memory_bytes} total={limit.memory_bytes} render={formatBytes} />
      <Meter label="CPU" used={usage.cpu_percent} total={limit.cpu_percent} render={(v) => `${v}%`} />
      <Meter label="Disk" used={usage.disk_bytes} total={limit.disk_bytes} render={formatBytes} />
    </div>
  );
}

function Meter({
  label,
  used,
  total,
  render,
}: {
  label: string;
  used: number;
  total: number;
  render: (v: number) => string;
}) {
  const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0;
  const over = used > total;
  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 6 }}>
        <span className="sp-stat__label" style={{ margin: 0 }}>
          {label}
        </span>
        <span className="sp-mono" style={{ fontSize: 12, color: over ? "#ff9b9b" : "var(--sp-text-muted)" }}>
          {render(used)} / {render(total)}
        </span>
      </div>
      <div className="sp-meter">
        <div className="sp-meter__fill" style={{ width: `${pct}%`, background: over ? "#ff9b9b" : "var(--sp-accent)" }} />
      </div>
    </div>
  );
}
