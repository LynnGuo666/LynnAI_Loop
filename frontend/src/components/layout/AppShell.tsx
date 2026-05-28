import { useEffect, useState } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { useAuthStore } from "../../stores/auth";
import { AnimatePresence, motion } from "framer-motion";

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
                ? "bg-[var(--loop-primary)]/15 text-[var(--loop-primary)] font-medium"
                : "text-[var(--loop-muted)] hover:text-[var(--loop-text)] hover:bg-white/5"
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
  return (
    <>
      <div className="flex items-center justify-between px-5 py-5">
        <div className="text-lg font-bold tracking-tight text-[var(--loop-primary)]">Loop</div>
        <ThemeButton theme={theme} onClick={toggleTheme} />
      </div>
      <NavLinks onClick={onNavigate} />
      <button
        onClick={() => { logout(); onNavigate?.(); }}
        className="m-3 px-3 py-2 text-sm rounded-lg text-[var(--loop-muted)] hover:text-red-400 hover:bg-red-400/10 transition"
      >
        退出登录
      </button>
    </>
  );
}

export function AppShell() {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="flex min-h-screen">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex w-56 shrink-0 border-r border-[var(--loop-border)] bg-[var(--loop-card)] flex-col h-screen sticky top-0">
        <SidebarContent theme={theme} toggleTheme={toggleTheme} />
      </aside>

      {/* Mobile drawer */}
      <AnimatePresence>
        {drawerOpen && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm md:hidden"
              onClick={() => setDrawerOpen(false)}
            />
            <motion.aside
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={{ type: "spring", damping: 25, stiffness: 300 }}
              className="fixed inset-y-0 left-0 z-50 w-64 bg-[var(--loop-card)] border-r border-[var(--loop-border)] flex flex-col md:hidden"
            >
              <SidebarContent theme={theme} toggleTheme={toggleTheme} onNavigate={() => setDrawerOpen(false)} />
            </motion.aside>
          </>
        )}
      </AnimatePresence>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Mobile top bar */}
        <header className="md:hidden flex items-center px-4 py-3 border-b border-[var(--loop-border)] bg-[var(--loop-card)] sticky top-0 z-30">
          <button
            onClick={() => setDrawerOpen(true)}
            className="p-2 -ml-2 rounded-lg text-[var(--loop-muted)] hover:text-[var(--loop-text)] hover:bg-white/5 transition"
          >
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
              <path d="M3 5h14M3 10h14M3 15h14" />
            </svg>
          </button>
          <span className="ml-3 text-base font-bold tracking-tight text-[var(--loop-primary)]">Loop</span>
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
    <button
      type="button"
      onClick={onClick}
      title={isDark ? "切换到浅色模式" : "切换到深色模式"}
      aria-label={isDark ? "切换到浅色模式" : "切换到深色模式"}
      className="grid h-8 w-8 place-items-center rounded-lg border border-[var(--loop-border)] text-[var(--loop-muted)] hover:bg-[var(--loop-hover)] hover:text-[var(--loop-text)] transition"
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
    </button>
  );
}
