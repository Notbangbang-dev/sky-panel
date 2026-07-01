import { useEffect, useState } from "react";
import { useTheme } from "../lib/ThemeProvider";
import type { AnimationIntensity, Theme } from "../lib/theme";
import { applyThemeToDocument } from "../lib/theme";

export function ThemeBuilderPage() {
  const { theme, themes, setThemeId, saveCustomTheme, deleteCustomTheme } = useTheme();
  const [draft, setDraft] = useState<Theme>(theme);
  const [saved, setSaved] = useState(false);

  // Editing here live-previews onto the real document; if the user
  // navigates away without saving, put the actual active theme back.
  useEffect(() => {
    return () => {
      if (!saved) applyThemeToDocument(theme);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function update<K extends keyof Theme>(key: K, value: Theme[K]) {
    const next = { ...draft, [key]: value };
    setDraft(next);
    applyThemeToDocument(next); // live preview
  }

  function save() {
    const id = draft.builtin ? `custom-${Date.now()}` : draft.id;
    saveCustomTheme({ ...draft, id, builtin: false, name: draft.name || "My theme" });
    setSaved(true);
  }

  return (
    <div>
      <h1 className="sp-page-title">Theme builder</h1>

      <div style={{ display: "grid", gridTemplateColumns: "320px 1fr", gap: 20 }}>
        <div className="sp-surface sp-card">
          <div className="sp-field">
            <label className="sp-label">Base preset</label>
            <select
              className="sp-select"
              onChange={(e) => {
                const base = themes.find((t) => t.id === e.target.value);
                if (base) {
                  setDraft(base);
                  applyThemeToDocument(base);
                }
              }}
            >
              {themes.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>

          <div className="sp-field">
            <label className="sp-label">Name</label>
            <input className="sp-input" value={draft.name} onChange={(e) => update("name", e.target.value)} />
          </div>

          <ColorField label="Background" value={draft.background} onChange={(v) => update("background", v)} />
          <ColorField label="Surface" value={draft.surface} onChange={(v) => update("surface", v)} />
          <ColorField label="Text" value={draft.text} onChange={(v) => update("text", v)} />
          <ColorField label="Accent" value={draft.accent} onChange={(v) => update("accent", v)} />

          <div className="sp-field">
            <label className="sp-label">Corner radius: {draft.radius}</label>
            <input
              type="range"
              min={0}
              max={24}
              value={parseInt(draft.radius)}
              onChange={(e) => update("radius", `${e.target.value}px`)}
              style={{ width: "100%" }}
            />
          </div>

          <div className="sp-field">
            <label className="sp-label">Background motion</label>
            <select
              className="sp-select"
              value={draft.animationIntensity}
              onChange={(e) => update("animationIntensity", e.target.value as AnimationIntensity)}
            >
              <option value="off">Off</option>
              <option value="subtle">Subtle</option>
              <option value="normal">Normal</option>
              <option value="high">High</option>
            </select>
          </div>

          <button className="sp-btn sp-btn--primary" onClick={save} style={{ width: "100%" }}>
            Save theme
          </button>
        </div>

        <div>
          <p className="sp-label" style={{ marginBottom: 10 }}>
            Your themes
          </p>
          <div className="sp-grid">
            {themes.map((t) => (
              <div key={t.id} className="sp-surface sp-card" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                  <span style={{ width: 18, height: 18, borderRadius: "50%", background: t.accent, display: "inline-block" }} />
                  {t.name}
                </div>
                <div style={{ display: "flex", gap: 6 }}>
                  <button
                    className="sp-btn sp-btn--sm"
                    onClick={() => {
                      setThemeId(t.id);
                      setSaved(true);
                    }}
                  >
                    Use
                  </button>
                  {!t.builtin && (
                    <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => deleteCustomTheme(t.id)}>
                      Delete
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function ColorField({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div className="sp-field">
      <label className="sp-label">{label}</label>
      <div style={{ display: "flex", gap: 8 }}>
        <input type="color" value={value} onChange={(e) => onChange(e.target.value)} style={{ width: 40, height: 34, border: "none", background: "none" }} />
        <input className="sp-input sp-mono" value={value} onChange={(e) => onChange(e.target.value)} />
      </div>
    </div>
  );
}
