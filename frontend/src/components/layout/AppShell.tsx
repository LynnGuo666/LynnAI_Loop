import { NavLink, Outlet } from "react-router-dom";
import { useAuthStore } from "../../stores/auth";

const links = [
  { to: "/", label: "仪表盘" },
  { to: "/channels", label: "渠道" },
  { to: "/keys", label: "密钥" },
  { to: "/usage", label: "用量" },
  { to: "/settings", label: "设置" },
];

export function Sidebar() {
  const logout = useAuthStore((s) => s.logout);
  return (
    <aside className="w-56 shrink-0 border-r border-[var(--loop-border)] bg-[var(--loop-card)] flex flex-col h-screen sticky top-0">
      <div className="px-5 py-5 text-lg font-bold tracking-tight text-[var(--loop-primary)]">
        Loop
      </div>
      <nav className="flex-1 px-3 space-y-1">
        {links.map((l) => (
          <NavLink
            key={l.to}
            to={l.to}
            end={l.to === "/"}
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
      <button
        onClick={logout}
        className="m-3 px-3 py-2 text-sm rounded-lg text-[var(--loop-muted)] hover:text-red-400 hover:bg-red-400/10 transition"
      >
        退出登录
      </button>
    </aside>
  );
}

export function AppShell() {
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <main className="flex-1 p-6 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}
