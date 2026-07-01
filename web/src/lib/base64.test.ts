import { describe, expect, it } from "vitest";
import { decodeUtf8Base64, encodeUtf8Base64 } from "./base64";

describe("encodeUtf8Base64 / decodeUtf8Base64", () => {
  it("round trips plain ASCII", () => {
    expect(decodeUtf8Base64(encodeUtf8Base64("server-name=Survival"))).toBe("server-name=Survival");
  });

  it("round trips multi-byte UTF-8 characters", () => {
    const text = "motd=欢迎 🎮";
    expect(decodeUtf8Base64(encodeUtf8Base64(text))).toBe(text);
  });

  it("round trips an empty string", () => {
    expect(decodeUtf8Base64(encodeUtf8Base64(""))).toBe("");
  });
});
