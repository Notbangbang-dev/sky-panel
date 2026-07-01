import { describe, expect, it, beforeEach } from "vitest";
import {
  applyThemeToDocument,
  DEFAULT_THEME,
  loadCustomThemes,
  PRESET_THEMES,
  saveCustomThemes,
  type Theme,
} from "./theme";

describe("applyThemeToDocument", () => {
  it("writes every theme color/radius onto :root as CSS variables", () => {
    applyThemeToDocument(DEFAULT_THEME);

    const root = document.documentElement;
    expect(root.style.getPropertyValue("--sp-bg")).toBe(DEFAULT_THEME.background);
    expect(root.style.getPropertyValue("--sp-surface")).toBe(DEFAULT_THEME.surface);
    expect(root.style.getPropertyValue("--sp-text")).toBe(DEFAULT_THEME.text);
    expect(root.style.getPropertyValue("--sp-accent")).toBe(DEFAULT_THEME.accent);
    expect(root.style.getPropertyValue("--sp-radius")).toBe(DEFAULT_THEME.radius);
  });

  it("sets data-theme and data-animation-intensity for the animated background to read", () => {
    const signal = PRESET_THEMES.find((t) => t.id === "signal")!;
    applyThemeToDocument(signal);

    expect(document.documentElement.dataset.theme).toBe("signal");
    expect(document.documentElement.dataset.animationIntensity).toBe(signal.animationIntensity);
  });

  it("re-theming overwrites previous values rather than leaving stale ones", () => {
    applyThemeToDocument(PRESET_THEMES[0]);
    applyThemeToDocument(PRESET_THEMES[1]);

    expect(document.documentElement.style.getPropertyValue("--sp-accent")).toBe(PRESET_THEMES[1].accent);
  });
});

describe("custom theme persistence", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("round-trips through localStorage", () => {
    const custom: Theme = { ...DEFAULT_THEME, id: "custom-1", name: "My Theme", builtin: false };
    saveCustomThemes([custom]);

    const loaded = loadCustomThemes();
    expect(loaded).toHaveLength(1);
    expect(loaded[0]).toEqual(custom);
  });

  it("returns an empty array when nothing has been saved", () => {
    expect(loadCustomThemes()).toEqual([]);
  });

  it("returns an empty array if the stored value is corrupted JSON", () => {
    localStorage.setItem("sky-panel:custom-themes", "{not valid json");
    expect(loadCustomThemes()).toEqual([]);
  });
});
