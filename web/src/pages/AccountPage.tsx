import { useState } from "react";
import { Link } from "react-router-dom";
import { authApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { ApiError } from "../lib/api";
import type { TotpSetup } from "../types/api";

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

      <div className="sp-surface sp-card" style={{ marginBottom: 20, maxWidth: 480 }}>
        <p className="sp-label">Username</p>
        <p style={{ marginBottom: 14 }}>{user.username}</p>
        <p className="sp-label">Email</p>
        <p>{user.email}</p>
      </div>

      <div className="sp-surface sp-card" style={{ marginBottom: 20, maxWidth: 480 }}>
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

      <Link to="/account/theme" className="sp-btn">
        Theme builder →
      </Link>
    </div>
  );
}
