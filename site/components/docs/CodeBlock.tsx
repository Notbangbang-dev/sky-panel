"use client";

import { useState } from "react";

// A code block with a filename/lang chip and a copy button — the small touch
// that makes docs feel finished. Monochrome shell; the copy affordance lights
// up in the signal accent on hover.
export function CodeBlock({ code, label = "shell" }: { code: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  function copy() {
    navigator.clipboard?.writeText(code).then(
      () => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      },
      () => setCopied(false),
    );
  }

  return (
    <div className="my-4 overflow-hidden rounded-lg border border-surface-border bg-black/30">
      <div className="flex items-center justify-between border-b border-surface-border bg-white/[0.02] px-3 py-1.5">
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-text-muted">{label}</span>
        <button
          onClick={copy}
          className="font-mono text-[10px] uppercase tracking-[0.2em] text-text-muted transition-colors hover:text-signal"
        >
          {copied ? "✓ copied" : "copy"}
        </button>
      </div>
      <pre className="overflow-x-auto p-4 text-xs leading-relaxed font-mono text-text">
        <code>{code}</code>
      </pre>
    </div>
  );
}
