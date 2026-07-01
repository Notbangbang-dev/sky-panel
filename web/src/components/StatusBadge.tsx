import type { ServerStatus } from "../types/api";

const VARIANT: Record<ServerStatus, string> = {
  running: "sp-badge--running",
  installing: "",
  offline: "sp-badge--offline",
  stopping: "sp-badge--offline",
  errored: "sp-badge--errored",
};

export function StatusBadge({ status }: { status: ServerStatus }) {
  return <span className={`sp-badge ${VARIANT[status]}`}>{status}</span>;
}
