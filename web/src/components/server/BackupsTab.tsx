import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { backupsApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";
import { useState } from "react";

function formatSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${bytes} B`;
}

function formatWhen(unixSecs: number): string {
  if (!unixSecs) return "—";
  return new Date(unixSecs * 1000).toLocaleString();
}

export function BackupsTab({ serverId }: { serverId: string }) {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const { data, isError } = useQuery({
    queryKey: ["servers", serverId, "backups"],
    queryFn: () => backupsApi.list(serverId),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["servers", serverId, "backups"] });
  const onError = (err: unknown) => setError(err instanceof ApiError ? err.message : "Operation failed");

  const create = useMutation({ mutationFn: () => backupsApi.create(serverId), onSuccess: invalidate, onError });
  const restore = useMutation({
    mutationFn: (filename: string) => backupsApi.restore(serverId, filename),
    onSuccess: () => setError(null),
    onError,
  });
  const remove = useMutation({
    mutationFn: (filename: string) => backupsApi.remove(serverId, filename),
    onSuccess: invalidate,
    onError,
  });

  const backups = data?.backups ?? [];

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <p className="sp-mono" style={{ color: "var(--sp-text-muted)", margin: 0 }}>
          Snapshots of this server's files, compressed on the node.
        </p>
        <button className="sp-btn sp-btn--primary" onClick={() => create.mutate()} disabled={create.isPending}>
          {create.isPending ? "Backing up…" : "+ Back up now"}
        </button>
      </div>

      {error && <p className="sp-error">{error}</p>}
      {isError && <p className="sp-mono">failed to load backups</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Created</th>
            <th>Size</th>
            <th>File</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {backups.map((b) => (
            <tr key={b.filename}>
              <td className="sp-mono">{formatWhen(b.created_at)}</td>
              <td className="sp-mono">{formatSize(b.size_bytes)}</td>
              <td className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
                {b.filename}
              </td>
              <td style={{ display: "flex", gap: 6 }}>
                <button
                  className="sp-btn sp-btn--sm"
                  onClick={() => {
                    if (window.confirm("Restore this backup? Current files will be overwritten.")) {
                      restore.mutate(b.filename);
                    }
                  }}
                  disabled={restore.isPending}
                >
                  Restore
                </button>
                <button
                  className="sp-btn sp-btn--sm sp-btn--danger"
                  onClick={() => remove.mutate(b.filename)}
                  disabled={remove.isPending}
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
          {backups.length === 0 && (
            <tr>
              <td colSpan={4} className="sp-mono">
                no backups yet
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
