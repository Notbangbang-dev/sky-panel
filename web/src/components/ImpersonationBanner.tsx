import { useAuthStore } from "../lib/authStore";

// Shown across the top while an admin is "viewing as" another user, so it's
// always obvious whose account is being acted on, with a one-click exit.
export function ImpersonationBanner() {
  const impersonating = useAuthStore((s) => s.impersonating);
  const user = useAuthStore((s) => s.user);
  const endImpersonation = useAuthStore((s) => s.endImpersonation);

  if (!impersonating) return null;

  return (
    <div className="sp-impersonation">
      <span className="sp-mono" style={{ fontSize: 12 }}>
        👁 Viewing as <strong>{user?.username}</strong> — actions affect their account
      </span>
      <button
        className="sp-btn sp-btn--sm"
        onClick={() => {
          endImpersonation();
          // Hard-navigate so React Query caches reset to the restored admin session.
          window.location.href = "/admin";
        }}
      >
        Exit view-as
      </button>
    </div>
  );
}
