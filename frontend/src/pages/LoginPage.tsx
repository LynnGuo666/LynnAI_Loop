import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth";
import { healthz } from "../api/client";

export function LoginPage() {
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const setAuth = useAuthStore((s) => s.setToken);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      localStorage.setItem("loop_token", token);
      await healthz();
      setAuth(token);
      navigate("/");
    } catch {
      localStorage.removeItem("loop_token");
      setError("管理员令牌无效");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center">
      <form onSubmit={handleSubmit} className="w-full max-w-sm space-y-6 p-8">
        <div className="text-center">
          <h1 className="text-3xl font-bold text-[var(--loop-primary)]">Loop</h1>
          <p className="text-sm text-[var(--loop-muted)] mt-2">API 密钥轮换代理</p>
        </div>
        <div>
          <input
            type="password"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="输入管理员令牌"
            className="w-full px-4 py-3 rounded-xl bg-[var(--loop-card)] border border-[var(--loop-border)] text-[var(--loop-text)] placeholder:text-[var(--loop-muted)] focus:outline-none focus:border-[var(--loop-primary)] transition"
            autoFocus
          />
        </div>
        {error && <p className="text-sm text-red-400 text-center">{error}</p>}
        <button
          type="submit"
          disabled={loading || !token}
          className="w-full py-3 rounded-xl bg-[var(--loop-primary)] text-white font-medium hover:opacity-90 transition disabled:opacity-40"
        >
          {loading ? "验证中..." : "登录"}
        </button>
      </form>
    </div>
  );
}
