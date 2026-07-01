import { useEffect, useRef } from "react";
import { useTheme } from "../lib/ThemeProvider";
import type { AnimationIntensity } from "../lib/theme";

interface Node {
  x: number;
  y: number;
  vx: number;
  vy: number;
}

const INTENSITY_CONFIG: Record<AnimationIntensity, { count: number; speed: number; linkDist: number; opacity: number }> = {
  off: { count: 0, speed: 0, linkDist: 0, opacity: 0 },
  subtle: { count: 34, speed: 0.06, linkDist: 130, opacity: 0.22 },
  normal: { count: 52, speed: 0.1, linkDist: 150, opacity: 0.32 },
  high: { count: 78, speed: 0.16, linkDist: 170, opacity: 0.44 },
};

/**
 * Ambient "server mesh" backdrop: a field of drifting nodes connected by
 * fading lines when close, evoking a topology of hosts rather than generic
 * particles. Colors come from the active theme's CSS variables so it
 * re-themes live; intensity/count come from the theme's animationIntensity.
 *
 * Size tracking is done by comparing clientWidth/clientHeight on every
 * animation frame rather than a `resize` event or ResizeObserver — some
 * automated/embedded browser contexts never fire either of those for the
 * root element, and the per-frame check is cheap enough not to matter.
 */
export function AnimatedBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const { theme } = useTheme();

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const config = prefersReducedMotion
      ? { ...INTENSITY_CONFIG[theme.animationIntensity], speed: 0 }
      : INTENSITY_CONFIG[theme.animationIntensity];

    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    let width = 0;
    let height = 0;
    let nodes: Node[] = [];
    let frameId = 0;
    let running = false;

    function readColor(varName: string, fallback: string): string {
      const value = getComputedStyle(document.documentElement).getPropertyValue(varName).trim();
      return value || fallback;
    }

    function resizeIfNeeded() {
      const nextWidth = document.documentElement.clientWidth;
      const nextHeight = document.documentElement.clientHeight;
      if (nextWidth === width && nextHeight === height) return false;

      width = nextWidth;
      height = nextHeight;
      canvas!.width = width * dpr;
      canvas!.height = height * dpr;
      canvas!.style.width = `${width}px`;
      canvas!.style.height = `${height}px`;
      ctx!.setTransform(dpr, 0, 0, dpr, 0, 0);
      return true;
    }

    function seedNodes() {
      nodes = Array.from({ length: config.count }, () => ({
        x: Math.random() * width,
        y: Math.random() * height,
        vx: (Math.random() - 0.5) * config.speed,
        vy: (Math.random() - 0.5) * config.speed,
      }));
    }

    function step() {
      if (!running) return;

      if (resizeIfNeeded()) {
        seedNodes();
      }

      const lineColor = readColor("--sp-text", "#f2f2f0");
      const nodeColor = readColor("--sp-accent", "#f2f2f0");

      ctx!.clearRect(0, 0, width, height);

      for (const n of nodes) {
        n.x += n.vx;
        n.y += n.vy;
        if (n.x < 0) n.x = width;
        if (n.x > width) n.x = 0;
        if (n.y < 0) n.y = height;
        if (n.y > height) n.y = 0;
      }

      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const a = nodes[i];
          const b = nodes[j];
          const dx = a.x - b.x;
          const dy = a.y - b.y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          if (dist < config.linkDist) {
            const alpha = (1 - dist / config.linkDist) * config.opacity;
            ctx!.strokeStyle = withAlpha(lineColor, alpha);
            ctx!.lineWidth = 1;
            ctx!.beginPath();
            ctx!.moveTo(a.x, a.y);
            ctx!.lineTo(b.x, b.y);
            ctx!.stroke();
          }
        }
      }

      for (const n of nodes) {
        ctx!.fillStyle = withAlpha(nodeColor, Math.min(config.opacity * 2.2, 0.9));
        ctx!.beginPath();
        ctx!.arc(n.x, n.y, 1.6, 0, Math.PI * 2);
        ctx!.fill();
      }

      frameId = requestAnimationFrame(step);
    }

    function start() {
      if (running || config.count === 0) return;
      running = true;
      frameId = requestAnimationFrame(step);
    }

    function stop() {
      running = false;
      cancelAnimationFrame(frameId);
    }

    function handleVisibility() {
      if (document.hidden) stop();
      else start();
    }

    document.addEventListener("visibilitychange", handleVisibility);
    start();

    return () => {
      stop();
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [theme]);

  return <canvas ref={canvasRef} className="sp-animated-background" aria-hidden="true" />;
}

function withAlpha(color: string, alpha: number): string {
  if (color.startsWith("#")) {
    const hex = color.replace("#", "");
    const bigint = parseInt(hex.length === 3 ? hex.repeat(2) : hex, 16);
    const r = (bigint >> 16) & 255;
    const g = (bigint >> 8) & 255;
    const b = bigint & 255;
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
  }
  return color;
}
