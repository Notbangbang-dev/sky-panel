import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";

export function AdminSettingsTab() {
  const queryClient = useQueryClient();
  const { data: settings } = useQuery({ queryKey: ["admin", "settings"], queryFn: adminApi.getSettings });

  const [key, setKey] = useState("");
  const [value, setValue] = useState("");

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin", "settings"] });

  const setSetting = useMutation({
    mutationFn: () => adminApi.setSetting(key, value),
    onSuccess: () => {
      invalidate();
      setKey("");
      setValue("");
    },
  });

  // Absent means enabled — matches the backend's default (an upgrade that
  // never opted into disabling registration should never silently lock it).
  const registrationEnabled = settings?.registration_enabled !== "false";
  const toggleRegistration = useMutation({
    mutationFn: (enabled: boolean) => adminApi.setSetting("registration_enabled", enabled ? "true" : "false"),
    onSuccess: invalidate,
  });

  return (
    <div>
      <div className="sp-surface sp-card" style={{ marginBottom: 20, maxWidth: 420 }}>
        <p className="sp-label" style={{ marginBottom: 10 }}>
          Registration
        </p>
        <label className="sp-mono" style={{ display: "flex", gap: 8, alignItems: "center", cursor: "pointer" }}>
          <input
            type="checkbox"
            checked={registrationEnabled}
            onChange={(e) => toggleRegistration.mutate(e.target.checked)}
          />
          Allow new accounts to sign up
        </label>
        {!registrationEnabled && (
          <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 8 }}>
            new signups are currently blocked — existing accounts can still log in
          </p>
        )}
      </div>

      <form
        className="sp-surface sp-card"
        style={{ marginBottom: 20, maxWidth: 420 }}
        onSubmit={(e) => {
          e.preventDefault();
          setSetting.mutate();
        }}
      >
        <div className="sp-field">
          <label className="sp-label">Key</label>
          <input className="sp-input sp-mono" value={key} onChange={(e) => setKey(e.target.value)} required />
        </div>
        <div className="sp-field">
          <label className="sp-label">Value</label>
          <input className="sp-input" value={value} onChange={(e) => setValue(e.target.value)} required />
        </div>
        <button className="sp-btn sp-btn--primary" type="submit">
          Save
        </button>
      </form>

      <table className="sp-table">
        <thead>
          <tr>
            <th>Key</th>
            <th>Value</th>
          </tr>
        </thead>
        <tbody>
          {Object.entries(settings ?? {}).map(([k, v]) => (
            <tr key={k}>
              <td className="sp-mono">{k}</td>
              <td>{v}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
