"use client";

import { useSyncExternalStore } from "react";
import { applyTheme, THEME_CHANGE_EVENT, type Theme } from "@/lib/theme";

function subscribe(callback: () => void) {
  window.addEventListener(THEME_CHANGE_EVENT, callback);
  return () => window.removeEventListener(THEME_CHANGE_EVENT, callback);
}

// The no-flash script in the root layout sets this before hydration, so
// reading it directly (rather than defaulting to "dark") avoids a visible
// flicker of the wrong icon on the very first render.
function getSnapshot(): Theme {
  return (document.documentElement.dataset.theme as Theme) ?? "dark";
}

function getServerSnapshot(): Theme {
  return "dark";
}

export function ThemeToggle() {
  const theme = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

  return (
    <button
      onClick={() => applyTheme(theme === "light" ? "dark" : "light")}
      aria-label="Toggle light/dark theme"
      className="rounded-full border border-surface-border w-9 h-9 flex items-center justify-center hover:bg-white/5 transition-colors font-mono text-sm"
    >
      {theme === "light" ? "☾" : "☀"}
    </button>
  );
}
