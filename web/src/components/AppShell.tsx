import { useEffect, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { useAuthStore } from "../lib/authStore";
import { authApi } from "../lib/endpoints";

export function AppShell({ children }: { children: ReactNode }) {
  const user = useAuthStore((s) => s.user);
  const updateUser = useAuthStore((s) => s.updateUser);

  // Keep the live balance/role fresh everywhere — poll /me and on window focus.
  // Coins change server-side (AFK, admin adjust) even on pages that don't push
  // an update themselves, so without this the topbar counter goes stale.
  const { data: me } = useQuery({
    queryKey: ["me"],
    queryFn: authApi.me,
    refetchInterval: 15_000,
    // Keep polling even when the panel is in a background tab, so the balance
    // is current the moment the user looks back at it.
    refetchIntervalInBackground: true,
    refetchOnWindowFocus: true,
  });

  useEffect(() => {
    if (!me) return;
    const cur = useAuthStore.getState().user;
    if (
      !cur ||
      cur.coins !== me.coins ||
      cur.role !== me.role ||
      cur.username !== me.username ||
      cur.totp_enabled !== me.totp_enabled
    ) {
      updateUser(me);
    }
  }, [me, updateUser]);

  return (
    <div className="sp-shell">
      <Sidebar isAdmin={user?.role === "admin"} />
      <div className="sp-shell__main">
        <Topbar username={user?.username ?? ""} coins={user?.coins ?? 0} />
        <main className="sp-shell__content">{children}</main>
      </div>
    </div>
  );
}
