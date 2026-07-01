import { useQuery } from "@tanstack/react-query";
import { nodesApi } from "../lib/endpoints";

export function NodesPage() {
  const { data: nodes, isLoading } = useQuery({ queryKey: ["nodes"], queryFn: nodesApi.list });

  return (
    <div>
      <h1 className="sp-page-title">Nodes</h1>
      <p className="sp-mono" style={{ color: "var(--sp-text-muted)", marginBottom: 20 }}>
        every node currently registered on this panel — pick one when creating a server
      </p>

      {isLoading && <p className="sp-mono">loading…</p>}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Address</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {nodes?.map((node) => (
            <tr key={node.id}>
              <td>{node.name}</td>
              <td className="sp-mono">{node.address}</td>
              <td>
                <span
                  className="sp-mono"
                  style={{ display: "inline-flex", alignItems: "center", gap: 6, color: node.connected ? "var(--sp-accent)" : "var(--sp-text-muted)" }}
                >
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: "50%",
                      background: node.connected ? "var(--sp-accent)" : "var(--sp-text-muted)",
                      display: "inline-block",
                    }}
                  />
                  {node.connected ? "online" : "offline"}
                </span>
              </td>
            </tr>
          ))}
          {nodes?.length === 0 && (
            <tr>
              <td colSpan={3} className="sp-mono">
                no nodes registered yet — an admin needs to add one first
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
