import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { modrinthApi } from "../../lib/endpoints";
import { ApiError } from "../../lib/api";
import type { ModrinthHit } from "../../types/api";

// Mods go to a loader server (Fabric/Forge/…) mods/ folder; plugins go to a
// Bukkit-family server plugins/ folder. The type also drives the folder we
// install into.
type Kind = "mod" | "plugin";

const LOADERS: Record<Kind, string[]> = {
  mod: ["", "fabric", "forge", "quilt", "neoforge"],
  plugin: ["", "paper", "spigot", "purpur", "bukkit"],
};

export function ModrinthTab({ serverId }: { serverId: string }) {
  const [kind, setKind] = useState<Kind>("mod");
  const [query, setQuery] = useState("");
  const [loader, setLoader] = useState("");
  const [version, setVersion] = useState("");
  const [hits, setHits] = useState<ModrinthHit[]>([]);
  const [installed, setInstalled] = useState<Record<string, string>>({});
  const [pending, setPending] = useState<Record<string, boolean>>({});
  const [error, setError] = useState<string | null>(null);

  const search = useMutation({
    mutationFn: () => modrinthApi.search({ q: query, type: kind, loader: loader || undefined, version: version || undefined, limit: 24 }),
    onSuccess: (res) => {
      setHits(res.hits);
      setError(null);
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : "Search failed"),
  });

  const install = useMutation({
    onMutate: (hit: ModrinthHit) => setPending((p) => ({ ...p, [hit.project_id]: true })),
    onSettled: (_d, _e, hit) =>
      setPending((p) => {
        const n = { ...p };
        delete n[hit.project_id];
        return n;
      }),
    mutationFn: async (hit: ModrinthHit) => {
      const versions = await modrinthApi.versions(hit.slug || hit.project_id, loader || undefined, version || undefined);
      if (!versions.length) throw new ApiError(404, "not_found", "No matching version for this loader/game version");
      const v = versions[0];
      const file = v.files.find((f) => f.primary) ?? v.files[0];
      if (!file) throw new ApiError(404, "not_found", "That version has no downloadable file");
      await modrinthApi.install(serverId, {
        download_url: file.url,
        filename: file.filename,
        folder: kind === "mod" ? "mods" : "plugins",
      });
      return { projectId: hit.project_id, filename: file.filename };
    },
    onSuccess: (r) => setInstalled((prev) => ({ ...prev, [r.projectId]: r.filename })),
    onError: (e) => setError(e instanceof ApiError ? e.message : "Install failed"),
  });

  return (
    <div>
      <div className="sp-surface sp-card" style={{ marginBottom: 14 }}>
        <div style={{ display: "flex", gap: 8, marginBottom: 10, flexWrap: "wrap", alignItems: "center" }}>
          <div style={{ display: "flex", gap: 4 }}>
            {(["mod", "plugin"] as Kind[]).map((k) => (
              <button
                key={k}
                className="sp-btn sp-btn--sm"
                style={k === kind ? { background: "var(--sp-accent)", color: "var(--sp-accent-text)" } : undefined}
                onClick={() => {
                  setKind(k);
                  setLoader("");
                }}
              >
                {k === "mod" ? "Mods" : "Plugins"}
              </button>
            ))}
          </div>
          <select className="sp-select" style={{ width: "auto" }} value={loader} onChange={(e) => setLoader(e.target.value)}>
            {LOADERS[kind].map((l) => (
              <option key={l} value={l}>
                {l ? l : "any loader"}
              </option>
            ))}
          </select>
          <input
            className="sp-input sp-mono"
            style={{ width: 120 }}
            placeholder="MC version"
            value={version}
            onChange={(e) => setVersion(e.target.value)}
          />
        </div>
        <form
          style={{ display: "flex", gap: 8 }}
          onSubmit={(e) => {
            e.preventDefault();
            search.mutate();
          }}
        >
          <input
            className="sp-input"
            placeholder={`Search Modrinth for ${kind === "mod" ? "mods" : "plugins"}…`}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <button className="sp-btn sp-btn--primary" type="submit" disabled={search.isPending}>
            {search.isPending ? "Searching…" : "Search"}
          </button>
        </form>
        {error && <p className="sp-error" style={{ marginBottom: 0 }}>{error}</p>}
      </div>

      <div style={{ display: "grid", gap: 10 }}>
        {hits.map((hit) => (
          <div key={hit.project_id} className="sp-surface" style={{ display: "flex", gap: 12, padding: 12, alignItems: "center" }}>
            {hit.icon_url ? (
              <img src={hit.icon_url} alt="" width={44} height={44} style={{ borderRadius: 8, flex: "0 0 44px", objectFit: "cover" }} />
            ) : (
              <div style={{ width: 44, height: 44, borderRadius: 8, background: "var(--sp-bg-alt)", flex: "0 0 44px" }} />
            )}
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: "flex", gap: 8, alignItems: "baseline" }}>
                <strong style={{ fontSize: 14 }}>{hit.title}</strong>
                <span className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)" }}>
                  by {hit.author} · {hit.downloads.toLocaleString()} dl
                </span>
              </div>
              <p style={{ fontSize: 12.5, color: "var(--sp-text-muted)", margin: "2px 0 0", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {hit.description}
              </p>
            </div>
            {installed[hit.project_id] ? (
              <span className="sp-badge" style={{ color: "var(--sp-accent)", borderColor: "var(--sp-accent)" }}>installed</span>
            ) : (
              <button
                className="sp-btn sp-btn--sm"
                disabled={!!pending[hit.project_id]}
                onClick={() => {
                  setError(null);
                  install.mutate(hit);
                }}
              >
                {pending[hit.project_id] ? "Installing…" : "Install"}
              </button>
            )}
          </div>
        ))}
        {hits.length === 0 && !search.isPending && (
          <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", textAlign: "center", padding: 24 }}>
            Search Modrinth to add {kind === "mod" ? "mods" : "plugins"} to this server. Files install straight into the
            server{"'"}s {kind === "mod" ? "mods/" : "plugins/"} folder.
          </p>
        )}
      </div>
    </div>
  );
}
