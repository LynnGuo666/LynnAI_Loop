import { BrowserRouter, Routes, Route, useNavigate } from "react-router-dom";
import { Agentation } from "agentation";
import { HeroUIProvider } from "@heroui/react";
import { ToastProvider } from "@heroui/toast";
import { AppShell } from "./components/layout/AppShell";
import { ProtectedRoute } from "./hooks/useAuth";
import { LoginPage } from "./pages/LoginPage";
import { DashboardPage } from "./pages/DashboardPage";
import { ChannelsPage } from "./pages/ChannelsPage";
import { ChannelDetailPage } from "./pages/ChannelDetailPage";
import { KeysPage } from "./pages/KeysPage";
import { UsagePage } from "./pages/UsagePage";
import { StatisticsPage } from "./pages/StatisticsPage";
import { SettingsPage } from "./pages/SettingsPage";

function AppInner() {
  const navigate = useNavigate();
  return (
    <HeroUIProvider navigate={navigate} locale="zh-CN">
      <ToastProvider placement="top-center" maxVisibleToasts={3} />
      {import.meta.env.DEV && <Agentation />}
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          element={
            <ProtectedRoute>
              <AppShell />
            </ProtectedRoute>
          }
        >
          <Route path="/" element={<DashboardPage />} />
          <Route path="/channels" element={<ChannelsPage />} />
          <Route path="/channels/:id" element={<ChannelDetailPage />} />
          <Route path="/keys" element={<KeysPage />} />
          <Route path="/usage" element={<UsagePage />} />
          <Route path="/statistics" element={<StatisticsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </HeroUIProvider>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AppInner />
    </BrowserRouter>
  );
}
