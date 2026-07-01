import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { authApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { ApiError } from "../lib/api";

export function LoginPage() {
  const navigate = useNavigate();
  const setSession = useAuthStore((s) => s.setSession);
  const { data: registrationStatus } = useQuery({ queryKey: ["registration-status"], queryFn: authApi.registrationStatus });

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [needsTotp, setNeedsTotp] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const tokens = await authApi.login(email, password, needsTotp ? totpCode : undefined);
      setSession(tokens);
      navigate("/");
    } catch (err) {
      if (err instanceof ApiError && err.code === "totp_required") {
        setNeedsTotp(true);
        setError("Enter your two-factor code to continue.");
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Something went wrong.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="sp-auth-page">
      <form className="sp-surface sp-auth-card" onSubmit={handleSubmit}>
        <h1 style={{ fontSize: 32, marginBottom: 4 }}>Sky Panel</h1>
        <p className="sp-mono" style={{ color: "var(--sp-text-muted)", fontSize: 12, marginBottom: 24 }}>
          sign in
        </p>

        <div className="sp-field">
          <label className="sp-label">Email</label>
          <input
            className="sp-input"
            type="email"
            name="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoFocus
          />
        </div>
        <div className="sp-field">
          <label className="sp-label">Password</label>
          <input
            className="sp-input"
            type="password"
            name="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>

        {needsTotp && (
          <div className="sp-field">
            <label className="sp-label">Two-factor code</label>
            <input className="sp-input sp-mono" value={totpCode} onChange={(e) => setTotpCode(e.target.value)} maxLength={6} autoFocus />
          </div>
        )}

        {error && <p className="sp-error">{error}</p>}

        <button className="sp-btn sp-btn--primary" type="submit" disabled={loading} style={{ width: "100%", marginTop: 8 }}>
          {loading ? "Signing in…" : "Sign in"}
        </button>

        {registrationStatus?.enabled !== false && (
          <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 18, textAlign: "center" }}>
            No account? <Link to="/register">Register</Link>
          </p>
        )}
      </form>
    </div>
  );
}
