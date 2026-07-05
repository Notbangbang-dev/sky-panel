import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { databasesApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";
import type { Database } from "../../types/api";

export function DatabasesTab({ serverId }: { serverId: string }) {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["databases", serverId],
    queryFn: () => databasesApi.list(serverId),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["databases", serverId] });
    queryClient.invalidateQueries({ queryKey: ["quota"] });
  };

  const create = useMutation({
    mutationFn: () => databasesApi.create(serverId, name.trim()),
    onSuccess: () => {
      setName("");
      setError(null);
      invalidate();
    },
    onError: (err) => setError(err instanceof ApiError ? err.message : "Failed to create database"),
  });

  const remove = useMutation({
    mutationFn: (id: string) => databasesApi.remove(serverId, id),
    onSuccess: invalidate,
  });

  const databases = data?.databases ?? [];

  return (
    <div>
      <form
        className="sp-surface sp-card"
        style={{ maxWidth: 520, marginBottom: 16 }}
        onSubmit={(e) => {
          e.preventDefault();
          if (name.trim()) create.mutate();
        }}
      >
        <label className="sp-label">New database</label>
        <div style={{ display: "flex", gap: 8 }}>
          <input
            className="sp-input"
            style={{ flex: 1 }}
            placeholder="e.g. survival, economy…"
            value={name}
            maxLength={32}
            onChange={(e) => setName(e.target.value)}
          />
          <button className="sp-btn sp-btn--primary" type="submit" disabled={create.isPending || !name.trim()}>
            {create.isPending ? "Creating…" : "Create"}
          </button>
        </div>
        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 8, marginBottom: 0 }}>
          A MariaDB database + user is created on this server's node. You get the full credentials below. Database slots
          come from your quota — grab more in the Store.
        </p>
        {error && <p className="sp-error" style={{ marginBottom: 0, marginTop: 10 }}>{error}</p>}
      </form>

      {isLoading ? (
        <p className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>Loading…</p>
      ) : databases.length === 0 ? (
        <div className="sp-surface sp-card" style={{ textAlign: "center", padding: "32px 20px" }}>
          <p className="sp-mono" style={{ color: "var(--sp-text-muted)", margin: 0 }}>
            No databases yet. Create one above.
          </p>
        </div>
      ) : (
        <div style={{ display: "grid", gap: 12 }}>
          {databases.map((db) => (
            <DatabaseCard key={db.id} db={db} onDelete={() => remove.mutate(db.id)} deleting={remove.isPending} />
          ))}
        </div>
      )}
    </div>
  );
}

function DatabaseCard({ db, onDelete, deleting }: { db: Database; onDelete: () => void; deleting: boolean }) {
  const [reveal, setReveal] = useState(false);
  const jdbc = `jdbc:mysql://${db.host}:${db.port}/${db.name}`;

  return (
    <div className="sp-surface sp-card">
      <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
        <h3 className="sp-mono" style={{ margin: 0, fontSize: 16, flex: 1 }}>{db.name}</h3>
        <button
          className="sp-btn sp-btn--sm sp-btn--danger"
          disabled={deleting}
          onClick={() => {
            if (window.confirm(`Delete database "${db.name}"? This permanently drops it and all its data.`)) onDelete();
          }}
        >
          Delete
        </button>
      </div>
      <div style={{ display: "grid", gap: 6 }}>
        <Field label="Host" value={db.host} />
        <Field label="Port" value={String(db.port)} />
        <Field label="Database" value={db.name} />
        <Field label="Username" value={db.username} />
        <Field label="Password" value={reveal ? db.password : "••••••••••••"} copyValue={db.password}>
          <button className="sp-btn sp-btn--sm sp-btn--ghost" onClick={() => setReveal((r) => !r)}>
            {reveal ? "Hide" : "Reveal"}
          </button>
        </Field>
        <Field label="JDBC" value={jdbc} />
      </div>
    </div>
  );
}

function Field({
  label,
  value,
  copyValue,
  children,
}: {
  label: string;
  value: string;
  copyValue?: string;
  children?: React.ReactNode;
}) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(copyValue ?? value).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    });
  };
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
      <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", width: 78, flex: "0 0 78px" }}>
        {label}
      </span>
      <code
        style={{
          flex: 1,
          fontSize: 13,
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
          background: "var(--sp-surface-2, rgba(255,255,255,0.04))",
          padding: "4px 8px",
          borderRadius: 4,
        }}
      >
        {value}
      </code>
      {children}
      <button className="sp-btn sp-btn--sm sp-btn--ghost" onClick={copy} title={`Copy ${label.toLowerCase()}`}>
        {copied ? "Copied" : "Copy"}
      </button>
    </div>
  );
}
