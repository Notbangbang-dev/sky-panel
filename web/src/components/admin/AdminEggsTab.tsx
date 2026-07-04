import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi, eggsApi, type EggInput } from "../../lib/endpoints";
import type { Egg, EggVariable } from "../../types/api";

const emptyForm: EggInput = {
  name: "",
  docker_image: "",
  startup: "",
  category: "",
  description: "",
  stop_command: "",
  variables: [],
};

// coerceEggInput narrows an arbitrary parsed JSON value to a safe EggInput,
// keeping only the known fields and dropping anything else — so importing a
// hand-edited or foreign file can't smuggle unexpected keys into the form.
function coerceEggInput(raw: unknown): EggInput {
  const o = (raw ?? {}) as Record<string, unknown>;
  const str = (v: unknown) => (typeof v === "string" ? v : "");
  const vars = Array.isArray(o.variables)
    ? (o.variables as Record<string, unknown>[]).map((v) => ({
        name: str(v?.name),
        env: str(v?.env),
        default: str(v?.default),
        user_editable: v?.user_editable !== false,
      }))
    : [];
  return {
    name: str(o.name),
    docker_image: str(o.docker_image),
    startup: str(o.startup),
    category: str(o.category),
    description: str(o.description),
    stop_command: str(o.stop_command),
    variables: vars,
  };
}

