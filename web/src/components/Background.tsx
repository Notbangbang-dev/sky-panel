import { useEffect, useState } from "react";
import { AnimatedBackground } from "./AnimatedBackground";
import { useAppearance } from "../lib/AppearanceProvider";

// Renders whichever background mode the admin selected, always fixed behind the
// app (z-index 0). "animated" keeps the original node-mesh canvas; the others
// layer a fixed image/video/gradient/solid with an optional blur + dim overlay.
// A broken image/video URL falls back to the animated mesh rather than leaving
// a dead background instance-wide.
export function Background() {
  const { background: bg } = useAppearance();
  const [mediaFailed, setMediaFailed] = useState(false);

  // Reset the failure flag whenever the source changes, and preload images
  // (CSS background-image has no onError) so a broken URL falls back too.
  useEffect(() => {
    setMediaFailed(false);
    if (bg.mode === "image" && bg.imageUrl) {
      const img = new Image();
      img.onerror = () => setMediaFailed(true);
      img.src = bg.imageUrl;
    }
  }, [bg.mode, bg.imageUrl, bg.videoUrl]);

  if (bg.mode === "animated") {
    return <AnimatedBackground />;
  }

  if (bg.mode === "solid") {
    return <div className="sp-bg-layer" aria-hidden style={{ background: bg.color || "var(--sp-bg)" }} />;
  }

  if (bg.mode === "gradient") {
    return <div className="sp-bg-layer" aria-hidden style={{ background: bg.gradient }} />;
  }

  if (bg.mode === "image" && bg.imageUrl && !mediaFailed) {
    return (
      <div className="sp-bg-layer" aria-hidden>
        <div
          className="sp-bg-media"
          style={{
            backgroundImage: `url("${bg.imageUrl}")`,
            filter: bg.blur ? `blur(${bg.blur}px)` : undefined,
          }}
        />
        {bg.dim > 0 && <div className="sp-bg-dim" style={{ opacity: bg.dim }} />}
      </div>
    );
  }

  if (bg.mode === "video" && bg.videoUrl && !mediaFailed) {
    return (
      <div className="sp-bg-layer" aria-hidden>
        <video
          className="sp-bg-media"
          src={bg.videoUrl}
          autoPlay
          muted
          loop
          playsInline
          onError={() => setMediaFailed(true)}
          style={{ filter: bg.blur ? `blur(${bg.blur}px)` : undefined }}
        />
        {bg.dim > 0 && <div className="sp-bg-dim" style={{ opacity: bg.dim }} />}
      </div>
    );
  }

  // Misconfigured (empty URL) or the media failed to load — fall back to the mesh.
  return <AnimatedBackground />;
}
