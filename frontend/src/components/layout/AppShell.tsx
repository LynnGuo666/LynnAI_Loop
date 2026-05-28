import { useEffect, useState } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { useAuthStore } from "../../stores/auth";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerBody,
  Button,
} from "@heroui/react";
import { healthz } from "../../api/client";

const links = [
  { to: "/", label: "仪表盘" },
  { to: "/channels", label: "渠道" },
  { to: "/keys", label: "密钥" },
  { to: "/usage", label: "用量" },
  { to: "/settings", label: "设置" },
];

function NavLinks({ onClick }: { onClick?: () => void }) {
  return (
    <nav className="flex-1 px-3 space-y-1">
      {links.map((l) => (
        <NavLink
          key={l.to}
          to={l.to}
          end={l.to === "/"}
          onClick={onClick}
          className={({ isActive }) =>
            `block px-3 py-2 rounded-lg text-sm transition ${
              isActive
                ? "bg-primary-100 text-primary font-medium"
                : "text-default-500 hover:text-foreground hover:bg-default-100"
            }`
          }
        >
          {l.label}
        </NavLink>
      ))}
    </nav>
  );
}

function SidebarContent({
  onNavigate,
  theme,
  toggleTheme,
}: {
  onNavigate?: () => void;
  theme: Theme;
  toggleTheme: () => void;
}) {
  const logout = useAuthStore((s) => s.logout);
  const [appVersion, setAppVersion] = useState<string>("");
  useEffect(() => {
    healthz().then((r) => setAppVersion(r.version)).catch(() => {});
  }, []);
  return (
    <>
      <div className="flex items-center justify-between px-5 py-5">
        <div className="text-lg font-bold tracking-tight text-primary">Loop</div>
        <ThemeButton theme={theme} onClick={toggleTheme} />
      </div>
      <NavLinks onClick={onNavigate} />
      <Button
        variant="light"
        color="danger"
        onPress={() => { logout(); onNavigate?.(); }}
        className="m-3 justify-start"
      >
        退出登录
      </Button>
      {appVersion && (
        <div className="px-5 pb-3 text-xs text-default-500 opacity-60">v{appVersion}</div>
      )}
    </>
  );
}

export function AppShell() {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="flex min-h-screen">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex w-56 shrink-0 border-r border-divider bg-content1 flex-col h-screen sticky top-0">
        <SidebarContent theme={theme} toggleTheme={toggleTheme} />
      </aside>

      {/* Mobile drawer */}
      <Drawer isOpen={drawerOpen} onOpenChange={setDrawerOpen} placement="left" size="xs">
        <DrawerContent>
          <DrawerHeader className="border-b border-divider">
            <span className="text-lg font-bold tracking-tight text-primary">Loop</span>
          </DrawerHeader>
          <DrawerBody className="p-0">
            <SidebarContent
              theme={theme}
              toggleTheme={toggleTheme}
              onNavigate={() => setDrawerOpen(false)}
            />
          </DrawerBody>
        </DrawerContent>
      </Drawer>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Mobile top bar */}
        <header className="md:hidden flex items-center px-4 py-3 border-b border-divider bg-content1 sticky top-0 z-30">
          <Button
            isIconOnly
            variant="light"
            onPress={() => setDrawerOpen(true)}
            className="text-default-500"
          >
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
              <path d="M3 5h14M3 10h14M3 15h14" />
            </svg>
          </Button>
          <span className="ml-3 text-base font-bold tracking-tight text-primary">Loop</span>
          <div className="ml-auto">
            <ThemeButton theme={theme} onClick={toggleTheme} />
          </div>
        </header>

        <main className="flex-1 p-4 md:p-6 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

type Theme = "dark" | "light";

function getInitialTheme(): Theme {
  const stored = localStorage.getItem("loop_theme");
  if (stored === "light" || stored === "dark") return stored;
  return "dark";
}

function useTheme() {
  const [theme, setTheme] = useState<Theme>(getInitialTheme);

  useEffect(() => {
    document.documentElement.classList.remove("dark", "light");
    document.documentElement.classList.add(theme);
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("loop_theme", theme);
  }, [theme]);

  return {
    theme,
    toggleTheme: () => setTheme((current) => (current === "dark" ? "light" : "dark")),
  };
}

function ThemeButton({ theme, onClick }: { theme: Theme; onClick: () => void }) {
  const isDark = theme === "dark";

  return (
    <Button
      isIconOnly
      variant="bordered"
      size="sm"
      onPress={onClick}
      title={isDark ? "切换到浅色模式" : "切换到深色模式"}
      aria-label={isDark ? "切换到浅色模式" : "切换到深色模式"}
      className="text-default-500"
    >
      {isDark ? (
        <svg width="16" height="16" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round">
          <circle cx="10" cy="10" r="3.5" />
          <path d="M10 1.8v2M10 16.2v2M3.2 3.2l1.4 1.4M15.4 15.4l1.4 1.4M1.8 10h2M16.2 10h2M3.2 16.8l1.4-1.4M15.4 4.6l1.4-1.4" />
        </svg>
      ) : (
        <svg width="16" height="16" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
          <path d="M16.5 11.2A6.5 6.5 0 0 1 8.8 3.5 6.8 6.8 0 1 0 16.5 11.2Z" />
        </svg>
      )}
    </Button>
  );
}
