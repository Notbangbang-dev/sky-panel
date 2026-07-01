import { useQuery } from "@tanstack/react-query";
import { serversApi } from "../../lib/endpoints";

function formatWhen(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

export function ActivityTab({ serverId }: { serverId: string }) {
  const { data: entries, isError } = useQuery({
    queryKey: ["servers", serverId, "activity"],
    queryFn: () => serversApi.activity(serverId),
  });

  if (isError) return <p className="sp-mono">failed to load activity</p>;

  return (
    <table className="sp-table">
      <thead>
        <tr>
          <th>When</th>
          <th>Action</th>
          <th>Details</th>
        </tr>
      </thead>
      <tbody>
        {entries?.map((e, i) => (
          <tr key={i}>
            <td className="sp-mono">{formatWhen(e.created_at)}</td>
            <td className="sp-mono">{e.action}</td>
            <td className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
              {e.metadata || "—"}
            </td>
          </tr>
        ))}
        {entries?.length === 0 && (
          <tr>
            <td colSpan={3} className="sp-mono">
              no activity recorded yet
            </td>
          </tr>
        )}
      </tbody>
    </table>
  );
}
