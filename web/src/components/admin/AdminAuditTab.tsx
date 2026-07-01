import { useQuery } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";

export function AdminAuditTab() {
  const { data: entries } = useQuery({ queryKey: ["admin", "audit"], queryFn: adminApi.auditLog });

  return (
    <table className="sp-table">
      <thead>
        <tr>
          <th>Actor</th>
          <th>Action</th>
          <th>Target</th>
          <th>Note</th>
          <th>When</th>
        </tr>
      </thead>
      <tbody>
        {entries?.map((e, i) => (
          <tr key={i}>
            <td className="sp-mono">{e.actor_id.slice(0, 8)}</td>
            <td>{e.action}</td>
            <td className="sp-mono">{e.target}</td>
            <td>{e.metadata}</td>
            <td className="sp-mono">{new Date(e.created_at).toLocaleString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
