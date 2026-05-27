import { useEffect, useState } from "react";
import { getSettings, updateSettings } from "../api/client";

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
      setMsg("Saved");
    } catch {
      setMsg("Failed to save");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 2000);
    }
  };

  return (
    <div className="space-y-6 max-w-xl">
      <h1 className="text-2xl font-bold">Settings</h1>
      <div className="space-y-5">
        <SettingToggle
          label="Recovery Probe"
          description="Periodically probe disabled keys and re-enable when they recover"
          value={settings.recovery_probe_enabled === "true"}
          onChange={(v) => setSettings({ ...settings, recovery_probe_enabled: String(v) })}
        />
        <SettingInput
          label="Auto Disable Threshold"
          description="Number of consecutive failures before a key is automatically disabled"
          value={settings.auto_disable_threshold || "5"}
          onChange={(v) => setSettings({ ...settings, auto_disable_threshold: v })}
          type="number"
        />
      </div>
      <div className="flex items-center gap-3">
        <button onClick={handleSave} disabled={saving}
          className="px-5 py-2.5 rounded-xl bg-[var(--loop-primary)] text-white text-sm font-medium hover:opacity-90 transition disabled:opacity-40">
          {saving ? "Saving..." : "Save Changes"}
        </button>
        {msg && <span className="text-sm text-green-400">{msg}</span>}
      </div>
    </div>
  );
}

function SettingToggle({ label, description, value, onChange }: { label: string; description: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-start justify-between p-4 rounded-xl border border-[var(--loop-border)] bg-[var(--loop-card)]">
      <div>
        <div className="text-sm font-medium">{label}</div>
        <div className="text-xs text-[var(--loop-muted)] mt-1">{description}</div>
      </div>
      <button
        onClick={() => onChange(!value)}
        className={`relative w-11 h-6 rounded-full transition ${value ? "bg-[var(--loop-primary)]" : "bg-[var(--loop-border)]"}`}
      >
        <div className={`absolute top-0.5 w-5 h-5 rounded-full bg-white transition ${value ? "left-5.5" : "left-0.5"}`} />
      </button>
    </div>
  );
}

function SettingInput({ label, description, value, onChange, type }: { label: string; description: string; value: string; onChange: (v: string) => void; type?: string }) {
  return (
    <div className="p-4 rounded-xl border border-[var(--loop-border)] bg-[var(--loop-card)]">
      <div className="text-sm font-medium">{label}</div>
      <div className="text-xs text-[var(--loop-muted)] mt-1 mb-3">{description}</div>
      <input
        type={type || "text"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-32 px-3 py-2 rounded-lg bg-[var(--loop-bg)] border border-[var(--loop-border)] text-[var(--loop-text)] text-sm focus:outline-none focus:border-[var(--loop-primary)]"
      />
    </div>
  );
}
