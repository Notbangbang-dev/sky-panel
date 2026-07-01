import { describe, expect, it } from "vitest";
import { wsBaseFor } from "./useTopic";

describe("wsBaseFor", () => {
  it("swaps the scheme when apiBase is an absolute URL (local dev, cross-port)", () => {
    expect(wsBaseFor("http://localhost:8080", { protocol: "http:", host: "localhost:5180" })).toBe(
      "ws://localhost:8080",
    );
    expect(wsBaseFor("https://api.example.com", { protocol: "https:", host: "example.com" })).toBe(
      "wss://api.example.com",
    );
  });

  it("derives ws(s)://host from the page location when apiBase is empty (same-origin prod deploy)", () => {
    expect(wsBaseFor("", { protocol: "http:", host: "13.59.239.108" })).toBe("ws://13.59.239.108");
    expect(wsBaseFor("", { protocol: "https:", host: "panel.example.com" })).toBe("wss://panel.example.com");
  });
});
