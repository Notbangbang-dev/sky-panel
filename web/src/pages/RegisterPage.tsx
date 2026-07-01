import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { authApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { ApiError } from "../lib/api";

export function RegisterPage() {
  const navigate = useNavigate();
  const setSession = useAuthStore((s) => s.setSession);

  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const tokens = await authApi.register(email, username, password);
      setSession(tokens);
      navigate("/");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="sp-auth-page">
      <form className="sp-surface sp-auth-card" onSubmit={handleSubmit}>
        <h1 style={{ fontSize: 32, marginBottom: 4 }}>Sky Panel</h1>
        <p className="sp-mono" style={{ color: "var(--sp-text-muted)", fontSize: 12, marginBottom: 24 }}>
          create account
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
          <label className="sp-label">Username</label>
          <input className="sp-input" name="username" value={username} onChange={(e) => setUsername(e.target.value)} required />
        </div>
        <div className="sp-field">
          <label className="sp-label">Password</label>
          <input
            className="sp-input"
            type="password"
            name="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={8}
            required
          />
        </div>

        {error && <p className="sp-error">{error}</p>}

        <button className="sp-btn sp-btn--primary" type="submit" disabled={loading} style={{ width: "100%", marginTop: 8 }}>
          {loading ? "Creating…" : "Create account"}
        </button>

        <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 18, textAlign: "center" }}>
          Already have an account? <Link to="/login">Sign in</Link>
        </p>
      </form>
    </div>
  );
}
