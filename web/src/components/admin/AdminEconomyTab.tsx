import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";

// Economy knobs, keyed to the settings the backend (coinsvc) reads at runtime.
// The `def` column is what the panel falls back to when a key is unset.
const FIELDS: { key: string; label: string; hint: string; def: number }[] = [
  { key: "afk.coins_per_heartbeat", label: "AFK coins per tick", hint: "coins granted each earning tick", def: 1 },
  { key: "afk.min_interval_seconds", label: "AFK min interval (s)", hint: "ticks faster than this earn nothing", def: 20 },
  { key: "afk.max_interval_seconds", label: "AFK max interval (s)", hint: "a longer gap ends the session", def: 90 },
  { key: "daily_reward.amount", label: "Daily reward (coins)", hint: "coins from a daily claim", def: 100 },
  { key: "daily_reward.interval_hours", label: "Daily reward cooldown (h)", hint: "hours between daily claims", def: 24 },
];

export function AdminEconomyTab() {
  const queryClient = useQueryClient();
  const { data: settings } = useQuery({ queryKey: ["admin", "settings"], queryFn: adminApi.getSettings });

  const [values, setValues] = useState<Record<string, string>>({});
  const [savedKey, setSavedKey] = useState<string | null>(null);

  // Seed the inputs from the stored settings (or the defaults) once loaded.
  useEffect(() => {
    if (!settings) return;
    const seeded: Record<string, string> = {};
    for (const f of FIELDS) seeded[f.key] = settings[f.key] ?? String(f.def);
    setValues(seeded);
  }, [settings]);

  const save = useMutation({
    mutationFn: ({ key, value }: { key: string; value: string }) => adminApi.setSetting(key, value),
    onSuccess: (_data, { key }) => {
      setSavedKey(key);
      setTimeout(() => setSavedKey(null), 2000);
      queryClient.invalidateQueries({ queryKey: ["admin", "settings"] });
    },
  });

  return (
    <div style={{ maxWidth: 560 }}>
      <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginBottom: 16 }}>
        Tune how the AFK page and daily reward pay out. Changes take effect immediately for new ticks and claims.
      </p>

      {FIELDS.map((f) => (
        <div key={f.key} className="sp-surface sp-card" style={{ marginBottom: 12, padding: 16 }}>
          <label className="sp-label">{f.label}</label>
          <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", margin: "4px 0 10px" }}>
            {f.hint} · default {f.def}
          </p>
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <input
              className="sp-input sp-mono"
              style={{ width: 140 }}
              type="number"
              min={1}
              value={values[f.key] ?? ""}
              onChange={(e) => setValues((v) => ({ ...v, [f.key]: e.target.value }))}
            />
            <button
              className="sp-btn sp-btn--sm sp-btn--primary"
              disabled={save.isPending}
              onClick={() => save.mutate({ key: f.key, value: values[f.key] ?? String(f.def) })}
            >
              Save
            </button>
            {savedKey === f.key && (
              <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-accent)" }}>
                saved
              </span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
