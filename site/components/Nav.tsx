import Link from "next/link";
import { ThemeToggle } from "./ThemeToggle";

export function Nav() {
  return (
    <header className="relative z-10 flex items-center justify-between px-6 md:px-12 py-6">
      <Link href="/" className="flex items-baseline gap-2">
        <span className="font-display text-2xl">SKY</span>
        <span className="font-mono text-[10px] tracking-[0.22em] text-text-muted">PANEL</span>
      </Link>
      <nav className="flex items-center gap-6 text-sm">
        <Link href="/docs" className="text-text-muted hover:text-text transition-colors">
          Docs
        </Link>
        <Link href="/changelog" className="text-text-muted hover:text-text transition-colors">
          Changelog
        </Link>
        <a
          href="https://github.com/Notbangbang-dev/sky-panel"
          className="text-text-muted hover:text-text transition-colors"
        >
          GitHub
        </a>
        <a
          href="https://github.com/Notbangbang-dev/sky-panel#readme"
          className="rounded-full border border-surface-border px-4 py-2 hover:bg-white/5 transition-colors"
        >
          Get started
        </a>
        <ThemeToggle />
      </nav>
    </header>
  );
}