export function AdminEggsTab() {
  const queryClient = useQueryClient();
  const { data: eggs } = useQuery({ queryKey: ["eggs"], queryFn: eggsApi.list });

  const [editingID, setEditingID] = useState<string | null>(null);
  const [form, setForm] = useState<EggInput>(emptyForm);
  const [importError, setImportError] = useState<string | null>(null);
  const [exportError, setExportError] = useState<string | null>(null);

  function loadFromJSON(text: string) {
    setImportError(null);
    let parsed: unknown;
    try {
      parsed = JSON.parse(text);
    } catch {
      setImportError("That doesn't look like valid JSON.");
      return;
    }
    const next = coerceEggInput(parsed);
    if (!next.name || !next.docker_image) {
      setImportError("JSON must include at least a name and docker_image.");
      return;
    }
    // Import always creates a new egg — clear any edit selection and load the
    // parsed definition into the form so the admin can review before saving.
    setEditingID(null);
    setForm(next);
  }

  function importFile(file: File) {
    file
      .text()
      .then(loadFromJSON)
      .catch(() => setImportError("Failed to read that file."));
  }

  async function exportEgg(egg: Egg) {
    setExportError(null);
    try {
      const def = await adminApi.exportEgg(egg.id);
      const blob = new Blob([JSON.stringify(def, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${egg.name.replace(/[^a-z0-9]+/gi, "-").toLowerCase() || "egg"}.egg.json`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setExportError(err instanceof Error ? err.message : "Failed to export egg.");
    }
  }

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["eggs"] });

  const resetForm = () => {
    setEditingID(null);
    setForm(emptyForm);
  };

  const create = useMutation({
    mutationFn: () => adminApi.createEgg(form),
    onSuccess: () => {
      resetForm();
      invalidate();
    },
  });
  const update = useMutation({
    mutationFn: () => adminApi.updateEgg(editingID!, form),
    onSuccess: () => {
      resetForm();
      invalidate();
    },
  });
  const remove = useMutation({
    mutationFn: (id: string) => adminApi.deleteEgg(id),
    onSuccess: (_data, id) => {
      if (id === editingID) resetForm();
      invalidate();
    },
  });

  function startEdit(egg: Egg) {
    setEditingID(egg.id);
    setForm({
      name: egg.name,
      docker_image: egg.docker_image,
      startup: egg.startup,
      category: egg.category,
      description: egg.description,
      stop_command: egg.stop_command,
      variables: egg.variables,
    });
  }

  function updateVariable(index: number, patch: Partial<EggVariable>) {
    setForm((f) => ({
      ...f,
      variables: (f.variables ?? []).map((v, i) => (i === index ? { ...v, ...patch } : v)),
    }));
  }

  function addVariable() {
    setForm((f) => ({
      ...f,
      variables: [...(f.variables ?? []), { name: "", env: "", default: "", user_editable: true }],
    }));
  }

  function removeVariable(index: number) {
    setForm((f) => ({ ...f, variables: (f.variables ?? []).filter((_, i) => i !== index) }));
  }

  const saving = create.isPending || update.isPending;
  const error = create.error ?? update.error;

  return (
    <div>
      <div className="sp-surface sp-card" style={{ marginBottom: 20, maxWidth: 640 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 10 }}>
          <div>
            <p className="sp-label" style={{ marginBottom: 2 }}>
              Import an egg
            </p>
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", margin: 0 }}>
              Load a <code>.egg.json</code> file exported from another install into the form below, then save it.
            </p>
          </div>
          <label className="sp-btn sp-btn--sm" style={{ cursor: "pointer" }}>
            Import JSON…
            <input
              type="file"
              accept="application/json,.json"
              style={{ display: "none" }}
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) importFile(file);
                e.target.value = "";
              }}
            />
          </label>
        </div>
        {importError && <p className="sp-error" style={{ marginBottom: 0 }}>{importError}</p>}
        {exportError && <p className="sp-error" style={{ marginBottom: 0 }}>{exportError}</p>}
      </div>

      <form
        className="sp-surface sp-card"
        style={{ marginBottom: 20, maxWidth: 640 }}
        onSubmit={(e) => {
          e.preventDefault();
          if (editingID) update.mutate();
          else create.mutate();
        }}
      >
        <p className="sp-label" style={{ marginBottom: 12 }}>
          {editingID ? "Edit egg" : "Create egg"}
        </p>

        <div className="sp-field">
          <label className="sp-label">Name</label>
          <input className="sp-input" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} required />
        </div>
        <div style={{ display: "flex", gap: 12 }}>
          <div className="sp-field" style={{ flex: 1 }}>
            <label className="sp-label">Category</label>
            <input className="sp-input" value={form.category} onChange={(e) => setForm((f) => ({ ...f, category: e.target.value }))} />
          </div>
          <div className="sp-field" style={{ flex: 2 }}>
            <label className="sp-label">Docker image</label>
            <input
              className="sp-input sp-mono"
              value={form.docker_image}
              onChange={(e) => setForm((f) => ({ ...f, docker_image: e.target.value }))}
              required
            />
          </div>
        </div>
        <div className="sp-field">
          <label className="sp-label">Description</label>
          <input className="sp-input" value={form.description} onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} />
        </div>
        <div className="sp-field">
          <label className="sp-label">Startup command (leave blank to use the image's own entrypoint)</label>
          <textarea
            className="sp-textarea sp-mono"
            value={form.startup}
            onChange={(e) => setForm((f) => ({ ...f, startup: e.target.value }))}
          />
        </div>
        <div className="sp-field">
          <label className="sp-label">Stop command</label>
          <input
            className="sp-input sp-mono"
            value={form.stop_command}
            onChange={(e) => setForm((f) => ({ ...f, stop_command: e.target.value }))}
          />
        </div>

        <div className="sp-field">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 6 }}>
            <label className="sp-label" style={{ marginBottom: 0 }}>
              Variables
            </label>
            <button className="sp-btn sp-btn--sm" type="button" onClick={addVariable}>
              + Add variable
            </button>
          </div>
          {(form.variables ?? []).map((v, i) => (
            <div key={i} style={{ display: "flex", gap: 6, marginBottom: 6, alignItems: "center" }}>
              <input
                className="sp-input sp-mono"
                placeholder="Label"
                value={v.name}
                onChange={(e) => updateVariable(i, { name: e.target.value })}
                style={{ flex: 1 }}
              />
              <input
                className="sp-input sp-mono"
                placeholder="ENV_NAME"
                value={v.env}
                onChange={(e) => updateVariable(i, { env: e.target.value.toUpperCase() })}
                style={{ flex: 1 }}
              />
              <input
                className="sp-input sp-mono"
                placeholder="Default"
                value={v.default}
                onChange={(e) => updateVariable(i, { default: e.target.value })}
                style={{ flex: 1 }}
              />
              <label className="sp-mono" style={{ display: "flex", gap: 4, alignItems: "center", whiteSpace: "nowrap" }}>
                <input type="checkbox" checked={v.user_editable} onChange={(e) => updateVariable(i, { user_editable: e.target.checked })} />
                editable
              </label>
              <button className="sp-btn sp-btn--sm sp-btn--danger" type="button" onClick={() => removeVariable(i)}>
                ×
              </button>
            </div>
          ))}
          {(form.variables ?? []).length === 0 && (
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
              no variables — the container gets no extra env vars beyond what the image defaults to
            </p>
          )}
        </div>

        {error && <p className="sp-error">{(error as Error).message}</p>}

        <div style={{ display: "flex", gap: 8 }}>
          <button className="sp-btn sp-btn--primary" type="submit" disabled={saving}>
            {editingID ? "Save changes" : "Create egg"}
          </button>
          {editingID && (
            <button className="sp-btn" type="button" onClick={resetForm}>
              Cancel
            </button>
          )}
        </div>
      </form>

      <table className="sp-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Category</th>
            <th>Image</th>
            <th>Variables</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {eggs?.map((egg) => (
            <tr key={egg.id}>
              <td>{egg.name}</td>
              <td className="sp-mono">{egg.category || "—"}</td>
              <td className="sp-mono">{egg.docker_image}</td>
              <td className="sp-mono">{egg.variables.length}</td>
              <td style={{ display: "flex", gap: 6, justifyContent: "flex-end" }}>
                <button className="sp-btn sp-btn--sm" onClick={() => startEdit(egg)}>
                  Edit
                </button>
                <button className="sp-btn sp-btn--sm sp-btn--ghost" onClick={() => exportEgg(egg)}>
                  Export
                </button>
                <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => remove.mutate(egg.id)}>
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
