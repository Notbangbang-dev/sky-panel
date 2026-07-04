import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { publicApi } from "./endpoints";
import { applyCustomCss, DEFAULT_BACKGROUND, parseBackground, type BackgroundConfig } from "./appearance";

interface AppearanceContextValue {
  /** Admin-selected preset theme id, or "" if unset. */
  adminThemeId: string;
  background: BackgroundConfig;
  maintenance: { enabled: boolean; message: string };
  /** Re-fetch appearance (used by the admin tab after saving). */
  reload: () => void;
}

const AppearanceContext = createContext<AppearanceContextValue>({
  adminThemeId: "",
  background: DEFAULT_BACKGROUND,
  maintenance: { enabled: false, message: "" },
  reload: () => {},
});

export function AppearanceProvider({ children }: { children: ReactNode }) {
  const [adminThemeId, setAdminThemeId] = useState("");
  const [background, setBackground] = useState<BackgroundConfig>(DEFAULT_BACKGROUND);
  const [maintenance, setMaintenance] = useState({ enabled: false, message: "" });

  const load = useCallback(() => {
    publicApi
      .appearance()
      .then((a) => {
        setAdminThemeId(a.theme_preset || "");
        setBackground(parseBackground(a.background));
        applyCustomCss(a.custom_css || "");
      })
      .catch(() => {
        // A fresh install has no appearance configured — keep the defaults.
      });
    publicApi
      .maintenance()
      .then((m) => setMaintenance({ enabled: m.enabled, message: m.message }))
      .catch(() => {});
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const value = useMemo(
    () => ({ adminThemeId, background, maintenance, reload: load }),
    [adminThemeId, background, maintenance, load],
  );

  return <AppearanceContext.Provider value={value}>{children}</AppearanceContext.Provider>;
}

export function useAppearance() {
  return useContext(AppearanceContext);
}
