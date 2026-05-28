import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth";
import { healthz } from "../api/client";
import { Input, Button } from "@heroui/react";

export function LoginPage() {
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const setAuth = useAuthStore((s) => s.setToken);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.SyntheticEvent) => {
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
          <h1 className="text-3xl font-bold text-primary">Loop</h1>
          <p className="text-sm text-default-500 mt-2">API 密钥轮换代理</p>
        </div>
        <Input
          type="password"
          value={token}
          onValueChange={setToken}
          placeholder="输入管理员令牌"
          autoFocus
          size="lg"
        />
        {error && <p className="text-sm text-danger text-center">{error}</p>}
        <Button
          type="submit"
          color="primary"
          size="lg"
          className="w-full"
          isLoading={loading}
          isDisabled={!token}
        >
          {loading ? "验证中..." : "登录"}
        </Button>
      </form>
    </div>
  );
}
