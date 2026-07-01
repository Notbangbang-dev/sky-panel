"use client";

import { useEffect, useRef } from "react";

interface Node {
  x: number;
  y: number;
  vx: number;
  vy: number;
}

const NODE_COUNT = 60;
const SPEED = 0.09;
const LINK_DIST = 150;

/**
 * The same "server mesh" ambient backdrop as the panel app, reimplemented
 * here since this is a separate Next.js project — a field of drifting nodes
 * connected by fading lines. Pauses when the tab is hidden and respects
 * prefers-reduced-motion.
 */
export function AnimatedBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const dpr = Math.min(window.devicePixelRatio || 1, 2);

    let width = 0;
    let height = 0;
    let nodes: Node[] = [];
    let frameId = 0;
    let running = false;

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
      nodes = Array.from({ length: NODE_COUNT }, () => ({
        x: Math.random() * width,
        y: Math.random() * height,
        vx: (Math.random() - 0.5) * (prefersReducedMotion ? 0 : SPEED),
        vy: (Math.random() - 0.5) * (prefersReducedMotion ? 0 : SPEED),
      }));
    }

    function step() {
      if (!running) return;
      if (resizeIfNeeded()) seedNodes();

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
          const dx = nodes[i].x - nodes[j].x;
          const dy = nodes[i].y - nodes[j].y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          if (dist < LINK_DIST) {
            const alpha = (1 - dist / LINK_DIST) * 0.3;
            ctx!.strokeStyle = `rgba(242, 242, 240, ${alpha})`;
            ctx!.lineWidth = 1;
            ctx!.beginPath();
            ctx!.moveTo(nodes[i].x, nodes[i].y);
            ctx!.lineTo(nodes[j].x, nodes[j].y);
            ctx!.stroke();
          }
        }
      }

      for (const n of nodes) {
        ctx!.fillStyle = "rgba(200, 255, 61, 0.75)";
        ctx!.beginPath();
        ctx!.arc(n.x, n.y, 1.6, 0, Math.PI * 2);
        ctx!.fill();
      }

      frameId = requestAnimationFrame(step);
    }

    function start() {
      if (running) return;
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
  }, []);

  return (
    <canvas
      ref={canvasRef}
      aria-hidden="true"
      className="fixed inset-0 z-0 pointer-events-none"
      style={{
        background:
          "radial-gradient(circle at 15% 20%, #0e1013 0%, transparent 55%), radial-gradient(circle at 85% 80%, #0e1013 0%, transparent 55%), #08090b",
      }}
    />
  );
}
