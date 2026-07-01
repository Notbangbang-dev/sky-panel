import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi, eggsApi } from "../../lib/endpoints";

export function AdminEggsTab() {
  const queryClient = useQueryClient();
  const { data: eggs } = useQuery({ queryKey: ["eggs"], queryFn: eggsApi.list });

  const [name, setName] = useState("");
  const [dockerImage, setDockerImage] = useState("");
  const [startup, setStartup] = useState("");

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["eggs"] });

  const create = useMutation({
    mutationFn: () => adminApi.createEgg({ name, docker_image: dockerImage, startup }),
    onSuccess: () => {
      setName("");
      setDockerImage("");
      setStartup("");
      invalidate();
    },
  });
  const remove = useMutation({ mutationFn: (id: string) => adminApi.deleteEgg(id), onSuccess: invalidate });

  return (
    <div>
      <form
        className="sp-surface sp-card"
        style={{ marginBottom: 20, maxWidth: 480 }}
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
          <label className="sp-label">Docker image</label>
          <input className="sp-input sp-mono" value={dockerImage} onChange={(e) => setDockerImage(e.target.value)} required />
        </div>
        <div className="sp-field">
          <label className="sp-label">Startup command</label>
          <textarea className="sp-textarea sp-mono" value={startup} onChange={(e) => setStartup(e.target.value)} required />
        </div>
        <button className="sp-btn sp-btn--primary" type="submit">
          Create egg
        </button>
      </form>

      <table className="sp-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Image</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {eggs?.map((egg) => (
            <tr key={egg.id}>
              <td>{egg.name}</td>
              <td className="sp-mono">{egg.docker_image}</td>
              <td>
                <button className="sp-btn sp-btn--sm sp-btn--danger" onClick={() => remove.mutate(egg.id)}>
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
