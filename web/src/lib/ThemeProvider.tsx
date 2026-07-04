import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  applyThemeToDocument,
  DEFAULT_THEME,
  loadActiveThemeId,
  loadCustomThemes,
  PRESET_THEMES,
  saveActiveThemeId,
  saveCustomThemes,
  type Theme,
} from "./theme";
import { useAppearance } from "./AppearanceProvider";

interface ThemeContextValue {
  theme: Theme;
  themes: Theme[];
  setThemeId: (id: string) => void;
  saveCustomTheme: (theme: Theme) => void;
  deleteCustomTheme: (id: string) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const { adminThemeId } = useAppearance();
  const [customThemes, setCustomThemes] = useState<Theme[]>(() => loadCustomThemes());
  // A user who has explicitly picked a theme keeps it; otherwise we follow the
  // instance-wide admin preset, falling back to the built-in default.
  const [hasUserChoice, setHasUserChoice] = useState<boolean>(() => loadActiveThemeId() !== null);
  const [activeThemeId, setActiveThemeId] = useState<string>(() => loadActiveThemeId() ?? DEFAULT_THEME.id);

  const themes = useMemo(() => [...PRESET_THEMES, ...customThemes], [customThemes]);
  const effectiveId = hasUserChoice ? activeThemeId : adminThemeId || DEFAULT_THEME.id;
  const theme = useMemo(
    () => themes.find((t) => t.id === effectiveId) ?? DEFAULT_THEME,
    [themes, effectiveId],
  );

  useEffect(() => {
    applyThemeToDocument(theme);
  }, [theme]);

  const setThemeId = useCallback((id: string) => {
    setActiveThemeId(id);
    setHasUserChoice(true);
    saveActiveThemeId(id);
  }, []);

  const saveCustomTheme = useCallback((next: Theme) => {
    setCustomThemes((prev) => {
      const withoutExisting = prev.filter((t) => t.id !== next.id);
      const updated = [...withoutExisting, next];
      saveCustomThemes(updated);
      return updated;
    });
    setThemeId(next.id);
  }, [setThemeId]);

  const deleteCustomTheme = useCallback((id: string) => {
    setCustomThemes((prev) => {
      const updated = prev.filter((t) => t.id !== id);
      saveCustomThemes(updated);
      return updated;
    });
    setActiveThemeId((current) => (current === id ? DEFAULT_THEME.id : current));
  }, []);

  const value = useMemo(
    () => ({ theme, themes, setThemeId, saveCustomTheme, deleteCustomTheme }),
    [theme, themes, setThemeId, saveCustomTheme, deleteCustomTheme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used within a ThemeProvider");
  return ctx;
}
