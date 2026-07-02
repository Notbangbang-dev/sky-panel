import { NavLink } from "react-router-dom";
import { motion } from "framer-motion";
import clsx from "clsx";

interface NavItem {
  label: string;
  to: string;
  glyph: string;
}

const PRIMARY_NAV: NavItem[] = [
  { label: "Dashboard", to: "/", glyph: "◆" },
  { label: "Servers", to: "/servers", glyph: "▣" },
  { label: "Nodes", to: "/nodes", glyph: "⬢" },
  { label: "Wallet", to: "/wallet", glyph: "◈" },
  { label: "Store", to: "/store", glyph: "⧉" },
  { label: "Leaderboard", to: "/leaderboard", glyph: "♛" },
  { label: "AFK", to: "/afk", glyph: "◐" },
  { label: "Account", to: "/account", glyph: "◎" },
];

const ADMIN_NAV: NavItem[] = [{ label: "Admin", to: "/admin", glyph: "▲" }];

export function Sidebar({ isAdmin = false }: { isAdmin?: boolean }) {
  return (
    <aside className="sp-sidebar sp-surface">
      <div className="sp-sidebar__brand">
        <span className="sp-sidebar__mark">SKY</span>
        <span className="sp-sidebar__mark-sub">PANEL</span>
      </div>

      <nav className="sp-sidebar__nav">
        {PRIMARY_NAV.map((item) => (
          <SidebarLink key={item.to} item={item} />
        ))}
      </nav>

      {isAdmin && (
        <>
          <div className="sp-sidebar__divider" />
          <nav className="sp-sidebar__nav">
            {ADMIN_NAV.map((item) => (
              <SidebarLink key={item.to} item={item} />
            ))}
          </nav>
        </>
      )}

      <div className="sp-sidebar__status">
        <span className="sp-sidebar__pulse" />
        <span>system online</span>
      </div>
    </aside>
  );
}

function SidebarLink({ item }: { item: NavItem }) {
  return (
    <NavLink to={item.to} className="sp-sidebar__link" end={item.to === "/"}>
      {({ isActive }) => (
        <span className={clsx("sp-sidebar__link-inner", isActive && "is-active")}>
          <span className="sp-sidebar__glyph">{item.glyph}</span>
          <span>{item.label}</span>
          {isActive && (
            <motion.span
              layoutId="sidebar-active-indicator"
              className="sp-sidebar__indicator"
              transition={{ type: "spring", stiffness: 400, damping: 32 }}
            />
          )}
        </span>
      )}
    </NavLink>
  );
}
