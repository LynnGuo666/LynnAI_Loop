import { useEffect, useState } from "react";
import { listAllKeys, enableKey, probeKey, probeKeys, listChannels, deleteKey, createKey, updateKey, exportKeys, importKeys } from "../api/client";
import { DataTable, ConfirmDialog, KeyFormModal, KeyImportModal } from "../components/common";
import type { APIKey, Channel, KeyImportItem, KeyProbe, KeyProbeBatchResponse } from "../types";
import {
  Button,
  Chip,
  Select,
  SelectItem,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@heroui/react";

export function KeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [filterChannel, setFilterChannel] = useState<number | "">("");
  const [filterStatus, setFilterStatus] = useState<"all" | "active" | "disabled">("all");
  const [showAddKey, setShowAddKey] = useState(false);
  const [showImportKeys, setShowImportKeys] = useState(false);
  const [editingKey, setEditingKey] = useState<APIKey | null>(null);
  const [delKeyId, setDelKeyId] = useState<number | null>(null);
  const [probeResult, setProbeResult] = useState<KeyProbe | null>(null);
  const [batchProbeResult, setBatchProbeResult] = useState<KeyProbeBatchResponse | null>(null);
  const [probingKeyId, setProbingKeyId] = useState<number | null>(null);
  const [batchProbing, setBatchProbing] = useState(false);
  const [showBatchProbeConfirm, setShowBatchProbeConfirm] = useState(false);

  const load = () => {
    listAllKeys().then(setKeys).catch(() => {});
    listChannels().then(setChannels).catch(() => {});
  };
  useEffect(() => { load(); }, []);

  const channelMap = new Map(channels.map((c) => [c.id, c.name]));

  const filtered = keys.filter((k) => {
    if (filterChannel !== "" && k.channel_id !== filterChannel) return false;
    if (filterStatus === "active" && !k.is_active) return false;
    if (filterStatus === "disabled" && k.is_active) return false;
    return true;
  });

  const handleEnable = async (id: number) => { await enableKey(id); load(); };
  const handleProbe = async (id: number) => {
    setProbingKeyId(id);
    try {
      const r = await probeKey(id);
      setProbeResult(r);
      load();
    } finally {
      setProbingKeyId(null);
    }
  };
  const handleBatchProbe = async () => {
    if (filtered.length === 0) return;
    setShowBatchProbeConfirm(true);
  };
  const executeBatchProbe = async (deleteOn401: boolean) => {
    const ids = filtered.map((k) => k.id);
    if (ids.length === 0) return;
    setShowBatchProbeConfirm(false);
    setBatchProbing(true);
    try {
      const result = await probeKeys(ids, deleteOn401);
      setBatchProbeResult(result);
      load();
    } finally {
      setBatchProbing(false);
    }
  };
  const handleDelete = async () => { if (delKeyId) await deleteKey(delKeyId); setDelKeyId(null); load(); };
  const handleAddKey = async (data: { channelId: number; keyValue: string; alias: string }) => {
    await createKey(data.channelId, { key_value: data.keyValue, alias: data.alias });
    load();
  };
  const handleImportKeys = async (data: { channelId?: number; keys: KeyImportItem[] }) => {
    const result = await importKeys({ channel_id: data.channelId, keys: data.keys });
    load();
    return result;
  };
  const handleExportKeys = async () => {
    const channelId = filterChannel === "" ? undefined : filterChannel;
    const exported = await exportKeys(channelId);
    downloadJSON(exported, `loop-keys${channelId ? `-channel-${channelId}` : ""}.json`);
  };
  const handleUpdateKey = async (data: { keyValue: string; alias: string }) => {
    if (!editingKey) return;
    await updateKey(editingKey.id, { key_value: data.keyValue, alias: data.alias });
    setEditingKey(null);
    load();
  };

  const handleBatchEnable = async () => {
    const disabled = filtered.filter((k) => !k.is_active);
    await Promise.all(disabled.map((k) => enableKey(k.id)));
    load();
  };

  const maskKey = (k: string) => (k.length <= 10 ? "****" : k.slice(0, 7) + "..." + k.slice(-4));

  const columns = [
    { key: "id", label: "ID" },
    {
      key: "channel_id",
      label: "渠道",
      render: (k: APIKey) => channelMap.get(k.channel_id) || `#${k.channel_id}`,
    },
    { key: "alias", label: "别名" },
    {
      key: "key_value",
      label: "密钥",
      render: (k: APIKey) => <span className="font-mono text-xs text-default-500">{maskKey(k.key_value)}</span>,
    },
    {
      key: "is_active",
      label: "状态",
      render: (k: APIKey) => (
        <Chip size="sm" variant="flat" color={k.is_active ? "success" : "danger"}>
          {k.is_active ? "启用" : "停用"}
        </Chip>
      ),
    },
    { key: "consecutive_failures", label: "连续失败" },
    { key: "total_successes", label: "成功次数" },
    {
      key: "actions",
      label: "",
      render: (k: APIKey) => (
        <div className="flex gap-1">
          {!k.is_active && <Button size="sm" variant="light" color="success" onPress={() => handleEnable(k.id)}>启用</Button>}
          <Button size="sm" variant="light" color="primary" onPress={() => setEditingKey(k)}>编辑</Button>
          <Button
            size="sm"
            variant="light"
            isLoading={probingKeyId === k.id}
            isDisabled={batchProbing}
            onPress={() => handleProbe(k.id)}
          >
            探测
          </Button>
          <Button size="sm" variant="light" color="danger" onPress={() => setDelKeyId(k.id)}>删除</Button>
        </div>
      ),
    },
  ];

  const disabledCount = filtered.filter((k) => !k.is_active).length;

  return (
    <div className="space-y-4 md:space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <h1 className="text-xl md:text-2xl font-bold">API 密钥</h1>
        <div className="flex flex-wrap gap-2 sm:gap-3">
          {disabledCount > 0 && (
            <Button color="success" variant="flat" size="sm" onPress={handleBatchEnable}>
              启用全部停用密钥（{disabledCount}）
            </Button>
          )}
          <Button
            variant="bordered"
            size="sm"
            isLoading={batchProbing}
            isDisabled={filtered.length === 0}
            onPress={handleBatchProbe}
          >
            批量探测（{filtered.length}）
          </Button>
          <Button color="primary" onPress={() => setShowAddKey(true)}>
            + 添加密钥
          </Button>
          <Button variant="bordered" onPress={() => setShowImportKeys(true)}>
            导入
          </Button>
          <Button variant="bordered" onPress={handleExportKeys}>
            导出
          </Button>
        </div>
      </div>
      <div className="flex flex-wrap gap-2 md:gap-3">
        <Select
          label="渠道"
          placeholder="全部渠道"
          selectedKeys={filterChannel !== "" ? new Set([String(filterChannel)]) : new Set()}
          onSelectionChange={(keys) => {
            const key = Array.from(keys)[0];
            setFilterChannel(key ? Number(key) : "");
          }}
          className="min-w-[140px]"
        >
          {channels.map((c) => (
            <SelectItem key={String(c.id)}>{c.name}</SelectItem>
          ))}
        </Select>
        <Select
          label="状态"
          placeholder="全部状态"
          selectedKeys={filterStatus !== "all" ? new Set([filterStatus]) : new Set()}
          onSelectionChange={(keys) => {
            const key = Array.from(keys)[0] as string;
            setFilterStatus((key as "all" | "active" | "disabled") || "all");
          }}
          className="min-w-[120px]"
        >
          <SelectItem key="all">全部状态</SelectItem>
          <SelectItem key="active">启用</SelectItem>
          <SelectItem key="disabled">停用</SelectItem>
        </Select>
      </div>
      <DataTable columns={columns} data={filtered} empty="未找到密钥" />
      {showAddKey && (
        <KeyFormModal
          title="添加 API 密钥"
          channels={channels}
          onClose={() => setShowAddKey(false)}
          onSubmit={handleAddKey}
        />
      )}
      {showImportKeys && (
        <KeyImportModal
          channels={channels}
          onClose={() => setShowImportKeys(false)}
          onImport={handleImportKeys}
        />
      )}
      {editingKey && (
        <KeyFormModal
          title="编辑 API 密钥"
          initialKey={editingKey}
          onClose={() => setEditingKey(null)}
          onSubmit={handleUpdateKey}
        />
      )}
      <ConfirmDialog open={delKeyId !== null} title="删除密钥" message="确定要永久移除这个 API 密钥吗？" onConfirm={handleDelete} onCancel={() => setDelKeyId(null)} danger />
      <Modal isOpen={probeResult !== null} onOpenChange={(isOpen) => !isOpen && setProbeResult(null)} size="sm">
        <ModalContent>
          <ModalHeader>探测结果</ModalHeader>
          <ModalBody>
            <div className="space-y-2 text-sm">
              <div>是否成功：<Chip size="sm" variant="flat" color={probeResult?.success ? "success" : "danger"}>{probeResult?.success ? "是" : "否"}</Chip></div>
              <div>状态码：{probeResult?.status_code}</div>
              <div>延迟：{probeResult?.latency_ms}ms</div>
              {probeResult?.error_msg && <div className="text-danger">错误：{probeResult.error_msg}</div>}
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="flat" onPress={() => setProbeResult(null)}>关闭</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
      <Modal isOpen={batchProbeResult !== null} onOpenChange={(isOpen) => !isOpen && setBatchProbeResult(null)} size="2xl">
        <ModalContent>
          <ModalHeader>批量探测结果</ModalHeader>
          <ModalBody>
            <div className="space-y-4 text-sm">
              <div className="flex flex-wrap gap-2">
                <Chip size="sm" variant="flat">总数：{batchProbeResult?.total || 0}</Chip>
                <Chip size="sm" variant="flat" color="success">成功：{batchProbeResult?.success || 0}</Chip>
                <Chip size="sm" variant="flat" color="danger">失败：{batchProbeResult?.failed || 0}</Chip>
                <Chip size="sm" variant="flat" color="warning">
                  已删除：{batchProbeResult?.results.filter((result) => result.deleted).length || 0}
                </Chip>
              </div>
              <div className="max-h-80 overflow-auto space-y-2">
                {batchProbeResult?.results.map((result) => {
                  const key = keys.find((item) => item.id === result.id);
                  const success = Boolean(result.probe?.success);
                  return (
                    <div key={result.id} className="flex flex-col gap-1 rounded-small border border-default-200 p-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{key?.alias || `#${result.id}`}</span>
                        <span className="font-mono text-xs text-default-500">{key ? maskKey(key.key_value) : ""}</span>
                        <Chip size="sm" variant="flat" color={success ? "success" : "danger"}>
                          {success ? "成功" : "失败"}
                        </Chip>
                        {result.deleted && (
                          <Chip size="sm" variant="flat" color="warning">
                            已删除
                          </Chip>
                        )}
                      </div>
                      <div className="text-xs text-default-500">
                        状态码：{result.probe?.status_code ?? "-"}，延迟：{result.probe?.latency_ms ?? 0}ms
                      </div>
                      {(result.error || result.probe?.error_msg) && (
                        <div className="text-xs text-danger">{result.error || result.probe?.error_msg}</div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="flat" onPress={() => setBatchProbeResult(null)}>关闭</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
      <Modal isOpen={showBatchProbeConfirm} onOpenChange={(isOpen) => !isOpen && setShowBatchProbeConfirm(false)} size="md">
        <ModalContent>
          <ModalHeader>批量探测</ModalHeader>
          <ModalBody>
            <div className="space-y-2 text-sm text-default-500">
              <p>将探测当前筛选出的 {filtered.length} 个密钥。</p>
              <p>选择“探测并删除 401”后，只有真正探测请求返回上游 401 的密钥会被删除；获取模型列表返回 401 不会删除。</p>
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="flat" onPress={() => setShowBatchProbeConfirm(false)}>
              取消
            </Button>
            <Button variant="bordered" onPress={() => executeBatchProbe(false)}>
              仅探测
            </Button>
            <Button color="danger" onPress={() => executeBatchProbe(true)}>
              探测并删除 401
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}

function downloadJSON(data: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
