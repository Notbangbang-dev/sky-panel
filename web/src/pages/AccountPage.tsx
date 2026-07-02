import { useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { accountApi, authApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { ApiError } from "../lib/api";
import { copyText } from "../lib/clipboard";
import type { ApiKey, Session, TotpSetup } from "../types/api";

export function AccountPage() {
  const user = useAuthStore((s) => s.user);
  const updateUser = useAuthStore((s) => s.updateUser);

  const [setup, setSetup] = useState<TotpSetup | null>(null);
  const [code, setCode] = useState("");
  const [message, setMessage] = useState<string | null>(null);

  async function startSetup() {
    setMessage(null);
    try {
      setSetup(await authApi.totpSetup());
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : "Failed to start setup");
    }
  }

  async function confirm() {
    if (!user) return;
    try {
      await authApi.totpConfirm(code);
      setSetup(null);
      setCode("");
      updateUser({ ...user, totp_enabled: true });
      setMessage("Two-factor authentication enabled.");
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : "Invalid code");
    }
  }

  async function disable() {
    if (!user) return;
    try {
      await authApi.totpDisable(code);
      setCode("");
      updateUser({ ...user, totp_enabled: false });
      setMessage("Two-factor authentication disabled.");
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : "Invalid code");
    }
  }

  if (!user) return null;

  return (
    <div>
      <h1 className="sp-page-title">Account</h1>

      <div className="sp-surface sp-card" style={{ marginBottom: 20, maxWidth: 520 }}>
        <p className="sp-label">Username</p>
        <p style={{ marginBottom: 14 }}>{user.username}</p>
        <p className="sp-label">Email</p>
        <p>{user.email}</p>
      </div>

      <ChangePasswordCard />

      <div className="sp-surface sp-card" style={{ marginBottom: 20, maxWidth: 520 }}>
        <h2 style={{ fontSize: 16, marginBottom: 10 }}>Two-factor authentication</h2>
        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginBottom: 14 }}>
          status: {user.totp_enabled ? "enabled" : "disabled"}
        </p>

        {!user.totp_enabled && !setup && (
          <button className="sp-btn" onClick={startSetup}>
            Set up 2FA
          </button>
        )}

        {setup && (
          <div>
            <p className="sp-mono" style={{ fontSize: 12, wordBreak: "break-all", marginBottom: 10 }}>
              secret: {setup.secret}
            </p>
            <div className="sp-field">
              <label className="sp-label">Enter the 6-digit code from your app</label>
              <input className="sp-input sp-mono" value={code} onChange={(e) => setCode(e.target.value)} maxLength={6} />
            </div>
            <button className="sp-btn sp-btn--primary" onClick={confirm}>
              Confirm
            </button>
          </div>
        )}

        {user.totp_enabled && (
          <div>
            <div className="sp-field">
              <label className="sp-label">Enter your current code to disable</label>
              <input className="sp-input sp-mono" value={code} onChange={(e) => setCode(e.target.value)} maxLength={6} />
            </div>
            <button className="sp-btn sp-btn--danger" onClick={disable}>
              Disable 2FA
            </button>
          </div>
        )}

        {message && (
          <p className="sp-mono" style={{ fontSize: 12, marginTop: 12 }}>
            {message}
          </p>
        )}
      </div>

      <SessionsCard />
      <ApiKeysCard />

      <Link to="/account/theme" className="sp-btn">
        Theme builder →
      </Link>
    </div>
  );
}

