import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { eggsApi, serversApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";
import { toast } from "../../lib/toast";
import type { Server } from "../../types/api";

// Mirror of panel-api's serversvc.substitute: replace every {{TOKEN}} using
// values, leaving unknown tokens intact so they stay visible instead of
// silently collapsing to an empty string.
function renderStartup(template: string, values: Record<string, string>): string {
  return template.replace(/\{\{\s*([A-Za-z0-9_]+)\s*\}\}/g, (match, token: string) =>
    Object.prototype.hasOwnProperty.call(values, token) ? values[token] : match,
  );
}

// StartupTab is the cloud-panel-style startup editor: it lets a server owner
// change the egg's user-editable variables (e.g. the Node entry file, the
// Minecraft version) and previews the exact command that will run, then
// re-provisions. Non-editable variables are fixed by the egg and shown for
// context only.
export function StartupTab({ server }: { server: Server }) {
  const queryClient = useQueryClient();
  const { data: eggs, isLoading } = useQuery({ queryKey: ["eggs"], queryFn: eggsApi.list });
  const egg = useMemo(() => eggs?.find((e) => e.id === server.egg_id), [eggs, server.egg_id]);

  // Only changed fields live here; the effective value falls back to the
  // server's saved override and then the egg default.
  const [overrides, setOverrides] = useState<Record<string, string>>({});
  const effective = (env: string, def: string) => overrides[env] ?? server.variables?.[env] ?? def;

  const editableVars = egg?.variables.filter((v) => v.user_editable) ?? [];
  const fixedVars = egg?.variables.filter((v) => !v.user_editable) ?? [];
  const dirty = Object.keys(overrides).length > 0;

  // Resolve every variable the way the backend will, for an accurate preview.
  const resolved = useMemo(() => {
    const map: Record<string, string> = {};
    for (const v of egg?.variables ?? []) {
      map[v.env] = v.user_editable ? effective(v.env, v.default) : v.default;
    }
    return map;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [egg, overrides, server.variables]);

  const resolvedCmd = egg
    ? egg.startup
      ? renderStartup(egg.startup, resolved)
      : "(the image's own default entrypoint runs — this egg sets no startup command)"
    : "";

  const save = useMutation({
    mutationFn: () => {
      const variables: Record<string, string> = {};
      for (const v of editableVars) variables[v.env] = effective(v.env, v.default);
      return serversApi.update(server.id, {
        name: server.name,
        memory_bytes: server.memory_bytes,
        cpu_limit: server.cpu_limit,
        disk_bytes: server.disk_bytes,
        backup_interval_hours: server.backup_interval_hours,
        variables,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["servers", server.id] });
      setOverrides({});
      toast.success(
        server.status === "running"
          ? "Startup saved — re-provisioning and restarting the server."
          : "Startup saved — the container will be re-provisioned.",
      );
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : "Failed to save startup"),
  });

  if (isLoading) return <p className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>Loading startup…</p>;
  if (!egg)
    return (
      <p className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
        This server's egg is unavailable, so its startup can't be edited here.
      </p>
    );

  return (
    <div style={{ maxWidth: 560 }}>
      <div className="sp-surface sp-card" style={{ marginBottom: 16 }}>
        <label className="sp-label">Startup command</label>
        <pre
          className="sp-mono"
          style={{
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
            background: "var(--sp-bg)",
            border: "1px solid var(--sp-surface-border)",
            borderRadius: 6,
            padding: "10px 12px",
            fontSize: 12.5,
            margin: "6px 0 0",
          }}
        >
          {resolvedCmd}
        </pre>
        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 8, marginBottom: 0 }}>
          Docker image: <span style={{ color: "var(--sp-text)" }}>{egg.docker_image}</span> · Software:{" "}
          <span style={{ color: "var(--sp-text)" }}>{egg.name}</span>
        </p>
      </div>

      <form
        className="sp-surface sp-card"
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate();
        }}
      >
        {editableVars.length === 0 && (
          <p className="sp-mono" style={{ fontSize: 13, color: "var(--sp-text-muted)", marginTop: 0 }}>
            This egg has no editable startup variables.
          </p>
        )}
        {editableVars.map((v) => (
          <div className="sp-field" key={v.env}>
            <label className="sp-label">{v.name}</label>
            <input
              className="sp-input sp-mono"
              value={effective(v.env, v.default)}
              onChange={(e) => setOverrides((prev) => ({ ...prev, [v.env]: e.target.value }))}
              placeholder={v.default}
            />
            <p className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", marginTop: 4 }}>
              {v.env} · default <code>{v.default || "(empty)"}</code>
            </p>
          </div>
        ))}

        {fixedVars.length > 0 && (
          <div className="sp-field">
            <label className="sp-label" style={{ color: "var(--sp-text-muted)" }}>
              Fixed by this egg
            </label>
            <div className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
              {fixedVars.map((v) => (
                <div key={v.env}>
                  {v.env} = <span style={{ color: "var(--sp-text)" }}>{v.default}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginBottom: 12 }}>
          Saving re-provisions the container with the new startup. Your files and world data are preserved; a running
          server restarts to apply the change.
        </p>
        <button className="sp-btn sp-btn--primary" type="submit" disabled={save.isPending || editableVars.length === 0 || !dirty}>
          {save.isPending ? "Saving…" : "Save & apply"}
        </button>
      </form>
    </div>
  );
}
