import { useState } from "react";
import { AdminUsersTab } from "../components/admin/AdminUsersTab";
import { AdminServersTab } from "../components/admin/AdminServersTab";
import { AdminCodesTab } from "../components/admin/AdminCodesTab";
import { AdminNodesTab } from "../components/admin/AdminNodesTab";
import { AdminAllocationsTab } from "../components/admin/AdminAllocationsTab";
import { AdminEggsTab } from "../components/admin/AdminEggsTab";
import { AdminEconomyTab } from "../components/admin/AdminEconomyTab";
import { AdminQuotasTab } from "../components/admin/AdminQuotasTab";
import { AdminSettingsTab } from "../components/admin/AdminSettingsTab";
import { AdminAuditTab } from "../components/admin/AdminAuditTab";
import { AdminBroadcastTab } from "../components/admin/AdminBroadcastTab";
import { AdminAppearanceTab } from "../components/admin/AdminAppearanceTab";
import { AdminAnalyticsTab } from "../components/admin/AdminAnalyticsTab";

const TABS = ["Analytics", "Users", "Servers", "Nodes", "Allocations", "Eggs", "Economy", "Codes", "Quotas", "Appearance", "Settings", "Audit log", "Broadcast"] as const;
type Tab = (typeof TABS)[number];

export function AdminPage() {
  const [tab, setTab] = useState<Tab>("Analytics");

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

      {tab === "Analytics" && <AdminAnalyticsTab />}
      {tab === "Users" && <AdminUsersTab />}
      {tab === "Servers" && <AdminServersTab />}
      {tab === "Nodes" && <AdminNodesTab />}
      {tab === "Allocations" && <AdminAllocationsTab />}
      {tab === "Eggs" && <AdminEggsTab />}
      {tab === "Economy" && <AdminEconomyTab />}
      {tab === "Codes" && <AdminCodesTab />}
      {tab === "Quotas" && <AdminQuotasTab />}
      {tab === "Appearance" && <AdminAppearanceTab />}
      {tab === "Settings" && <AdminSettingsTab />}
      {tab === "Audit log" && <AdminAuditTab />}
      {tab === "Broadcast" && <AdminBroadcastTab />}
    </div>
  );
}