function ChangePasswordCard() {
  const user = useAuthStore((s) => s.user);
  const setSession = useAuthStore((s) => s.setSession);

  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [note, setNote] = useState<{ text: string; ok: boolean } | null>(null);

  const change = useMutation({
    mutationFn: () => accountApi.changePassword(current, next),
    onSuccess: async () => {
      setCurrent("");
      const email = user?.email;
      const pw = next;
      setNext("");
      setConfirm("");
      // Changing the password revokes every session server-side. Re-authenticate
      // seamlessly so the current tab stays logged in; if that fails (e.g. 2FA is
      // now required), fall back to telling them to sign in again.
      try {
        if (email) setSession(await authApi.login(email, pw));
        setNote({ text: "Password changed. Other sessions were signed out.", ok: true });
      } catch {
        setNote({ text: "Password changed. You've been signed out everywhere — please log in again.", ok: true });
      }
    },
    onError: (err) =>
      setNote({
        text:
          err instanceof ApiError && err.code === "wrong_password"
            ? "Current password is incorrect."
            : err instanceof ApiError
              ? err.message
              : "Could not change password.",
        ok: false,
      }),
  });

  const tooShort = next.length > 0 && next.length < 8;
  const mismatch = confirm.length > 0 && next !== confirm;
  const canSubmit = current.length > 0 && next.length >= 8 && next === confirm && !change.isPending;

  return (
    <div className="sp-surface sp-card" style={{ marginBottom: 20, maxWidth: 520 }}>
      <h2 style={{ fontSize: 16, marginBottom: 10 }}>Change password</h2>
      <div className="sp-field">
        <label className="sp-label">Current password</label>
        <input
          className="sp-input"
          type="password"
          autoComplete="current-password"
          value={current}
          onChange={(e) => setCurrent(e.target.value)}
        />
      </div>
      <div className="sp-field">
        <label className="sp-label">New password</label>
        <input
          className="sp-input"
          type="password"
          autoComplete="new-password"
          value={next}
          onChange={(e) => setNext(e.target.value)}
        />
        {tooShort && <p className="sp-error" style={{ marginTop: 6 }}>Use at least 8 characters.</p>}
      </div>
      <div className="sp-field">
        <label className="sp-label">Confirm new password</label>
        <input
          className="sp-input"
          type="password"
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
        />
        {mismatch && <p className="sp-error" style={{ marginTop: 6 }}>Passwords don't match.</p>}
      </div>
      <button className="sp-btn sp-btn--primary" onClick={() => change.mutate()} disabled={!canSubmit}>
        {change.isPending ? "Updating…" : "Update password"}
      </button>
      {note && (
        <p
          className="sp-mono"
          style={{ fontSize: 12, marginTop: 12, color: note.ok ? "var(--sp-accent)" : "#ff9b9b" }}
        >
          {note.text}
        </p>
      )}
    </div>
  );
}

function formatDate(iso?: string): string {
  if (!iso) return "—";
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "—";
  return new Date(t).toLocaleString();
}

function SessionsCard() {
  const queryClient = useQueryClient();
  const refreshToken = useAuthStore((s) => s.refreshToken);

  const { data: sessions, isError } = useQuery({
    queryKey: ["account", "sessions"],
    queryFn: () => accountApi.listSessions(refreshToken ?? undefined),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["account", "sessions"] });

  const revoke = useMutation({ mutationFn: (id: string) => accountApi.revokeSession(id), onSuccess: invalidate });
  const revokeOthers = useMutation({
    mutationFn: () => accountApi.revokeOtherSessions(refreshToken ?? ""),
    onSuccess: invalidate,
  });

  const list = sessions ?? [];
  const hasOthers = list.some((s: Session) => !s.current);

  return (
    <div className="sp-surface sp-card" style={{ marginBottom: 20 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
        <div>
          <h2 style={{ fontSize: 16, margin: 0 }}>Active sessions</h2>
          <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", margin: "4px 0 0" }}>
            Every device or browser currently signed in to your account.
          </p>
        </div>
        <button
          className="sp-btn sp-btn--sm sp-btn--danger"
          onClick={() => revokeOthers.mutate()}
          disabled={!refreshToken || !hasOthers || revokeOthers.isPending}
          title={!refreshToken ? "Can't identify this device's session" : undefined}
        >
          {revokeOthers.isPending ? "Signing out…" : "Sign out others"}
        </button>
      </div>

      {isError && <p className="sp-error">failed to load sessions</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Signed in</th>
            <th>Expires</th>
            <th></th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {list.map((s: Session) => (
            <tr key={s.id}>
              <td className="sp-mono">{formatDate(s.created_at)}</td>
              <td className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
                {formatDate(s.expires_at)}
              </td>
              <td>
                {s.current && (
                  <span className="sp-badge sp-badge--running">this device</span>
                )}
              </td>
              <td style={{ textAlign: "right" }}>
                {!s.current && (
                  <button
                    className="sp-btn sp-btn--sm sp-btn--danger"
                    onClick={() => revoke.mutate(s.id)}
                    disabled={revoke.isPending}
                  >
                    Revoke
                  </button>
                )}
              </td>
            </tr>
          ))}
          {list.length === 0 && (
            <tr>
              <td colSpan={4} className="sp-mono">
                no active sessions
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function ApiKeysCard() {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [freshKey, setFreshKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const { data: keys, isError } = useQuery({
    queryKey: ["account", "apikeys"],
    queryFn: accountApi.listApiKeys,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["account", "apikeys"] });

  const create = useMutation({
    mutationFn: () => accountApi.createApiKey(name.trim() || "API key"),
    onSuccess: (res) => {
      setFreshKey(res.key);
      setCopied(false);
      setName("");
      invalidate();
    },
  });
  const remove = useMutation({ mutationFn: (id: string) => accountApi.deleteApiKey(id), onSuccess: invalidate });

  const list = keys ?? [];

  return (
    <div className="sp-surface sp-card" style={{ marginBottom: 20 }}>
      <h2 style={{ fontSize: 16, margin: "0 0 4px" }}>API keys</h2>
      <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", margin: "0 0 14px" }}>
        Use a key as a <code>Bearer</code> token to drive the API from scripts. Keys carry your full account access.
      </p>

      <div style={{ display: "flex", gap: 8, marginBottom: 14, flexWrap: "wrap" }}>
        <input
          className="sp-input"
          placeholder="Key name (e.g. deploy-bot)"
          value={name}
          onChange={(e) => setName(e.target.value)}
          style={{ flex: 1, minWidth: 200 }}
        />
        <button className="sp-btn sp-btn--primary" onClick={() => create.mutate()} disabled={create.isPending}>
          {create.isPending ? "Creating…" : "+ Create key"}
        </button>
      </div>

      {freshKey && (
        <div
          className="sp-surface"
          style={{ padding: 12, marginBottom: 14, border: "1px solid var(--sp-accent)" }}
        >
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
            <p className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", margin: 0 }}>
              Copy this now — it won't be shown again.
            </p>
            <button
              className="sp-btn sp-btn--sm"
              onClick={() => {
                setFreshKey(null);
                setCopied(false);
              }}
              title="Hide the key"
            >
              Done
            </button>
          </div>
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <code className="sp-mono" style={{ fontSize: 13, wordBreak: "break-all", flex: 1 }}>
              {freshKey}
            </code>
            <button
              className="sp-btn sp-btn--sm"
              onClick={() => {
                copyText(freshKey).then((ok) => {
                  setCopied(ok);
                  if (ok) setTimeout(() => setCopied(false), 1500);
                });
              }}
            >
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
        </div>
      )}

      {isError && <p className="sp-error">failed to load API keys</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Created</th>
            <th>Last used</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {list.map((k: ApiKey) => (
            <tr key={k.id}>
              <td>{k.name || "—"}</td>
              <td className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
                {formatDate(k.created_at)}
              </td>
              <td className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
                {k.last_used_at ? formatDate(k.last_used_at) : "never"}
              </td>
              <td style={{ textAlign: "right" }}>
                <button
                  className="sp-btn sp-btn--sm sp-btn--danger"
                  onClick={() => remove.mutate(k.id)}
                  disabled={remove.isPending}
                >
                  Revoke
                </button>
              </td>
            </tr>
          ))}
          {list.length === 0 && (
            <tr>
              <td colSpan={4} className="sp-mono">
                no API keys yet
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
