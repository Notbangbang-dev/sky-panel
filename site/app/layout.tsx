import type { Metadata } from "next";
import { IBM_Plex_Sans, IBM_Plex_Mono, Instrument_Serif } from "next/font/google";
import { noFlashScript } from "@/lib/theme";
import "./globals.css";

const plexSans = IBM_Plex_Sans({
  variable: "--font-plex-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

const plexMono = IBM_Plex_Mono({
  variable: "--font-plex-mono",
  subsets: ["latin"],
  weight: ["400", "500", "600"],
});

const instrumentSerif = Instrument_Serif({
  variable: "--font-instrument-serif",
  subsets: ["latin"],
  weight: "400",
});

export const metadata: Metadata = {
  title: "Sky Panel — a game server panel that doesn't get in your way",
  description:
    "Go + Rust backend, real Docker orchestration, live stats, a coin economy, and an admin console that actually works. Self-hosted, open source.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      lang="en"
      className={`${plexSans.variable} ${plexMono.variable} ${instrumentSerif.variable} h-full antialiased`}
      // The no-flash script below sets data-theme on this element before
      // React hydrates, which will always differ from the server-rendered
      // markup (the server doesn't know the visitor's stored preference).
      // That's expected — suppress the resulting warning rather than the
      // (worse) alternative of a flash of the wrong theme on every load.
      suppressHydrationWarning
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: noFlashScript }} />
      </head>
      <body className="min-h-full flex flex-col bg-bg text-text">
        <div className="grain" />
        {children}
      </body>
    </html>
  );
}
