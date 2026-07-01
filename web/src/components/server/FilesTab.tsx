import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { filesApi } from "../../lib/endpoints";
import { decodeUtf8Base64, encodeUtf8Base64 } from "../../lib/base64";
import type { FileEntry } from "../../types/api";

function joinPath(dir: string, name: string): string {
  return dir ? `${dir}/${name}` : name;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function FilesTab({ serverId }: { serverId: string }) {
  const queryClient = useQueryClient();
  const [path, setPath] = useState("");
  const [editingPath, setEditingPath] = useState<string | null>(null);
  const [editorContent, setEditorContent] = useState("");
  const [newFolderName, setNewFolderName] = useState<string | null>(null);
  const [renaming, setRenaming] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");

  const listing = useQuery({
    queryKey: ["servers", serverId, "files", path],
    queryFn: () => filesApi.list(serverId, path),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["servers", serverId, "files"] });

  const openFile = useMutation({
    mutationFn: (name: string) => filesApi.read(serverId, joinPath(path, name)),
    onSuccess: (result, name) => {
      setEditingPath(joinPath(path, name));
      setEditorContent(decodeUtf8Base64(result.content_base64));
    },
  });

  const saveFile = useMutation({
    mutationFn: () => filesApi.write(serverId, editingPath!, encodeUtf8Base64(editorContent)),
    onSuccess: () => {
      setEditingPath(null);
      invalidate();
    },
  });

  const createFolder = useMutation({
    mutationFn: () => filesApi.mkdir(serverId, joinPath(path, newFolderName!)),
    onSuccess: () => {
      setNewFolderName(null);
      invalidate();
    },
  });

  const removeEntry = useMutation({
    mutationFn: (name: string) => filesApi.remove(serverId, joinPath(path, name)),
    onSuccess: invalidate,
  });

  const rename = useMutation({
    mutationFn: (name: string) => filesApi.rename(serverId, joinPath(path, name), joinPath(path, renameValue)),
    onSuccess: () => {
      setRenaming(null);
      invalidate();
    },
  });

  const crumbs = path ? path.split("/") : [];

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
        <div className="sp-mono">
          <span style={{ cursor: "pointer" }} onClick={() => setPath("")}>
            /
          </span>
          {crumbs.map((c, i) => (
            <span key={i}>
              <span style={{ cursor: "pointer" }} onClick={() => setPath(crumbs.slice(0, i + 1).join("/"))}>
                {c}
              </span>
              {i < crumbs.length - 1 && "/"}
            </span>
          ))}
        </div>
        <button className="sp-btn sp-btn--sm" onClick={() => setNewFolderName("")}>
          New folder
        </button>
      </div>

      {newFolderName !== null && (
        <form
          className="sp-surface sp-card"
          style={{ marginBottom: 16, maxWidth: 360, display: "flex", gap: 8, alignItems: "flex-end" }}
          onSubmit={(e) => {
            e.preventDefault();
            createFolder.mutate();
          }}
        >
          <div className="sp-field" style={{ marginBottom: 0, flex: 1 }}>
            <label className="sp-label">Folder name</label>
            <input className="sp-input" value={newFolderName} onChange={(e) => setNewFolderName(e.target.value)} autoFocus required />
          </div>
          <button className="sp-btn sp-btn--primary sp-btn--sm" type="submit">
            Create
          </button>
          <button className="sp-btn sp-btn--sm" type="button" onClick={() => setNewFolderName(null)}>
            Cancel
          </button>
        </form>
      )}

      {listing.isError && <p className="sp-mono">failed to load this directory</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Size</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {listing.data?.entries.map((entry: FileEntry) => (
            <tr key={entry.name}>
              <td>
                {renaming === entry.name ? (
                  <input
                    className="sp-input"
                    value={renameValue}
                    onChange={(e) => setRenameValue(e.target.value)}
                    autoFocus
                    onKeyDown={(e) => {
                      if (e.key === "Enter") rename.mutate(entry.name);
                      if (e.key === "Escape") setRenaming(null);
                    }}
                  />
                ) : entry.is_dir ? (
                  <span style={{ cursor: "pointer" }} onClick={() => setPath(joinPath(path, entry.name))}>
                    📁 {entry.name}
                  </span>
                ) : (
                  <span style={{ cursor: "pointer" }} onClick={() => openFile.mutate(entry.name)}>
                    {entry.name}
                  </span>
                )}
              </td>
              <td className="sp-mono">{entry.is_dir ? "—" : formatSize(entry.size_bytes)}</td>
              <td style={{ display: "flex", gap: 6, justifyContent: "flex-end" }}>
                {renaming === entry.name ? (
                  <button className="sp-btn sp-btn--sm sp-btn--primary" onClick={() => rename.mutate(entry.name)}>
                    Save
                  </button>
                ) : (
                  <button
                    className="sp-btn sp-btn--sm"
                    onClick={() => {
                      setRenaming(entry.name);
                      setRenameValue(entry.name);
                    }}
                  >
                    Rename
                  </button>
                )}
                <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => removeEntry.mutate(entry.name)}>
                  Delete
                </button>
              </td>
            </tr>
          ))}
          {listing.data?.entries.length === 0 && (
            <tr>
              <td colSpan={3} className="sp-mono">
                empty directory
              </td>
            </tr>
          )}
        </tbody>
      </table>

      {editingPath !== null && (
        <div className="sp-surface sp-card" style={{ marginTop: 16 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
            <p className="sp-label" style={{ marginBottom: 0 }}>
              {editingPath}
            </p>
            <div style={{ display: "flex", gap: 6 }}>
              <button className="sp-btn sp-btn--sm sp-btn--primary" onClick={() => saveFile.mutate()}>
                Save
              </button>
              <button className="sp-btn sp-btn--sm" onClick={() => setEditingPath(null)}>
                Close
              </button>
            </div>
          </div>
          <textarea
            className="sp-textarea sp-mono"
            style={{ width: "100%", height: 320 }}
            value={editorContent}
            onChange={(e) => setEditorContent(e.target.value)}
          />
        </div>
      )}
    </div>
  );
}
