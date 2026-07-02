import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { filesApi } from "../../lib/endpoints";
import { bytesToBase64, base64ToBytes, decodeUtf8Base64, encodeUtf8Base64 } from "../../lib/base64";
import type { FileEntry } from "../../types/api";

// The command channel caps files at 10 MB (see sky-daemon MAX_FILE_BYTES); the
// file manager is for config/plugin-sized files, not bulk transfer.
const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

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
  const [creating, setCreating] = useState<null | "file" | "folder">(null);
  const [createName, setCreateName] = useState("");
  const [renaming, setRenaming] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [notice, setNotice] = useState<{ text: string; ok: boolean } | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);

  // webkitdirectory/directory aren't in the React input types, so set them on
  // the DOM node directly to turn the second picker into a folder picker.
  useEffect(() => {
    const el = folderInputRef.current;
    if (el) {
      el.setAttribute("webkitdirectory", "");
      el.setAttribute("directory", "");
    }
  }, []);

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

  async function create() {
    const name = createName.trim();
    if (!name) return;
    try {
      if (creating === "folder") await filesApi.mkdir(serverId, joinPath(path, name));
      else await filesApi.write(serverId, joinPath(path, name), ""); // empty file
      setCreating(null);
      setCreateName("");
      invalidate();
    } catch {
      setNotice({ text: `Could not create ${name}.`, ok: false });
    }
  }

  // Upload a set of files, each written at `relOf(file)` under the current dir.
  // For a folder upload that's the webkitRelativePath, so the daemon recreates
  // the tree (write_file creates parent dirs). Sequential to stay gentle.
  async function upload(files: File[], relOf: (f: File) => string) {
    setNotice(null);
    const skipped: string[] = [];
    const failed: string[] = [];
    const eligible = files.filter((f) => {
      if (f.size > MAX_UPLOAD_BYTES) {
        skipped.push(f.name);
        return false;
      }
      return true;
    });

    for (let i = 0; i < eligible.length; i++) {
      const f = eligible[i];
      setBusy(`Uploading ${i + 1}/${eligible.length}…`);
      try {
        const bytes = new Uint8Array(await f.arrayBuffer());
        await filesApi.write(serverId, joinPath(path, relOf(f)), bytesToBase64(bytes));
      } catch {
        failed.push(relOf(f));
      }
    }

    setBusy(null);
    invalidate();
    const parts: string[] = [];
    if (eligible.length - failed.length > 0) parts.push(`Uploaded ${eligible.length - failed.length} file(s).`);
    if (skipped.length) parts.push(`Skipped (over 10 MB): ${skipped.join(", ")}.`);
    if (failed.length) parts.push(`Failed: ${failed.join(", ")}.`);
    if (parts.length) setNotice({ text: parts.join(" "), ok: failed.length === 0 && skipped.length === 0 });
  }

  async function download(name: string) {
    try {
      const res = await filesApi.read(serverId, joinPath(path, name));
      const blob = new Blob([base64ToBytes(res.content_base64) as unknown as BlobPart]);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = name;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch {
      setNotice({ text: `Could not download ${name} (files over 10 MB can't be downloaded here).`, ok: false });
    }
  }

  const crumbs = path ? path.split("/") : [];

  return (
    <div>
      {/* ---- Toolbar ---- */}
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 12, marginBottom: 12, flexWrap: "wrap" }}>
        <div className="sp-mono" style={{ fontSize: 13 }}>
          <span style={{ cursor: "pointer", color: "var(--sp-text-muted)" }} onClick={() => setPath("")}>
            home
          </span>
          {crumbs.map((c, i) => (
            <span key={i}>
              <span style={{ color: "var(--sp-text-muted)" }}> / </span>
              <span style={{ cursor: "pointer" }} onClick={() => setPath(crumbs.slice(0, i + 1).join("/"))}>
                {c}
              </span>
            </span>
          ))}
        </div>
        <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
          <button className="sp-btn sp-btn--sm sp-btn--primary" onClick={() => fileInputRef.current?.click()} disabled={!!busy}>
            ↑ Upload
          </button>
          <button className="sp-btn sp-btn--sm" onClick={() => folderInputRef.current?.click()} disabled={!!busy}>
            ↑ Folder
          </button>
          <button className="sp-btn sp-btn--sm" onClick={() => { setCreating("file"); setCreateName(""); }}>
            + File
          </button>
          <button className="sp-btn sp-btn--sm" onClick={() => { setCreating("folder"); setCreateName(""); }}>
            + Folder
          </button>
        </div>
      </div>

      {/* Hidden native pickers driven by the toolbar buttons. */}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        style={{ display: "none" }}
        onChange={(e) => {
          const files = Array.from(e.target.files ?? []);
          e.target.value = "";
          if (files.length) upload(files, (f) => f.name);
        }}
      />
      <input
        ref={folderInputRef}
        type="file"
        multiple
        style={{ display: "none" }}
        onChange={(e) => {
          const files = Array.from(e.target.files ?? []);
          e.target.value = "";
          if (files.length) upload(files, (f) => f.webkitRelativePath || f.name);
        }}
      />

      {creating && (
        <form
          className="sp-surface sp-card"
          style={{ marginBottom: 16, maxWidth: 380, display: "flex", gap: 8, alignItems: "flex-end" }}
          onSubmit={(e) => {
            e.preventDefault();
            create();
          }}
        >
          <div className="sp-field" style={{ marginBottom: 0, flex: 1 }}>
            <label className="sp-label">New {creating} name</label>
            <input className="sp-input" value={createName} onChange={(e) => setCreateName(e.target.value)} autoFocus required />
          </div>
          <button className="sp-btn sp-btn--primary sp-btn--sm" type="submit">
            Create
          </button>
          <button className="sp-btn sp-btn--sm" type="button" onClick={() => setCreating(null)}>
            Cancel
          </button>
        </form>
      )}

      {busy && <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-accent)", marginBottom: 10 }}>{busy}</p>}
      {notice && (
        <p className="sp-mono" style={{ fontSize: 12, marginBottom: 10, color: notice.ok ? "var(--sp-accent)" : "#ff9b9b" }}>
          {notice.text}
        </p>
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
                    📄 {entry.name}
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
                  <>
                    {!entry.is_dir && (
                      <>
                        <button className="sp-btn sp-btn--sm" onClick={() => openFile.mutate(entry.name)}>
                          Edit
                        </button>
                        <button className="sp-btn sp-btn--sm" onClick={() => download(entry.name)}>
                          Download
                        </button>
                      </>
                    )}
                    <button
                      className="sp-btn sp-btn--sm"
                      onClick={() => {
                        setRenaming(entry.name);
                        setRenameValue(entry.name);
                      }}
                    >
                      Rename
                    </button>
                  </>
                )}
                <button
                  className="sp-btn sp-btn--sm sp-btn--danger"
                  onClick={() => {
                    if (window.confirm(`Delete “${entry.name}”${entry.is_dir ? " and everything in it" : ""}?`)) {
                      removeEntry.mutate(entry.name);
                    }
                  }}
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
          {listing.data?.entries.length === 0 && (
            <tr>
              <td colSpan={3} className="sp-mono">
                empty directory — upload or create something
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
