import { useEffect, useState } from "react";
import { getSettings, updateSettings } from "../api/client";
import {
  Card,
  CardBody,
  Input,
  Switch,
  Button,
} from "@heroui/react";

export function SettingsPage() {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");

  useEffect(() => { getSettings().then(setSettings).catch(() => {}); }, []);

  const handleSave = async () => {
    setSaving(true);
    setMsg("");
    try {
      const updated = await updateSettings(settings);
      setSettings(updated);
      setMsg("已保存");
    } catch {
      setMsg("保存失败");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 2000);
    }
  };

  return (
    <div className="space-y-4 md:space-y-6 max-w-xl">
      <h1 className="text-xl md:text-2xl font-bold">设置</h1>
      <div className="space-y-5">
        <Card>
          <CardBody className="p-4 flex items-start justify-between gap-3">
            <div>
              <div className="text-sm font-medium">恢复探测</div>
              <div className="text-xs text-default-500 mt-1">定期探测已停用的密钥，恢复可用后自动重新启用</div>
            </div>
            <Switch
              isSelected={settings.recovery_probe_enabled === "true"}
              onValueChange={(v: boolean) => setSettings({ ...settings, recovery_probe_enabled: String(v) })}
            />
          </CardBody>
        </Card>
        <Card>
          <CardBody className="p-4">
            <div className="text-sm font-medium">自动停用阈值</div>
            <div className="text-xs text-default-500 mt-1 mb-3">密钥连续失败达到该次数后会被自动停用</div>
            <Input
              type="number"
              value={settings.auto_disable_threshold || "5"}
              onValueChange={(v) => setSettings({ ...settings, auto_disable_threshold: v })}
              className="w-32"
            />
          </CardBody>
        </Card>
      </div>
      <div className="flex items-center gap-3">
        <Button
          color="primary"
          onPress={handleSave}
          isLoading={saving}
        >
          {saving ? "保存中..." : "保存更改"}
        </Button>
        {msg && <span className="text-sm text-success">{msg}</span>}
      </div>
    </div>
  );
}
