import type { ReactNode } from "react";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { useAuthStore } from "../lib/authStore";

export function AppShell({ children }: { children: ReactNode }) {
  const user = useAuthStore((s) => s.user);

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
