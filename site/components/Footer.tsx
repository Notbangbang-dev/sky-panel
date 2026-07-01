export function Footer() {
  return (
    <footer className="relative z-10 border-t border-surface-border px-6 md:px-12 py-8 mt-auto">
      <div className="flex flex-col md:flex-row justify-between gap-4 text-sm text-text-muted">
        <p className="font-mono">Sky Panel — Deploy. Scale. Dominate.</p>
        <div className="flex gap-6">
          <a href="https://github.com/Notbangbang-dev/sky-panel" className="hover:text-text transition-colors">
            GitHub
          </a>
          <a href="https://github.com/Notbangbang-dev/sky-panel/blob/main/LICENSE" className="hover:text-text transition-colors">
            MIT License
          </a>
        </div>
      </div>
    </footer>
  );
}
