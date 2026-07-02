import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { schedulesApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";
import type { Schedule, ScheduleAction } from "../../types/api";

const ACTIONS: { value: ScheduleAction; label: string }[] = [
  { value: "start", label: "Start server" },
  { value: "stop", label: "Stop server" },
  { value: "restart", label: "Restart server" },
  { value: "kill", label: "Kill server" },
  { value: "backup", label: "Take a backup" },
  { value: "command", label: "Run console command" },
];

const ACTION_LABEL: Record<string, string> = Object.fromEntries(ACTIONS.map((a) => [a.value, a.label]));

// A few friendly presets so people don't have to think in raw minutes.
const INTERVAL_PRESETS: { label: string; minutes: number }[] = [
  { label: "Every 30 min", minutes: 30 },
  { label: "Hourly", minutes: 60 },
  { label: "Every 6 hours", minutes: 360 },
  { label: "Every 12 hours", minutes: 720 },
  { label: "Daily", minutes: 1440 },
];

function formatInterval(minutes: number): string {
  if (minutes % 1440 === 0) {
    const d = minutes / 1440;
    return d === 1 ? "every day" : `every ${d} days`;
  }
  if (minutes % 60 === 0) {
    const h = minutes / 60;
    return h === 1 ? "every hour" : `every ${h} hours`;
  }
  return `every ${minutes} min`;
}

function formatWhen(iso?: string): string {
  if (!iso) return "never";
  const t = Date.parse(iso);
  return Number.isNaN(t) ? "never" : new Date(t).toLocaleString();
}

export function SchedulesTab({ serverId }: { serverId: string }) {
  const queryClient = useQueryClient();
  const [action, setAction] = useState<ScheduleAction>("restart");
  const [name, setName] = useState("");
  const [payload, setPayload] = useState("");
  const [minutes, setMinutes] = useState(1440);
  const [error, setError] = useState<string | null>(null);

  const { data, isError } = useQuery({
    queryKey: ["servers", serverId, "schedules"],
    queryFn: () => schedulesApi.list(serverId),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["servers", serverId, "schedules"] });
  const onError = (err: unknown) => setError(err instanceof ApiError ? err.message : "Operation failed");

  const create = useMutation({
    mutationFn: () =>
      schedulesApi.create(serverId, {
        name: name.trim() || ACTION_LABEL[action],
        action,
        payload: action === "command" ? payload : undefined,
        interval_minutes: minutes,
      }),
    onSuccess: () => {
      setName("");
      setPayload("");
      setError(null);
      invalidate();
    },
    onError,
  });

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => schedulesApi.toggle(serverId, id, enabled),
    onSuccess: invalidate,
    onError,
  });

  const remove = useMutation({
    mutationFn: (id: string) => schedulesApi.remove(serverId, id),
    onSuccess: invalidate,
    onError,
  });

  const schedules = data ?? [];
  const commandMissing = action === "command" && payload.trim() === "";
  const canCreate = minutes >= 1 && !commandMissing && !create.isPending;

  return (
    <div>
      <div className="sp-surface sp-card" style={{ marginBottom: 18 }}>
        <h2 style={{ fontSize: 16, margin: "0 0 4px" }}>New automation</h2>
        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", margin: "0 0 14px" }}>
          The panel runs this action on a fixed interval, on its own — even while you're away.
        </p>

        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", gap: 12 }}>
          <div className="sp-field" style={{ margin: 0 }}>
            <label className="sp-label">Action</label>
            <select className="sp-select" value={action} onChange={(e) => setAction(e.target.value as ScheduleAction)}>
              {ACTIONS.map((a) => (
                <option key={a.value} value={a.value}>
                  {a.label}
                </option>
              ))}
            </select>
          </div>

          <div className="sp-field" style={{ margin: 0 }}>
            <label className="sp-label">Runs</label>
            <select className="sp-select" value={minutes} onChange={(e) => setMinutes(Number(e.target.value))}>
              {INTERVAL_PRESETS.map((p) => (
                <option key={p.minutes} value={p.minutes}>
                  {p.label}
                </option>
              ))}
              {!INTERVAL_PRESETS.some((p) => p.minutes === minutes) && (
                <option value={minutes}>{formatInterval(minutes)}</option>
              )}
            </select>
          </div>

          <div className="sp-field" style={{ margin: 0 }}>
            <label className="sp-label">Label (optional)</label>
            <input
              className="sp-input"
              placeholder={ACTION_LABEL[action]}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
        </div>

        {action === "command" && (
          <div className="sp-field" style={{ marginTop: 12, marginBottom: 0 }}>
            <label className="sp-label">Command to run</label>
            <input
              className="sp-input sp-mono"
              placeholder="say Server restarting in 5 minutes"
              value={payload}
              onChange={(e) => setPayload(e.target.value)}
            />
          </div>
        )}

        {error && <p className="sp-error" style={{ marginTop: 12 }}>{error}</p>}

        <button
          className="sp-btn sp-btn--primary"
          style={{ marginTop: 14 }}
          onClick={() => {
            setError(null);
            create.mutate();
          }}
          disabled={!canCreate}
        >
          {create.isPending ? "Creating…" : "+ Add automation"}
        </button>
      </div>

      {isError && <p className="sp-mono">failed to load schedules</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Automation</th>
            <th>Runs</th>
            <th>Last run</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {schedules.map((s: Schedule) => (
            <tr key={s.id} style={{ opacity: s.enabled ? 1 : 0.55 }}>
              <td>
                <div>{s.name || ACTION_LABEL[s.action] || s.action}</div>
                <div className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)" }}>
                  {ACTION_LABEL[s.action] ?? s.action}
                  {s.action === "command" && s.payload ? `: ${s.payload}` : ""}
                </div>
              </td>
              <td className="sp-mono">{formatInterval(s.interval_minutes)}</td>
              <td className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
                {formatWhen(s.last_run_at)}
              </td>
              <td>
                <span className={s.enabled ? "sp-badge sp-badge--running" : "sp-badge sp-badge--offline"}>
                  {s.enabled ? "active" : "paused"}
                </span>
              </td>
              <td style={{ display: "flex", gap: 6, justifyContent: "flex-end" }}>
                <button
                  className="sp-btn sp-btn--sm"
                  onClick={() => toggle.mutate({ id: s.id, enabled: !s.enabled })}
                  disabled={toggle.isPending}
                >
                  {s.enabled ? "Pause" : "Resume"}
                </button>
                <button
                  className="sp-btn sp-btn--sm sp-btn--danger"
                  onClick={() => remove.mutate(s.id)}
                  disabled={remove.isPending}
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
          {schedules.length === 0 && (
            <tr>
              <td colSpan={5} className="sp-mono">
                no automations yet
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
