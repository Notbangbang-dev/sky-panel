import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";

export function AdminSettingsTab() {
  const queryClient = useQueryClient();
  const { data: settings } = useQuery({ queryKey: ["admin", "settings"], queryFn: adminApi.getSettings });

  const [key, setKey] = useState("");
  const [value, setValue] = useState("");

  const setSetting = useMutation({
    mutationFn: () => adminApi.setSetting(key, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "settings"] });
      setKey("");
      setValue("");
    },
  });

  return (
    <div>
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
