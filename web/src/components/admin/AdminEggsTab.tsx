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

export function AdminEggsTab() {
  const queryClient = useQueryClient();
  const { data: eggs } = useQuery({ queryKey: ["eggs"], queryFn: eggsApi.list });

  const [editingID, setEditingID] = useState<string | null>(null);
  const [form, setForm] = useState<EggInput>(emptyForm);

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
