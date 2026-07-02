import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "./lib/ThemeProvider";
import { AnimatedBackground } from "./components/AnimatedBackground";
import { AppShell } from "./components/AppShell";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { BroadcastBanner } from "./components/BroadcastBanner";
import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { DashboardPage } from "./pages/DashboardPage";
import { ServersListPage } from "./pages/ServersListPage";
import { ServerDetailPage } from "./pages/ServerDetailPage";
import { ReinstallPage } from "./pages/ReinstallPage";
import { NodesPage } from "./pages/NodesPage";
import { AfkPage } from "./pages/AfkPage";
import { StorePage } from "./pages/StorePage";
import { WalletPage } from "./pages/WalletPage";
import { AccountPage } from "./pages/AccountPage";
import { ThemeBuilderPage } from "./pages/ThemeBuilderPage";
import { AdminPage } from "./pages/AdminPage";

const queryClient = new QueryClient();

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <ProtectedRoute>
      <AppShell>{children}</AppShell>
    </ProtectedRoute>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AnimatedBackground />
        <div className="sp-scanlines" />
        <div className="sp-grain" />
        <BrowserRouter>
          <BroadcastBanner />
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />

            <Route path="/" element={<Shell><DashboardPage /></Shell>} />
            <Route path="/servers" element={<Shell><ServersListPage /></Shell>} />
            <Route path="/servers/:id" element={<Shell><ServerDetailPage /></Shell>} />
            <Route path="/servers/:id/reinstall" element={<Shell><ReinstallPage /></Shell>} />
            <Route path="/nodes" element={<Shell><NodesPage /></Shell>} />
            <Route path="/wallet" element={<Shell><WalletPage /></Shell>} />
            <Route path="/store" element={<Shell><StorePage /></Shell>} />
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
        </BrowserRouter>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;
