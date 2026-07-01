import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { apiRequest, ApiError } from "./api";
import { useAuthStore } from "./authStore";

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("apiRequest", () => {
  beforeEach(() => {
    useAuthStore.getState().clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns parsed JSON on success", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { hello: "world" }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiRequest<{ hello: string }>("/api/v1/whatever", { auth: false });
    expect(result).toEqual({ hello: "world" });
  });

  it("returns undefined for a 204 No Content response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 204 })));

    const result = await apiRequest<void>("/api/v1/whatever", { method: "POST", auth: false });
    expect(result).toBeUndefined();
  });

  it("throws an ApiError with the server's error code and message on failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(jsonResponse(409, { error: "already_exists", message: "email taken" })),
    );

    await expect(apiRequest("/api/v1/auth/register", { method: "POST", auth: false })).rejects.toMatchObject({
      status: 409,
      code: "already_exists",
      message: "email taken",
    });
  });

  it("is an instance of ApiError", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(400, { error: "bad_request", message: "nope" })));

    await expect(apiRequest("/x", { auth: false })).rejects.toBeInstanceOf(ApiError);
  });

  it("attaches the Authorization header when auth is required and a token is present", async () => {
    useAuthStore.getState().setSession({
      access_token: "access-123",
      refresh_token: "refresh-123",
      user: { id: "u1", email: "a@b.com", username: "a", role: "user", totp_enabled: false, coins: 0 },
    });

    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, {}));
    vi.stubGlobal("fetch", fetchMock);

    await apiRequest("/api/v1/me");

    const headers = fetchMock.mock.calls[0][1].headers as Record<string, string>;
    expect(headers["Authorization"]).toBe("Bearer access-123");
  });

  it("refreshes the access token and retries once on a 401", async () => {
    useAuthStore.getState().setSession({
      access_token: "expired-access",
      refresh_token: "refresh-123",
      user: { id: "u1", email: "a@b.com", username: "a", role: "user", totp_enabled: false, coins: 0 },
    });

    const fetchMock = vi.fn();
    // 1) original request -> 401
    fetchMock.mockResolvedValueOnce(jsonResponse(401, { error: "invalid_or_expired_token", message: "nope" }));
    // 2) refresh call -> new tokens
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        access_token: "fresh-access",
        refresh_token: "fresh-refresh",
        user: { id: "u1", email: "a@b.com", username: "a", role: "user", totp_enabled: false, coins: 0 },
      }),
    );
    // 3) retried original request -> success
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { ok: true }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiRequest<{ ok: boolean }>("/api/v1/me");

    expect(result).toEqual({ ok: true });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(useAuthStore.getState().accessToken).toBe("fresh-access");
  });
});
