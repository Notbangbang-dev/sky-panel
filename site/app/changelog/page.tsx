import { AnimatedBackground } from "@/components/AnimatedBackground";
import { Nav } from "@/components/Nav";
import { Footer } from "@/components/Footer";

export const revalidate = 300;

interface Entry {
  heading: string;
  body: string[];
}

function parseChangelog(markdown: string): Entry[] {
  const lines = markdown.split("\n");
  const entries: Entry[] = [];
  let current: Entry | null = null;

  for (const line of lines) {
    if (line.startsWith("## ")) {
      if (current) entries.push(current);
      current = { heading: line.replace(/^##\s*/, ""), body: [] };
    } else if (current && line.trim() && !line.startsWith("# ")) {
      current.body.push(line);
    }
  }
  if (current) entries.push(current);
  return entries;
}

async function getChangelog(): Promise<Entry[]> {
  try {
    const res = await fetch("https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/CHANGELOG.md", {
      next: { revalidate: 300 },
    });
    if (!res.ok) return [];
    return parseChangelog(await res.text());
  } catch {
    return [];
  }
}

export default async function ChangelogPage() {
  const entries = await getChangelog();

  return (
    <>
      <AnimatedBackground />
      <Nav />

      <main className="relative z-10 px-6 md:px-12 max-w-3xl mx-auto py-20 flex-1">
        <h1 className="font-display text-4xl mb-10">Changelog</h1>

        {entries.length === 0 && (
          <p className="text-text-muted">
            Couldn&apos;t load the changelog right now — see it directly on{" "}
            <a href="https://github.com/Notbangbang-dev/sky-panel/blob/main/CHANGELOG.md" className="underline">
              GitHub
            </a>
            .
          </p>
        )}

        <div className="space-y-12">
          {entries.map((entry) => (
            <section key={entry.heading}>
              <h2 className="font-mono text-sm text-signal mb-3">{entry.heading}</h2>
              <div className="space-y-2 text-sm text-text-muted leading-relaxed">
                {entry.body.map((line, i) => (
                  <p key={i}>{line.replace(/^-\s*/, "").replace(/^###\s*/, "")}</p>
                ))}
              </div>
            </section>
          ))}
        </div>
      </main>

      <Footer />
    </>
  );
}
