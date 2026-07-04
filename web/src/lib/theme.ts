export type AnimationIntensity = "off" | "subtle" | "normal" | "high";

export interface Theme {
  id: string;
  name: string;
  background: string;
  backgroundAlt: string;
  surface: string;
  surfaceBorder: string;
  text: string;
  textMuted: string;
  accent: string;
  accentText: string;
  radius: string;
  animationIntensity: AnimationIntensity;
  builtin?: boolean;
}

export const PRESET_THEMES: Theme[] = [
  {
    id: "monochrome",
    name: "Monochrome",
    background: "#08090b",
    backgroundAlt: "#0e1013",
    surface: "#131417",
    surfaceBorder: "#26282e",
    text: "#f2f2f0",
    textMuted: "#8b8d94",
    accent: "#f2f2f0",
    accentText: "#08090b",
    radius: "10px",
    animationIntensity: "normal",
    builtin: true,
  },
  {
    id: "signal",
    name: "Signal",
    background: "#08090b",
    backgroundAlt: "#0e1013",
    surface: "#131417",
    surfaceBorder: "#262b1f",
    text: "#f2f2f0",
    textMuted: "#8b8d94",
    accent: "#c8ff3d",
    accentText: "#08090b",
    radius: "10px",
    animationIntensity: "normal",
    builtin: true,
  },
  {
    id: "nebula",
    name: "Nebula",
    background: "#0a0812",
    backgroundAlt: "#100c1c",
    surface: "#16112459",
    surfaceBorder: "#2c2148",
    text: "#efeaff",
    textMuted: "#9a90bd",
    accent: "#a985ff",
    accentText: "#0a0812",
    radius: "12px",
    animationIntensity: "high",
    builtin: true,
  },
  {
    id: "ember",
    name: "Ember",
    background: "#0f0906",
    backgroundAlt: "#160d08",
    surface: "#1b1109",
    surfaceBorder: "#3a2213",
    text: "#fbeee3",
    textMuted: "#b39684",
    accent: "#ff7a3d",
    accentText: "#0f0906",
    radius: "8px",
    animationIntensity: "normal",
    builtin: true,
  },
  {
    id: "arctic",
    name: "Arctic",
    background: "#060b0e",
    backgroundAlt: "#0a1216",
    surface: "#0e191f",
    surfaceBorder: "#1c333d",
    text: "#e8f6fb",
    textMuted: "#84a3ae",
    accent: "#4fd6ff",
    accentText: "#060b0e",
    radius: "10px",
    animationIntensity: "normal",
    builtin: true,
  },
  {
    id: "void",
    name: "Void",
    background: "#000000",
    backgroundAlt: "#060606",
    surface: "#0c0c0c",
    surfaceBorder: "#1e1e1e",
    text: "#ffffff",
    textMuted: "#7d7d7d",
    accent: "#ffffff",
    accentText: "#000000",
    radius: "2px",
    animationIntensity: "high",
    builtin: true,
  },
  {
    id: "paper",
    name: "Paper",
    background: "#eeece6",
    backgroundAlt: "#e4e1d8",
    surface: "#f7f5ef",
    surfaceBorder: "#cfcabd",
    text: "#1a1917",
    textMuted: "#6a675e",
    accent: "#1a1917",
    accentText: "#f7f5ef",
    radius: "6px",
    animationIntensity: "subtle",
    builtin: true,
  },
  {
    id: "sakura",
    name: "Sakura",
    background: "#0d0a0c",
    backgroundAlt: "#140f12",
    surface: "#1a1317",
    surfaceBorder: "#38222e",
    text: "#fbe9f1",
    textMuted: "#b58fa2",
    accent: "#ff8fc7",
    accentText: "#0d0a0c",
    radius: "14px",
    animationIntensity: "normal",
    builtin: true,
  },
];

export const DEFAULT_THEME = PRESET_THEMES[0];

const CSS_VAR_MAP: Record<keyof Omit<Theme, "id" | "name" | "animationIntensity" | "builtin">, string> = {
  background: "--sp-bg",
  backgroundAlt: "--sp-bg-alt",
  surface: "--sp-surface",
  surfaceBorder: "--sp-surface-border",
  text: "--sp-text",
  textMuted: "--sp-text-muted",
  accent: "--sp-accent",
  accentText: "--sp-accent-text",
  radius: "--sp-radius",
};

export function applyThemeToDocument(theme: Theme) {
  const root = document.documentElement;
  for (const [key, cssVar] of Object.entries(CSS_VAR_MAP) as [keyof typeof CSS_VAR_MAP, string][]) {
    root.style.setProperty(cssVar, theme[key] as string);
  }
  root.dataset.animationIntensity = theme.animationIntensity;
  root.dataset.theme = theme.id;
}

export function findPreset(id: string): Theme | undefined {
  return PRESET_THEMES.find((t) => t.id === id);
}

const CUSTOM_THEMES_KEY = "sky-panel:custom-themes";
const ACTIVE_THEME_KEY = "sky-panel:active-theme";

export function loadCustomThemes(): Theme[] {
  try {
    const raw = localStorage.getItem(CUSTOM_THEMES_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export function saveCustomThemes(themes: Theme[]) {
  localStorage.setItem(CUSTOM_THEMES_KEY, JSON.stringify(themes));
}

export function loadActiveThemeId(): string | null {
  return localStorage.getItem(ACTIVE_THEME_KEY);
}

export function saveActiveThemeId(id: string) {
  localStorage.setItem(ACTIVE_THEME_KEY, id);
}
