import type { Metadata } from "next";
import { AnimatedBackground } from "@/components/AnimatedBackground";
import { Nav } from "@/components/Nav";
import { Footer } from "@/components/Footer";
import { CHANGELOG_FALLBACK } from "@/lib/changelog-fallback";

export const revalidate = 300;

export const metadata: Metadata = {
  title: "Changelog — Sky Panel",
  description: "Every Sky Panel release, newest first.",
};

type Block =
  | { type: "sub"; text: string }
  | { type: "bullets"; items: string[] }
  | { type: "para"; text: string };

interface Entry {
  version: string;
  date?: string;
  blocks: Block[];
}

// Parse CHANGELOG.md (Keep a Changelog style) into version entries, preserving
// section subheadings and bullet lists rather than flattening everything to
// plain paragraphs. `## [0.15.0] - 2026-07-02` → version + date.
function parseChangelog(markdown: string): Entry[] {
  const entries: Entry[] = [];
  let current: Entry | null = null;
  let bullets: string[] | null = null;

  const flushBullets = () => {
    if (current && bullets && bullets.length) current.blocks.push({ type: "bullets", items: bullets });
    bullets = null;
  };

  for (const raw of markdown.split("\n")) {
    const line = raw.trimEnd();
    if (line.startsWith("## ")) {
      flushBullets();
      if (current) entries.push(current);
      const heading = line.replace(/^##\s*/, "");
      const m = heading.match(/\[?([0-9]+\.[0-9]+\.[0-9]+)\]?\s*-?\s*(.*)$/);
      current = { version: m ? m[1] : heading, date: m && m[2] ? m[2].trim() : undefined, blocks: [] };
    } else if (!current) {
      continue;
    } else if (line.startsWith("### ")) {
      flushBullets();
      current.blocks.push({ type: "sub", text: line.replace(/^###\s*/, "") });
    } else if (line.startsWith("- ") || line.startsWith("* ")) {
      if (!bullets) bullets = [];
      bullets.push(line.replace(/^[-*]\s*/, ""));
    } else if (line.trim() && !line.startsWith("# ")) {
      flushBullets();
      current.blocks.push({ type: "para", text: line.trim() });
    } else {
      flushBullets();
    }
  }
  flushBullets();
  if (current) entries.push(current);
  return entries;
}

// Minimal inline markdown: **bold** and `code`. Our own changelog is the only
// input, so this stays deliberately small rather than pulling in a parser.
function inline(text: string, key: string) {
  return text
    .split(/(\*\*[^*]+\*\*|`[^`]+`)/g)
    .filter(Boolean)
    .map((part, i) => {
      if (part.startsWith("**") && part.endsWith("**")) {
        return (
          <strong key={`${key}-${i}`} className="font-semibold text-text">
            {part.slice(2, -2)}
          </strong>
        );
      }
      if (part.startsWith("`") && part.endsWith("`")) {
        return (
          <code key={`${key}-${i}`} className="rounded bg-white/5 px-1 py-0.5 font-mono text-[0.85em] text-text">
            {part.slice(1, -1)}
          </code>
        );
      }
      return <span key={`${key}-${i}`}>{part}</span>;
    });
}

// Parse the build-time bundled changelog so the page always has content to
// render even when the network fetch is unavailable.
function fallbackEntries(): Entry[] {
  return parseChangelog(CHANGELOG_FALLBACK);
}

async function getChangelog(): Promise<Entry[]> {
  try {
    const res = await fetch("https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/CHANGELOG.md", {
      next: { revalidate: 300 },
    });
    if (!res.ok) return fallbackEntries();
    const parsed = parseChangelog(await res.text());
    // A successful-but-empty response (e.g. a redirect page) should still
    // render the bundled entries rather than a blank timeline.
    return parsed.length ? parsed : fallbackEntries();
  } catch {
    return fallbackEntries();
  }
}

export default async function ChangelogPage() {
  const entries = await getChangelog();

  return (
    <>
      <AnimatedBackground />
      <Nav />

      <main className="relative z-10 mx-auto w-full max-w-3xl flex-1 px-6 py-16 md:px-12">
        <header className="reveal mb-14">
          <p className="mb-2 font-mono text-[11px] uppercase tracking-[0.28em] text-text-muted">Changelog</p>
          <h1 className="font-display text-4xl md:text-5xl">What&apos;s new</h1>
          <p className="mt-3 text-text-muted">Every Sky Panel release, newest first.</p>
        </header>

        {entries.length === 0 && (
          <p className="text-text-muted">
            Couldn&apos;t load the changelog right now — see it directly on{" "}
            <a href="https://github.com/Notbangbang-dev/sky-panel/blob/main/CHANGELOG.md" className="underline">
              GitHub
            </a>
            .
          </p>
        )}

        {/* Vertical timeline: a hairline runs down the left, each release marked by a node. */}
        <div className="relative border-l border-surface-border pl-8">
          <div className="space-y-14">
            {entries.map((entry) => (
              <article key={entry.version} id={`v${entry.version}`} className="relative scroll-mt-24">
                <span
                  className="absolute -left-[41px] top-1.5 grid h-4 w-4 place-items-center rounded-full border border-surface-border bg-bg"
                  aria-hidden
                >
                  <span className="h-1.5 w-1.5 rounded-full" style={{ background: "var(--sp-signal)" }} />
                </span>

                <div className="flex items-baseline gap-3">
                  <h2 className="font-display text-2xl text-text">v{entry.version}</h2>
                  {entry.date && <span className="font-mono text-xs text-text-muted">{entry.date}</span>}
                </div>

                <div className="mt-4 space-y-3 text-sm leading-relaxed text-text-muted">
                  {entry.blocks.map((block, i) => {
                    if (block.type === "sub") {
                      return (
                        <p key={i} className="pt-2 font-mono text-[12px] uppercase tracking-[0.16em] text-text">
                          {block.text}
                        </p>
                      );
                    }
                    if (block.type === "bullets") {
                      return (
                        <ul key={i} className="space-y-1.5">
                          {block.items.map((item, j) => (
                            <li key={j} className="flex gap-2">
                              <span className="mt-2 h-1 w-1 shrink-0 rounded-full" style={{ background: "var(--sp-signal)" }} aria-hidden />
                              <span>{inline(item, `${i}-${j}`)}</span>
                            </li>
                          ))}
                        </ul>
                      );
                    }
                    return <p key={i}>{inline(block.text, `p-${i}`)}</p>;
                  })}
                </div>
              </article>
            ))}
          </div>
        </div>
      </main>

      <Footer />
    </>
  );
}
