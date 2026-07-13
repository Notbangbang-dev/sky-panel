import { BrowserRouter, Navigate, Route, Routes, useParams } from "react-router-dom";
import { QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "./lib/api";
import { toast, ToastHost } from "./lib/toast";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { ThemeProvider } from "./lib/ThemeProvider";
import { AppearanceProvider } from "./lib/AppearanceProvider";
import { Background } from "./components/Background";
import { MaintenanceGate } from "./components/MaintenanceGate";
import { AppShell } from "./components/AppShell";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { BroadcastBanner } from "./components/BroadcastBanner";
import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { StatusPage } from "./pages/StatusPage";
import { DashboardPage } from "./pages/DashboardPage";
import { ServersListPage } from "./pages/ServersListPage";
import { ServerDetailPage } from "./pages/ServerDetailPage";
import { ReinstallPage } from "./pages/ReinstallPage";
import { NodesPage } from "./pages/NodesPage";
import { AfkPage } from "./pages/AfkPage";
import { StorePage } from "./pages/StorePage";
import { WalletPage } from "./pages/WalletPage";
import { LeaderboardPage } from "./pages/LeaderboardPage";
import { AchievementsPage } from "./pages/AchievementsPage";
import { AccountPage } from "./pages/AccountPage";
import { ThemeBuilderPage } from "./pages/ThemeBuilderPage";
import { AdminPage } from "./pages/AdminPage";

const queryClient = new QueryClient({
  // Surface background query failures as a toast instead of failing silently.
  // A 401 is handled by the API layer's token refresh, so don't nag about it.
  queryCache: new QueryCache({
    onError: (error) => {
      if (error instanceof ApiError && error.status === 401) return;
      toast.error(error instanceof Error ? error.message : "Something went wrong");
    },
  }),
  defaultOptions: {
    queries: {
      // Don't retry client errors (4xx) — they won't succeed on retry and just
      // delay the error. Retry a couple times for transient 5xx/network faults.
      retry: (failureCount, error) => {
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false;
        return failureCount < 2;
      },
      staleTime: 30_000,
      refetchOnWindowFocus: false,
    },
  },
});

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <ProtectedRoute>
      <AppShell>{children}</AppShell>
    </ProtectedRoute>
  );
}

// Keying the detail page on the server id forces a full remount when navigating
// between servers on the same route, so its console/stats/sparkline buffers
// (and the xterm instance) reset instead of showing the previous server's data.
function ServerDetailRoute() {
  const { id } = useParams<{ id: string }>();
  return <ServerDetailPage key={id} />;
}

function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <AppearanceProvider>
          <ThemeProvider>
            <Background />
            <div className="sp-scanlines" />
            <div className="sp-grain" />
            <ToastHost />
          <BrowserRouter>
            <BroadcastBanner />
            <MaintenanceGate>
              <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/status/:id" element={<StatusPage />} />

            <Route path="/" element={<Shell><DashboardPage /></Shell>} />
            <Route path="/servers" element={<Shell><ServersListPage /></Shell>} />
            <Route path="/servers/:id" element={<Shell><ServerDetailRoute /></Shell>} />
            <Route path="/servers/:id/reinstall" element={<Shell><ReinstallPage /></Shell>} />
            <Route path="/nodes" element={<Shell><NodesPage /></Shell>} />
            <Route path="/wallet" element={<Shell><WalletPage /></Shell>} />
            <Route path="/store" element={<Shell><StorePage /></Shell>} />
            <Route path="/leaderboard" element={<Shell><LeaderboardPage /></Shell>} />
            <Route path="/achievements" element={<Shell><AchievementsPage /></Shell>} />
            <Route path="/afk" element={<Shell><AfkPage /></Shell>} />
            <Route path="/account" element={<Shell><AccountPage /></Shell>} />
            <Route path="/account/theme" element={<Shell><ThemeBuilderPage /></Shell>} />
            <Route
              path="/admin"
              element={
                <ProtectedRoute adminOnly>
                  <AppShell>
                    <AdminPage />
                  </AppShell>
                </ProtectedRoute>
              }
            />

                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </MaintenanceGate>
          </BrowserRouter>
        </ThemeProvider>
      </AppearanceProvider>
    </QueryClientProvider>
    </ErrorBoundary>
  );
}

export default App;
