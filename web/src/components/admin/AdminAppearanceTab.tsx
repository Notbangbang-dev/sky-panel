import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";
import { PRESET_THEMES, applyThemeToDocument, findPreset } from "../../lib/theme";
import {
  DEFAULT_BACKGROUND,
  parseBackground,
  type BackgroundConfig,
  type BackgroundMode,
} from "../../lib/appearance";
import { useAppearance } from "../../lib/AppearanceProvider";

const BG_MODES: { value: BackgroundMode; label: string }[] = [
  { value: "animated", label: "Animated mesh" },
  { value: "gradient", label: "Gradient" },
  { value: "solid", label: "Solid color" },
  { value: "image", label: "Image URL" },
  { value: "video", label: "Video URL" },
];

export function AdminAppearanceTab() {
  const queryClient = useQueryClient();
  const { reload } = useAppearance();
  const { data: settings } = useQuery({ queryKey: ["admin-settings"], queryFn: adminApi.getSettings });

  const [themePreset, setThemePreset] = useState("");
  const [customCss, setCustomCss] = useState("");
  const [bg, setBg] = useState<BackgroundConfig>(DEFAULT_BACKGROUND);
  const [maintEnabled, setMaintEnabled] = useState(false);
  const [maintMessage, setMaintMessage] = useState("");
  const [savedNote, setSavedNote] = useState<string | null>(null);

  // Hydrate the form once settings load.
  useEffect(() => {
    if (!settings) return;
    setThemePreset(settings["appearance.theme_preset"] ?? "");
    setCustomCss(settings["appearance.custom_css"] ?? "");
    setBg(parseBackground(settings["appearance.background"] ?? ""));
    setMaintEnabled(settings["maintenance.enabled"] === "true");
    setMaintMessage(settings["maintenance.message"] ?? "");
  }, [settings]);

  const save = useMutation({
    mutationFn: async () => {
      await adminApi.setSetting("appearance.theme_preset", themePreset);
      await adminApi.setSetting("appearance.custom_css", customCss);
      await adminApi.setSetting("appearance.background", JSON.stringify(bg));
      await adminApi.setSetting("maintenance.enabled", maintEnabled ? "true" : "false");
      await adminApi.setSetting("maintenance.message", maintMessage);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-settings"] });
      reload();
      setSavedNote("Saved — appearance applied instance-wide.");
      setTimeout(() => setSavedNote(null), 2500);
    },
  });

  // Live-preview a preset click for this admin immediately.
  function pickPreset(id: string) {
    setThemePreset(id);
    const preset = findPreset(id);
    if (preset) applyThemeToDocument(preset);
  }

  const patchBg = (patch: Partial<BackgroundConfig>) => setBg((b) => ({ ...b, ...patch }));

  return (
    <div style={{ maxWidth: 720 }}>
      <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 0 }}>
        These control the look of the whole instance — every user, and the login page. Individual users can still pick
        their own theme in Account → Theme.
      </p>

      {/* Preset themes */}
      <section className="sp-surface sp-card" style={{ marginBottom: 16 }}>
        <p className="sp-label">Preset theme</p>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(150px, 1fr))", gap: 10 }}>
          <ThemeSwatch id="" name="No default" active={themePreset === ""} onClick={() => pickPreset("")} />
          {PRESET_THEMES.map((t) => (
            <ThemeSwatch key={t.id} id={t.id} name={t.name} active={themePreset === t.id} onClick={() => pickPreset(t.id)} />
          ))}
        </div>
      </section>

      {/* Background */}
      <section className="sp-surface sp-card" style={{ marginBottom: 16 }}>
        <p className="sp-label">Background</p>
        <div className="sp-field">
          <label className="sp-label">Mode</label>
          <select className="sp-select" value={bg.mode} onChange={(e) => patchBg({ mode: e.target.value as BackgroundMode })}>
            {BG_MODES.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </div>

        {bg.mode === "solid" && (
          <div className="sp-field">
            <label className="sp-label">Color</label>
            <input className="sp-input sp-mono" placeholder="#08090b" value={bg.color} onChange={(e) => patchBg({ color: e.target.value })} />
          </div>
        )}
        {bg.mode === "gradient" && (
          <div className="sp-field">
            <label className="sp-label">CSS gradient</label>
            <input
              className="sp-input sp-mono"
              placeholder="linear-gradient(135deg, #08090b, #14121c)"
              value={bg.gradient}
              onChange={(e) => patchBg({ gradient: e.target.value })}
            />
          </div>
        )}
        {(bg.mode === "image" || bg.mode === "video") && (
          <>
            <div className="sp-field">
              <label className="sp-label">{bg.mode === "image" ? "Image URL" : "Video URL (mp4/webm)"}</label>
              <input
                className="sp-input sp-mono"
                placeholder="https://…"
                value={bg.mode === "image" ? bg.imageUrl : bg.videoUrl}
                onChange={(e) => patchBg(bg.mode === "image" ? { imageUrl: e.target.value } : { videoUrl: e.target.value })}
              />
            </div>
            <div style={{ display: "flex", gap: 16 }}>
              <div className="sp-field" style={{ flex: 1 }}>
                <label className="sp-label">Blur: {bg.blur}px</label>
                <input type="range" min={0} max={24} value={bg.blur} onChange={(e) => patchBg({ blur: Number(e.target.value) })} style={{ width: "100%" }} />
              </div>
              <div className="sp-field" style={{ flex: 1 }}>
                <label className="sp-label">Dim: {Math.round(bg.dim * 100)}%</label>
                <input type="range" min={0} max={100} value={Math.round(bg.dim * 100)} onChange={(e) => patchBg({ dim: Number(e.target.value) / 100 })} style={{ width: "100%" }} />
              </div>
            </div>
          </>
        )}
      </section>

      {/* Custom CSS */}
      <section className="sp-surface sp-card" style={{ marginBottom: 16 }}>
        <p className="sp-label">Custom CSS</p>
        <p className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", marginTop: 0 }}>
          Injected into every page — style any class freely. To override a theme <code>--sp-*</code> token, add{" "}
          <code>!important</code> (the active theme sets them inline). Applies after save.
        </p>
        <textarea
          className="sp-textarea sp-mono"
          rows={7}
          placeholder=":root { --sp-accent: #ff4d6d; }"
          value={customCss}
          onChange={(e) => setCustomCss(e.target.value)}
        />
      </section>

      {/* Maintenance */}
      <section className="sp-surface sp-card" style={{ marginBottom: 16 }}>
        <p className="sp-label">Maintenance mode</p>
        <label className="sp-mono" style={{ display: "flex", gap: 8, alignItems: "center", fontSize: 13 }}>
          <input type="checkbox" checked={maintEnabled} onChange={(e) => setMaintEnabled(e.target.checked)} />
          Take the panel offline for everyone except admins
        </label>
        <div className="sp-field" style={{ marginTop: 10 }}>
          <label className="sp-label">Message shown to users</label>
          <input
            className="sp-input"
            placeholder="Back shortly — upgrading the fleet."
            value={maintMessage}
            onChange={(e) => setMaintMessage(e.target.value)}
          />
        </div>
      </section>

      <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
        <button className="sp-btn sp-btn--primary" onClick={() => save.mutate()} disabled={save.isPending}>
          {save.isPending ? "Saving…" : "Save appearance"}
        </button>
        {savedNote && <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-accent)" }}>{savedNote}</span>}
        {save.isError && <span className="sp-error" style={{ margin: 0 }}>Failed to save.</span>}
      </div>
    </div>
  );
}

function ThemeSwatch({ id, name, active, onClick }: { id: string; name: string; active: boolean; onClick: () => void }) {
  const preset = findPreset(id);
  return (
    <button
      type="button"
      onClick={onClick}
      className="sp-surface"
      style={{
        padding: 10,
        cursor: "pointer",
        textAlign: "left",
        borderColor: active ? "var(--sp-accent)" : "var(--sp-surface-border)",
        boxShadow: active ? "0 0 0 1px var(--sp-accent)" : undefined,
      }}
    >
      <div style={{ display: "flex", gap: 5, marginBottom: 8 }}>
        {preset ? (
          [preset.background, preset.surface, preset.accent].map((c, i) => (
            <span key={i} style={{ width: 20, height: 20, borderRadius: 5, background: c, border: "1px solid var(--sp-surface-border)" }} />
          ))
        ) : (
          <span style={{ width: 20, height: 20, borderRadius: 5, border: "1px dashed var(--sp-surface-border)" }} />
        )}
      </div>
      <span className="sp-mono" style={{ fontSize: 12 }}>{name}</span>
    </button>
  );
}
