import type { ReactNode } from "react";
import { useAppearance } from "../lib/AppearanceProvider";
import { useAuthStore } from "../lib/authStore";

// When maintenance mode is on, everyone except admins sees a full-screen
// notice instead of the app. Admins pass through (with the rest of the panel
// still usable) so they can turn it back off.
export function MaintenanceGate({ children }: { children: ReactNode }) {
  const { maintenance } = useAppearance();
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin";

  if (maintenance.enabled && !isAdmin) {
    return (
      <div className="sp-maintenance">
        <div className="sp-maintenance__card sp-surface">
          <div className="sp-maintenance__glyph">◐</div>
          <h1 className="sp-maintenance__title">Under maintenance</h1>
          <p className="sp-maintenance__msg">
            {maintenance.message || "Sky Panel is briefly offline for maintenance. Please check back shortly."}
          </p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
