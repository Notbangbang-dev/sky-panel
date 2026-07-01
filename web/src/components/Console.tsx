import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

interface ConsoleProps {
  lines: string[];
  onInput: (line: string) => void;
}

const KEY_ENTER = "\r";
const KEY_BACKSPACE_CODE = 127; // DEL, what terminals send for the Backspace key

export function Console({ lines, onInput }: ConsoleProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const writtenCountRef = useRef(0);
  const bufferRef = useRef("");
  const onInputRef = useRef(onInput);
  onInputRef.current = onInput;

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const term = new Terminal({
      convertEol: true,
      fontFamily: "IBM Plex Mono, monospace",
      fontSize: 13,
      theme: {
        background: "#00000000",
        foreground: getVar("--sp-text", "#f2f2f0"),
        cursor: getVar("--sp-accent", "#f2f2f0"),
      },
      cursorBlink: true,
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(container);
    fitAddon.fit();

    term.onData((data) => {
      if (data === KEY_ENTER) {
        const line = bufferRef.current;
        bufferRef.current = "";
        term.write("\r\n");
        if (line.trim()) onInputRef.current(line);
        return;
      }

      if (data.charCodeAt(0) === KEY_BACKSPACE_CODE) {
        if (bufferRef.current.length > 0) {
          bufferRef.current = bufferRef.current.slice(0, -1);
          term.write("\b \b");
        }
        return;
      }

      if (data >= " ") {
        bufferRef.current += data;
        term.write(data);
      }
    });

    const resizeObserver = new ResizeObserver(() => fitAddon.fit());
    resizeObserver.observe(container);

    termRef.current = term;
    writtenCountRef.current = 0;

    return () => {
      resizeObserver.disconnect();
      term.dispose();
      termRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const term = termRef.current;
    if (!term) return;
    for (let i = writtenCountRef.current; i < lines.length; i++) {
      term.writeln(lines[i]);
    }
    writtenCountRef.current = lines.length;
  }, [lines]);

  return <div ref={containerRef} style={{ height: "100%", width: "100%" }} />;
}

function getVar(name: string, fallback: string): string {
  if (typeof document === "undefined") return fallback;
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || fallback;
}
