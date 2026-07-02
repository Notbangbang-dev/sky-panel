import { useState } from "react";
import { AdminUsersTab } from "../components/admin/AdminUsersTab";
import { AdminNodesTab } from "../components/admin/AdminNodesTab";
import { AdminAllocationsTab } from "../components/admin/AdminAllocationsTab";
import { AdminEggsTab } from "../components/admin/AdminEggsTab";
import { AdminEconomyTab } from "../components/admin/AdminEconomyTab";
import { AdminSettingsTab } from "../components/admin/AdminSettingsTab";
import { AdminAuditTab } from "../components/admin/AdminAuditTab";
import { AdminBroadcastTab } from "../components/admin/AdminBroadcastTab";

const TABS = ["Users", "Nodes", "Allocations", "Eggs", "Economy", "Settings", "Audit log", "Broadcast"] as const;
type Tab = (typeof TABS)[number];

export function AdminPage() {
  const [tab, setTab] = useState<Tab>("Users");

  return (
    <div>
      <h1 className="sp-page-title">Admin</h1>

      <div style={{ display: "flex", gap: 6, marginBottom: 20, flexWrap: "wrap" }}>
        {TABS.map((t) => (
          <button
            key={t}
            className="sp-btn sp-btn--sm"
            style={t === tab ? { background: "var(--sp-accent)", color: "var(--sp-accent-text)" } : undefined}
            onClick={() => setTab(t)}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "Users" && <AdminUsersTab />}
      {tab === "Nodes" && <AdminNodesTab />}
      {tab === "Allocations" && <AdminAllocationsTab />}
      {tab === "Eggs" && <AdminEggsTab />}
      {tab === "Economy" && <AdminEconomyTab />}
      {tab === "Settings" && <AdminSettingsTab />}
      {tab === "Audit log" && <AdminAuditTab />}
      {tab === "Broadcast" && <AdminBroadcastTab />}
    </div>
  );
}
