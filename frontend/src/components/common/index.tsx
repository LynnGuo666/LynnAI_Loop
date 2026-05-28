import { useEffect, useState } from "react";
import {
  Card,
  CardBody,
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Input,
  Select,
  SelectItem,
  Textarea,
  Button,
} from "@heroui/react";
import type { ReactNode } from "react";
import type { APIKey, Channel, KeyImportItem, KeyImportResponse } from "../../types";

interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
  color?: string;
}

export function StatCard({ label, value, sub, color }: StatCardProps) {
  return (
    <Card>
      <CardBody className="p-5">
        <div className="text-xs text-default-500 uppercase tracking-wider mb-1">{label}</div>
        <div className={`text-xl md:text-2xl font-bold ${color || "text-foreground"}`}>{value}</div>
        {sub && <div className="text-xs text-default-500 mt-1">{sub}</div>}
      </CardBody>
    </Card>
  );
}

interface Column<T> {
  key: string;
  label: string;
  render?: (row: T) => ReactNode;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  empty?: string;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function DataTable<T extends Record<string, any>>({ columns, data, empty }: DataTableProps<T>) {
  return (
    <Table aria-label="Data table">
      <TableHeader>
        {columns.map((c) => (
          <TableColumn key={c.key}>{c.label}</TableColumn>
        ))}
      </TableHeader>
      <TableBody emptyContent={empty || "暂无数据"}>
        {data.map((row, i) => (
          <TableRow key={i}>
            {columns.map((c) => (
              <TableCell key={c.key}>{c.render ? c.render(row) : String(row[c.key] ?? "")}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
  danger?: boolean;
}

export function ConfirmDialog({ open, title, message, onConfirm, onCancel, danger }: ConfirmDialogProps) {
  return (
    <Modal isOpen={open} onOpenChange={(isOpen) => !isOpen && onCancel()}>
      <ModalContent>
        <ModalHeader>{title}</ModalHeader>
        <ModalBody>
          <p className="text-default-500">{message}</p>
        </ModalBody>
        <ModalFooter>
          <Button variant="flat" onPress={onCancel}>
            取消
          </Button>
          <Button color={danger ? "danger" : "primary"} onPress={onConfirm}>
            确认
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}

interface KeyFormModalProps {
  title: string;
  channels?: Channel[];
  fixedChannelId?: number;
  initialKey?: APIKey | null;
  onClose: () => void;
  onSubmit: (data: { channelId: number; keyValue: string; alias: string }) => Promise<void> | void;
}

export function KeyFormModal({ title, channels = [], fixedChannelId, initialKey, onClose, onSubmit }: KeyFormModalProps) {
  const [channelId, setChannelId] = useState<number | "">(fixedChannelId ?? initialKey?.channel_id ?? "");
  const [keyValue, setKeyValue] = useState(initialKey?.key_value ?? "");
  const [alias, setAlias] = useState(initialKey?.alias ?? "");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const canChooseChannel = !initialKey && fixedChannelId === undefined;

  useEffect(() => {
    setChannelId(fixedChannelId ?? initialKey?.channel_id ?? "");
    setKeyValue(initialKey?.key_value ?? "");
    setAlias(initialKey?.alias ?? "");
    setError("");
    setSaving(false);
  }, [fixedChannelId, initialKey]);

  const handleSubmit = async () => {
    const trimmedKey = keyValue.trim();
    if (channelId === "") {
      setError("请选择渠道");
      return;
    }
    if (!trimmedKey) {
      setError("请填写 API Key");
      return;
    }

    setSaving(true);
    setError("");
    try {
      await onSubmit({ channelId: Number(channelId), keyValue: trimmedKey, alias: alias.trim() });
      onClose();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      setError(msg.includes("already exists") || msg.includes("409") ? "这个 API Key 已存在，不能重复添加" : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal isOpen onOpenChange={(isOpen) => !isOpen && onClose()} size="md" scrollBehavior="inside">
      <ModalContent>
        <ModalHeader>{title}</ModalHeader>
        <ModalBody className="gap-4">
          {canChooseChannel && (
            <Select
              label="渠道"
              placeholder="选择渠道"
              selectedKeys={channelId !== "" ? new Set([String(channelId)]) : new Set()}
              onSelectionChange={(keys) => {
                const key = Array.from(keys)[0];
                setChannelId(key ? Number(key) : "");
              }}
            >
              {channels.map((channel) => (
                <SelectItem key={String(channel.id)}>{channel.name}</SelectItem>
              ))}
            </Select>
          )}
          <Input
            label="别名"
            placeholder="别名（可选）"
            value={alias}
            onValueChange={setAlias}
          />
          <Input
            label="API Key"
            placeholder="sk-ant-..."
            value={keyValue}
            onValueChange={setKeyValue}
            className="font-mono"
          />
          {error && <div className="text-sm text-danger">{error}</div>}
        </ModalBody>
        <ModalFooter>
          <Button variant="flat" onPress={onClose}>
            取消
          </Button>
          <Button
            color="primary"
            onPress={handleSubmit}
            isLoading={saving}
            isDisabled={!keyValue.trim() || channelId === ""}
          >
            保存
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}

interface KeyImportModalProps {
  channels?: Channel[];
  fixedChannelId?: number;
  onClose: () => void;
  onImport: (data: { channelId?: number; keys: KeyImportItem[] }) => Promise<KeyImportResponse>;
}

export function KeyImportModal({ channels = [], fixedChannelId, onClose, onImport }: KeyImportModalProps) {
  const [channelId, setChannelId] = useState<number | "">(fixedChannelId ?? "");
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [result, setResult] = useState<KeyImportResponse | null>(null);
  const [saving, setSaving] = useState(false);
  const canChooseChannel = fixedChannelId === undefined;

  const handleImport = async () => {
    let keys: KeyImportItem[];
    try {
      keys = parseImportKeys(text);
    } catch (err) {
      setError(err instanceof Error ? err.message : "导入内容格式不正确");
      return;
    }
    if (keys.length === 0) {
      setError("没有可导入的 Key");
      return;
    }
    if (canChooseChannel && channelId === "" && keys.some((key) => !key.channel_id)) {
      setError("请选择渠道，或使用包含 channel_id 的 JSON");
      return;
    }

    setSaving(true);
    setError("");
    try {
      const imported = await onImport({
        channelId: channelId === "" ? undefined : Number(channelId),
        keys,
      });
      setResult(imported);
    } catch {
      setError("导入失败");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal isOpen onOpenChange={(isOpen) => !isOpen && onClose()} size="xl" scrollBehavior="inside">
      <ModalContent>
        <ModalHeader>导入 API 密钥</ModalHeader>
        <ModalBody className="gap-4">
          {canChooseChannel && (
            <Select
              label="渠道"
              placeholder="从 JSON 中读取渠道，或选择统一导入渠道"
              selectedKeys={channelId !== "" ? new Set([String(channelId)]) : new Set()}
              onSelectionChange={(keys) => {
                const key = Array.from(keys)[0];
                setChannelId(key ? Number(key) : "");
              }}
            >
              {channels.map((channel) => (
                <SelectItem key={String(channel.id)}>{channel.name}</SelectItem>
              ))}
            </Select>
          )}
          <Textarea
            label="导入内容"
            placeholder={'粘贴导出的 JSON，或每行一个 Key\nsk-ant-...\nsk-ant-...'}
            value={text}
            onValueChange={(v) => { setText(v); setError(""); setResult(null); }}
            minRows={6}
            className="font-mono text-xs"
          />
          {error && <div className="text-sm text-danger">{error}</div>}
          {result && (
            <div className="text-sm text-default-500">
              已导入 {result.created} 个，跳过重复 {result.skipped} 个，失败 {result.failed} 个
            </div>
          )}
        </ModalBody>
        <ModalFooter>
          <Button variant="flat" onPress={onClose}>
            关闭
          </Button>
          <Button
            color="primary"
            onPress={handleImport}
            isLoading={saving}
            isDisabled={!text.trim()}
          >
            导入
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}

function parseImportKeys(text: string): KeyImportItem[] {
  const trimmed = text.trim();
  if (!trimmed) return [];

  if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
    const parsed = JSON.parse(trimmed);
    const data = Array.isArray(parsed) ? parsed : parsed.data;
    if (!Array.isArray(data)) {
      throw new Error("JSON 需要是数组，或包含 data 数组");
    }
    return data
      .map((item: Record<string, unknown>) => ({
        channel_id: Number(item.channel_id) || undefined,
        key_value: String(item.key_value || item.key || "").trim(),
        alias: String(item.alias || "").trim(),
        is_active: typeof item.is_active === "boolean" ? item.is_active : undefined,
      }))
      .filter((item: KeyImportItem) => item.key_value);
  }

  return trimmed
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => ({ key_value: line }));
}
