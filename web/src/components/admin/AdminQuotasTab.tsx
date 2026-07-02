import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";
import { bytesPerMB } from "../../lib/format";

// Global default quota (applied to every user on top of their purchased/granted
// bonus) plus the unlimited-CPU switch, all backed by the settings store.
const BYTE_FIELDS = [
  { key: "quota.default_memory_bytes", label: "Default memory (MB)", def: 2048 },
  { key: "quota.default_disk_bytes", label: "Default disk (MB)", def: 10240 },
];
const CPU_FIELD = { key: "quota.default_cpu_percent", label: "Default CPU (% of one core)", def: 200 };

export function AdminQuotasTab() {
  const queryClient = useQueryClient();
  const { data: settings } = useQuery({ queryKey: ["admin", "settings"], queryFn: adminApi.getSettings });

  const [values, setValues] = useState<Record<string, string>>({});
  const [allowUnlimitedCpu, setAllowUnlimitedCpu] = useState(true);
  const [saved, setSaved] = useState<string | null>(null);

  useEffect(() => {
    if (!settings) return;
    const seeded: Record<string, string> = {};
    for (const f of BYTE_FIELDS) {
      seeded[f.key] = settings[f.key] ? String(Math.round(Number(settings[f.key]) / bytesPerMB)) : String(f.def);
    }
    seeded[CPU_FIELD.key] = settings[CPU_FIELD.key] ?? String(CPU_FIELD.def);
    setValues(seeded);
    // Unset defaults to allowed (the backend's fallback), matching existing behavior.
    setAllowUnlimitedCpu((settings["quota.allow_unlimited_cpu"] ?? "true").toLowerCase() !== "false");
  }, [settings]);

  const save = useMutation({
    mutationFn: ({ key, value }: { key: string; value: string }) => adminApi.setSetting(key, value),
    onSuccess: (_d, { key }) => {
      setSaved(key);
      setTimeout(() => setSaved(null), 2000);
      queryClient.invalidateQueries({ queryKey: ["admin", "settings"] });
    },
  });

  const saveMb = (key: string) => save.mutate({ key, value: String(Number(values[key] || 0) * bytesPerMB) });

  return (
    <div style={{ maxWidth: 560 }}>
      <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginBottom: 16 }}>
        Every user's limit is this default plus whatever bonus they've bought or been granted. Turn off unlimited CPU to
        force every server to reserve CPU from the quota, like memory and disk.
      </p>

      <div className="sp-surface sp-card" style={{ marginBottom: 12, padding: 16 }}>
        <label className="sp-label" style={{ display: "flex", alignItems: "center", gap: 10, cursor: "pointer", textTransform: "none" }}>
          <input
            type="checkbox"
            checked={!allowUnlimitedCpu}
            onChange={(e) => {
              const disallow = e.target.checked;
              setAllowUnlimitedCpu(!disallow);
              save.mutate({ key: "quota.allow_unlimited_cpu", value: disallow ? "false" : "true" });
            }}
          />
          Disable unlimited CPU (require a CPU limit within quota)
        </label>
        {saved === "quota.allow_unlimited_cpu" && (
          <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-accent)", marginLeft: 26 }}>
            saved
          </span>
        )}
      </div>

      {[...BYTE_FIELDS, CPU_FIELD].map((f) => {
        const isBytes = f.key !== CPU_FIELD.key;
        return (
          <div key={f.key} className="sp-surface sp-card" style={{ marginBottom: 12, padding: 16 }}>
            <label className="sp-label">{f.label}</label>
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", margin: "4px 0 10px" }}>
              default {f.def}
            </p>
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
              <input
                className="sp-input sp-mono"
                style={{ width: 160 }}
                type="number"
                min={1}
                value={values[f.key] ?? ""}
                onChange={(e) => setValues((v) => ({ ...v, [f.key]: e.target.value }))}
              />
              <button
                className="sp-btn sp-btn--sm sp-btn--primary"
                disabled={save.isPending}
                onClick={() => (isBytes ? saveMb(f.key) : save.mutate({ key: f.key, value: values[f.key] ?? String(f.def) }))}
              >
                Save
              </button>
              {saved === f.key && (
                <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-accent)" }}>
                  saved
                </span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
