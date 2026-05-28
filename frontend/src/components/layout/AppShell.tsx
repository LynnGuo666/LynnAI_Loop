import { useState } from "react";
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

function SidebarContent({ onNavigate }: { onNavigate?: () => void }) {
  const logout = useAuthStore((s) => s.logout);
  return (
    <>
      <div className="px-5 py-5 text-lg font-bold tracking-tight text-[var(--loop-primary)]">
        Loop
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

  return (
    <div className="flex min-h-screen">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex w-56 shrink-0 border-r border-[var(--loop-border)] bg-[var(--loop-card)] flex-col h-screen sticky top-0">
        <SidebarContent />
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
              <SidebarContent onNavigate={() => setDrawerOpen(false)} />
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
        </header>

        <main className="flex-1 p-4 md:p-6 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
