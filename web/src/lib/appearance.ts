// Instance-wide appearance, controlled by admins and applied to every client.
// The three pieces — a theme preset id, a custom-CSS blob, and a background
// config — are stored as plain settings on the server and read from the public
// /appearance endpoint (so the login page gets them too).

export type BackgroundMode = "animated" | "solid" | "gradient" | "image" | "video";

export interface BackgroundConfig {
  mode: BackgroundMode;
  color: string; // solid mode
  gradient: string; // gradient mode: a full CSS background value
  imageUrl: string; // image mode
  videoUrl: string; // video mode
  blur: number; // px, applied to image/video
  dim: number; // 0..1 dark overlay over image/video
}

export const DEFAULT_BACKGROUND: BackgroundConfig = {
  mode: "animated",
  color: "",
  gradient: "linear-gradient(135deg, #08090b, #14121c 60%, #08090b)",
  imageUrl: "",
  videoUrl: "",
  blur: 0,
  dim: 0.4,
};

export function parseBackground(json: string): BackgroundConfig {
  if (!json) return DEFAULT_BACKGROUND;
  try {
    const parsed = JSON.parse(json) as Partial<BackgroundConfig>;
    return { ...DEFAULT_BACKGROUND, ...parsed };
  } catch {
    return DEFAULT_BACKGROUND;
  }
}

export interface Appearance {
  themePreset: string;
  customCss: string;
  background: BackgroundConfig;
}

const CUSTOM_CSS_ELEMENT_ID = "sp-admin-css";

// Injects (or updates/removes) the admin custom CSS into a single <style> tag.
export function applyCustomCss(css: string) {
  let el = document.getElementById(CUSTOM_CSS_ELEMENT_ID) as HTMLStyleElement | null;
  if (!css) {
    if (el) el.remove();
    return;
  }
  if (!el) {
    el = document.createElement("style");
    el.id = CUSTOM_CSS_ELEMENT_ID;
    document.head.appendChild(el);
  }
  el.textContent = css;
}
