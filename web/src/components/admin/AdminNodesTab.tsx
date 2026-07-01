import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "../../lib/endpoints";

export function AdminNodesTab() {
  const queryClient = useQueryClient();
  const { data: nodes } = useQuery({ queryKey: ["admin", "nodes"], queryFn: adminApi.listNodes });

  const [name, setName] = useState("");
  const [address, setAddress] = useState("");
  const [issuedToken, setIssuedToken] = useState<string | null>(null);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin", "nodes"] });

  const create = useMutation({
    mutationFn: () => adminApi.createNode(name, address),
    onSuccess: (result) => {
      setIssuedToken(result.node_token);
      setName("");
      setAddress("");
      invalidate();
    },
  });
  const remove = useMutation({ mutationFn: (id: string) => adminApi.deleteNode(id), onSuccess: invalidate });

  return (
    <div>
      <form
        className="sp-surface sp-card"
        style={{ marginBottom: 20, maxWidth: 420 }}
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate();
        }}
      >
        <div className="sp-field">
          <label className="sp-label">Name</label>
          <input className="sp-input" value={name} onChange={(e) => setName(e.target.value)} required />
        </div>
        <div className="sp-field">
          <label className="sp-label">Address</label>
          <input className="sp-input" value={address} onChange={(e) => setAddress(e.target.value)} placeholder="1.2.3.4" required />
        </div>
        <button className="sp-btn sp-btn--primary" type="submit">
          Register node
        </button>
      </form>

      {issuedToken && (
        <div className="sp-surface sp-card" style={{ marginBottom: 20, maxWidth: 500 }}>
          <p className="sp-label">Node token (shown once — paste into SKY_NODE_TOKEN on the node)</p>
          <p className="sp-mono" style={{ wordBreak: "break-all" }}>
            {issuedToken}
          </p>
        </div>
      )}

      <table className="sp-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Address</th>
            <th>Docker socket</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {nodes?.map((n) => (
            <tr key={n.id}>
              <td>{n.name}</td>
              <td className="sp-mono">{n.address}</td>
              <td className="sp-mono">{n.docker_socket}</td>
              <td>
                <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => remove.mutate(n.id)}>
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
