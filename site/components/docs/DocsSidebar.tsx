"use client";

import { useEffect, useState } from "react";

export interface DocItem {
  id: string;
  label: string;
}

export interface DocGroup {
  group: string;
  items: DocItem[];
}

// Sticky docs nav with scrollspy: an IntersectionObserver marks the section
// nearest the top of the viewport as active, and the matching link gets the
// signal-accent left bar. rootMargin biases the "active" band to the upper
// third so a heading lights up as it reaches the top, not the middle.
export function DocsSidebar({ groups }: { groups: DocGroup[] }) {
  const ids = groups.flatMap((g) => g.items.map((i) => i.id));
  const [active, setActive] = useState(ids[0] ?? "");

  useEffect(() => {
    const els = ids.map((id) => document.getElementById(id)).filter((el): el is HTMLElement => el !== null);
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]) setActive(visible[0].target.id);
      },
      { rootMargin: "-15% 0px -70% 0px", threshold: 0 },
    );
    els.forEach((el) => observer.observe(el));
    return () => observer.disconnect();
    // ids is derived from the static groups prop; re-run only if it changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groups]);

  return (
    <nav className="flex flex-col gap-5 text-sm">
      {groups.map((g) => (
        <div key={g.group} className="flex flex-col gap-0.5">
          <p className="mb-1 font-mono text-[10px] uppercase tracking-[0.2em] text-text-muted/70">{g.group}</p>
          {g.items.map((item) => {
            const isActive = active === item.id;
            return (
              <a
                key={item.id}
                href={`#${item.id}`}
                className="border-l-2 py-1 pl-3 transition-colors"
                style={{
                  borderColor: isActive ? "var(--sp-signal)" : "var(--sp-surface-border)",
                  color: isActive ? "var(--sp-text)" : "var(--sp-text-muted)",
                }}
              >
                {item.label}
              </a>
            );
          })}
        </div>
      ))}
    </nav>
  );
}
