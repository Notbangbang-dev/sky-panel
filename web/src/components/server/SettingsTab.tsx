import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { serversApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";
import type { Server } from "../../types/api";

export function SettingsTab({ server }: { server: Server }) {
  const queryClient = useQueryClient();

  const [name, setName] = useState(server.name);
  const [memoryMb, setMemoryMb] = useState(Math.round(server.memory_bytes / 1024 / 1024));
  const [cpuLimit, setCpuLimit] = useState(server.cpu_limit);
  const [diskMb, setDiskMb] = useState(Math.round(server.disk_bytes / 1024 / 1024));
  const [backupHours, setBackupHours] = useState(server.backup_interval_hours);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const [description, setDescription] = useState(server.description ?? "");
  const [descSaved, setDescSaved] = useState(false);
  const saveDescription = useMutation({
    mutationFn: () => serversApi.setDescription(server.id, description),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["servers", server.id] });
      setDescSaved(true);
      setTimeout(() => setDescSaved(false), 2500);
    },
  });

  const togglePublic = useMutation({
    mutationFn: (isPublic: boolean) => serversApi.setPublicStatus(server.id, isPublic),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["servers", server.id] }),
  });
  const statusUrl = `${window.location.origin}/status/${server.id}`;

  const save = useMutation({
    mutationFn: () =>
      serversApi.update(server.id, {
        name,
        memory_bytes: memoryMb * 1024 * 1024,
        cpu_limit: cpuLimit,
        disk_bytes: diskMb * 1024 * 1024,
        backup_interval_hours: backupHours,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["servers", server.id] });
      queryClient.invalidateQueries({ queryKey: ["quota"] });
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    },
    onError: (err) => setError(err instanceof ApiError ? err.message : "Failed to save settings"),
  });

  return (
    <>
      <div className="sp-surface sp-card" style={{ maxWidth: 480, marginBottom: 16 }}>
        <div className="sp-field" style={{ marginBottom: 8 }}>
          <label className="sp-label">Notes / description</label>
          <textarea
            className="sp-textarea"
            style={{ width: "100%", minHeight: 70 }}
            maxLength={500}
            placeholder="What's this server for? (only you and admins see this)"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
          <button className="sp-btn sp-btn--sm sp-btn--primary" onClick={() => saveDescription.mutate()} disabled={saveDescription.isPending}>
            Save notes
          </button>
          {descSaved && <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-accent)" }}>Saved — no restart needed.</span>}
        </div>
      </div>

      <div className="sp-surface sp-card" style={{ maxWidth: 480, marginBottom: 16 }}>
        <label className="sp-mono" style={{ display: "flex", gap: 8, alignItems: "center", fontSize: 13 }}>
          <input
            type="checkbox"
            checked={server.public_status}
            disabled={togglePublic.isPending}
            onChange={(e) => togglePublic.mutate(e.target.checked)}
          />
          Public status page — a shareable link showing this server's online status, players and version
        </label>
        {server.public_status && (
          <p className="sp-mono" style={{ fontSize: 12, marginTop: 8, marginBottom: 0 }}>
            <a href={statusUrl} target="_blank" rel="noreferrer" style={{ color: "var(--sp-accent)" }}>
              {statusUrl}
            </a>
          </p>
        )}
      </div>

      <form
        className="sp-surface sp-card"
        style={{ maxWidth: 480 }}
        onSubmit={(e) => {
          e.preventDefault();
          setError(null);
          setSaved(false);
          save.mutate();
        }}
      >
      <div className="sp-field">
        <label className="sp-label">Name</label>
        <input className="sp-input" value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div className="sp-field">
        <label className="sp-label">Memory (MB)</label>
        <input
          className="sp-input"
          type="number"
          value={memoryMb}
          onChange={(e) => setMemoryMb(Number(e.target.value))}
          min={128}
          step={1}
        />
      </div>
      <div className="sp-field">
        <label className="sp-label">CPU limit (% of one core)</label>
        <input
          className="sp-input"
          type="number"
          value={cpuLimit}
          onChange={(e) => setCpuLimit(Number(e.target.value))}
          min={0}
          step={1}
        />
        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 6 }}>
          0 = unlimited · 100 = one full core · 200 = two cores
        </p>
      </div>
      <div className="sp-field">
        <label className="sp-label">Disk (MB)</label>
        <input
          className="sp-input"
          type="number"
          value={diskMb}
          onChange={(e) => setDiskMb(Number(e.target.value))}
          min={0}
          step={1}
        />
      </div>
      <div className="sp-field">
        <label className="sp-label">Automatic backups every (hours)</label>
        <input
          className="sp-input"
          type="number"
          value={backupHours}
          onChange={(e) => setBackupHours(Number(e.target.value))}
          min={0}
          step={1}
        />
        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 6 }}>
          0 = disabled. The panel snapshots this server on the interval you set.
        </p>
      </div>

      <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginBottom: 12 }}>
        Saving re-provisions the container with the new limits. Your files and world data are preserved.
      </p>

      {error && <p className="sp-error">{error}</p>}
      {saved && (
        <p className="sp-mono" style={{ color: "var(--sp-accent)", marginBottom: 10 }}>
          Settings saved.
        </p>
      )}
      <button className="sp-btn sp-btn--primary" type="submit" disabled={save.isPending}>
        {save.isPending ? "Saving…" : "Save changes"}
      </button>
      </form>
    </>
  );
}
