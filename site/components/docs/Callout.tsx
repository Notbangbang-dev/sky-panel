// A left-accented callout for tips and warnings. Info uses the signal accent;
// warn stays monochrome (muted) so the palette never breaks its black/white rule.
export function Callout({
  tone = "info",
  title,
  children,
}: {
  tone?: "info" | "warn";
  title?: string;
  children: React.ReactNode;
}) {
  const accent = tone === "warn" ? "var(--sp-text-muted)" : "var(--sp-signal)";
  return (
    <div
      className="my-4 rounded-lg border border-surface-border px-4 py-3 text-sm text-text-muted"
      style={{ borderLeftWidth: 3, borderLeftColor: accent, background: "color-mix(in srgb, var(--sp-surface) 55%, transparent)" }}
    >
      {title && (
        <p className="mb-1 font-mono text-[11px] uppercase tracking-[0.18em]" style={{ color: accent }}>
          {title}
        </p>
      )}
      <div className="space-y-2 leading-relaxed">{children}</div>
    </div>
  );
}
