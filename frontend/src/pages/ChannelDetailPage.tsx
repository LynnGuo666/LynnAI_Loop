import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { getChannel, listKeysByChannel, createKey, deleteKey, enableKey, probeKey, probeKeys, getUsageTimeseries, updateChannel, updateKey, listChannelModels, exportKeys, importKeys, listUsage } from "../api/client";
import { StatCard, DataTable, ConfirmDialog, KeyFormModal, KeyImportModal } from "../components/common";
import type { Channel, APIKey, TimeseriesPoint, KeyProbe, KeyProbeBatchResponse, KeyImportItem, UsageLog } from "../types";
import { AreaChart, Area, CartesianGrid, Line, LineChart, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";
import {
  Card,
  CardBody,
  Button,
  Chip,
  Input,
  Select,
  SelectItem,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Spinner,
} from "@heroui/react";

const PROTOCOL_OPTIONS = [
  { key: "anthropic_messages", label: "Anthropic Messages", endpoint: "/v1/messages" },
  { key: "openai_chat_completions", label: "OpenAI Chat Completions", endpoint: "/v1/chat/completions" },
  { key: "openai_responses", label: "OpenAI Responses", endpoint: "/v1/responses" },
  { key: "gemini_generate_content", label: "Gemini Generate Content", endpoint: "/v1beta/models/{model}:generateContent" },
];

function protocolLabel(protocol: string) {
  return PROTOCOL_OPTIONS.find((item) => item.key === protocol)?.label || protocol || "Anthropic Messages";
}

function proxyEndpoint(channel: Channel) {
  const endpoint = PROTOCOL_OPTIONS.find((item) => item.key === channel.protocol)?.endpoint || "/v1/messages";
  return `/channel/${channel.id}${endpoint}`;
}

interface ChannelPerformancePoint {
  time: string;
  first_token_ms: number;
  output_tokens_per_sec: number;
}

export function ChannelDetailPage() {
  const { id } = useParams<{ id: string }>();
  const channelId = Number(id);
  const [channel, setChannel] = useState<Channel | null>(null);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [timeseries, setTimeseries] = useState<TimeseriesPoint[]>([]);
  const [showAddKey, setShowAddKey] = useState(false);
  const [showImportKeys, setShowImportKeys] = useState(false);
  const [editingKey, setEditingKey] = useState<APIKey | null>(null);
  const [delKeyId, setDelKeyId] = useState<number | null>(null);
  const [probeResult, setProbeResult] = useState<KeyProbe | null>(null);
  const [batchProbeResult, setBatchProbeResult] = useState<KeyProbeBatchResponse | null>(null);
  const [probingKeyId, setProbingKeyId] = useState<number | null>(null);
  const [batchProbing, setBatchProbing] = useState(false);
  const [showBatchProbeConfirm, setShowBatchProbeConfirm] = useState(false);
  const [probeModel, setProbeModel] = useState("");
  const [protocol, setProtocol] = useState("anthropic_messages");
  const [modelOptions, setModelOptions] = useState<string[]>([]);
  const [modelLoading, setModelLoading] = useState(false);
  const [modelMsg, setModelMsg] = useState("");
  const [performanceLogs, setPerformanceLogs] = useState<UsageLog[]>([]);

  const load = () => {
    getChannel(channelId).then(setChannel).catch(() => {});
    listKeysByChannel(channelId).then(setKeys).catch(() => {});
    getUsageTimeseries(7).then(setTimeseries).catch(() => {});
    listUsage({ channel_id: channelId, page: 1, page_size: 100 }).then((r) => setPerformanceLogs(r.data || [])).catch(() => {});
  };
  useEffect(() => { load(); }, [channelId]);
  useEffect(() => {
    if (channel) {
      setProbeModel(channel.probe_model || "");
      setProtocol(channel.protocol || "anthropic_messages");
    }
  }, [channel?.id, channel?.probe_model, channel?.protocol]);

  const activeKeys = keys.filter((k) => k.is_active).length;

  const maskKey = (k: string) => {
    if (k.length <= 10) return "****";
    return k.slice(0, 7) + "..." + k.slice(-4);
  };

  const handleAddKey = async (data: { channelId: number; keyValue: string; alias: string }) => {
    await createKey(channelId, { key_value: data.keyValue, alias: data.alias });
    load();
  };

  const handleImportKeys = async (data: { keys: KeyImportItem[] }) => {
    const result = await importKeys({ channel_id: channelId, keys: data.keys });
    load();
    return result;
  };

  const handleExportKeys = async () => {
    const exported = await exportKeys(channelId);
    downloadJSON(exported, `loop-keys-channel-${channelId}.json`);
  };

  const handleUpdateKey = async (data: { channelId: number; keyValue: string; alias: string }) => {
    if (!editingKey) return;
    await updateKey(editingKey.id, { key_value: data.keyValue, alias: data.alias });
    setEditingKey(null);
    load();
  };

  const handleDeleteKey = async () => {
    if (delKeyId) await deleteKey(delKeyId);
    setDelKeyId(null);
    load();
  };

  const handleEnable = async (keyId: number) => {
    await enableKey(keyId);
    load();
  };

  const handleProbe = async (keyId: number) => {
    setProbingKeyId(keyId);
    try {
      const result = await probeKey(keyId);
      setProbeResult(result);
      load();
    } finally {
      setProbingKeyId(null);
    }
  };

  const handleBatchProbe = async () => {
    if (keys.length === 0) return;
    setShowBatchProbeConfirm(true);
  };

  const executeBatchProbe = async (deleteOn401: boolean) => {
    const ids = keys.map((k) => k.id);
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

  const handleLoadModels = async () => {
    setModelLoading(true);
    setModelMsg("");
    try {
      const models = await listChannelModels(channelId);
      setModelOptions(models);
      if (!probeModel && models.length > 0) setProbeModel(models[0]);
      setModelMsg(models.length > 0 ? `已获取 ${models.length} 个模型` : "未获取到模型，可手动填写");
    } catch (err) {
      setModelMsg(err instanceof Error ? `获取模型失败：${err.message}` : "获取模型失败");
    } finally {
      setModelLoading(false);
      setTimeout(() => setModelMsg(""), 3000);
    }
  };

  const handleSaveProbeModel = async () => {
    if (!channel) return;
    setModelMsg("");
    try {
      const updated = await updateChannel(channel.id, { probe_model: probeModel.trim(), protocol });
      setChannel(updated);
      setModelMsg("协议和探测模型已保存");
    } catch {
      setModelMsg("保存探测模型失败");
    } finally {
      setTimeout(() => setModelMsg(""), 3000);
    }
  };

  const columns = [
    { key: "id", label: "ID" },
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
          {!k.is_active && (
            <Button size="sm" variant="light" color="success" onPress={() => handleEnable(k.id)}>启用</Button>
          )}
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

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
  };
  const performanceData = buildPerformanceData(performanceLogs);

  if (!channel) return <div className="flex justify-center py-8"><Spinner /></div>;

  return (
    <div className="space-y-4 md:space-y-6">
      <div>
        <h1 className="text-xl md:text-2xl font-bold">{channel.name}</h1>
        <p className="text-xs md:text-sm text-default-500 mt-1 break-all">{channel.base_url}</p>
        <p className="text-xs text-default-500 mt-1">{protocolLabel(channel.protocol)}</p>
        {channel.description && <p className="text-sm text-default-500">{channel.description}</p>}
      </div>
      <Card>
        <CardBody className="p-5 space-y-2">
          <h2 className="text-sm font-medium">调用提示</h2>
          <div className="text-xs text-default-500 space-y-1">
            <p className="break-all">外部调用此渠道：<span className="font-mono text-foreground">{proxyEndpoint(channel)}</span></p>
            <p className="break-all">请求必须携带管理员令牌：<span className="font-mono text-foreground">Authorization: Bearer &lt;adminToken&gt;</span> 或 <span className="font-mono text-foreground">x-api-key: &lt;adminToken&gt;</span>。</p>
            <p>探测会消耗上游少量请求，但只记录到探测历史，不计入用量统计。</p>
          </div>
        </CardBody>
      </Card>
      <Card>
        <CardBody className="p-5 space-y-3">
          <div>
            <h2 className="text-sm font-medium">协议与探测模型</h2>
            <p className="text-xs text-default-500 mt-1">用于业务代理、手动探测和自动恢复探测。端点不支持模型列表时，可以直接手填模型 ID；探测结果不会进入用量页。</p>
          </div>
          <div className="flex flex-col sm:flex-row flex-wrap gap-3">
            <Select
              label="上游协议"
              selectedKeys={new Set([protocol])}
              onSelectionChange={(keys) => {
                const key = Array.from(keys)[0] as string;
                if (key) setProtocol(key);
              }}
              className="sm:min-w-72"
            >
              {PROTOCOL_OPTIONS.map((item) => (
                <SelectItem key={item.key}>{item.label}</SelectItem>
              ))}
            </Select>
            {modelOptions.length > 0 && (
              <Select
                label="模型列表"
                placeholder="从模型列表选择"
                selectedKeys={modelOptions.includes(probeModel) ? new Set([probeModel]) : new Set()}
                onSelectionChange={(keys) => {
                  const key = Array.from(keys)[0] as string;
                  if (key) setProbeModel(key);
                }}
                className="sm:min-w-64"
              >
                {modelOptions.map((model) => (
                  <SelectItem key={model}>{model}</SelectItem>
                ))}
              </Select>
            )}
            <Input
              label="模型 ID"
              placeholder="手动填写模型 ID"
              value={probeModel}
              onValueChange={setProbeModel}
              className="sm:min-w-80 flex-1"
            />
            <div className="flex gap-2 items-end">
              <Button
                variant="bordered"
                onPress={handleLoadModels}
                isLoading={modelLoading}
              >
                获取模型列表
              </Button>
              <Button color="primary" onPress={handleSaveProbeModel}>
                保存
              </Button>
            </div>
          </div>
          {modelMsg && <div className="text-xs text-default-500">{modelMsg}</div>}
        </CardBody>
      </Card>
      <div className="grid grid-cols-3 gap-3 md:gap-4">
        <StatCard label="密钥总数" value={keys.length} />
        <StatCard label="可用密钥" value={activeKeys} color="text-success" />
        <StatCard label="停用密钥" value={keys.length - activeKeys} color="text-danger" />
      </div>
      <Card>
        <CardBody className="p-6">
          <div className="mb-4">
            <h2 className="text-sm font-medium">请求性能</h2>
            <p className="mt-1 text-xs text-default-500">基于此渠道最近 100 条业务请求，绘制首字耗时和输出速度。</p>
          </div>
          {performanceData.length > 0 ? (
            <ResponsiveContainer width="100%" height={260}>
              <LineChart data={performanceData}>
                <CartesianGrid stroke="var(--loop-chart-grid)" vertical={false} />
                <XAxis dataKey="time" tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} />
                <YAxis yAxisId="latency" tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} />
                <YAxis yAxisId="speed" orientation="right" tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} />
                <Tooltip
                  contentStyle={{
                    background: "var(--heroui-background)",
                    border: "1px solid var(--heroui-default-200)",
                    borderRadius: 8,
                    color: "var(--heroui-foreground)",
                    fontSize: 12,
                  }}
                  labelStyle={{ color: "var(--heroui-default-500)" }}
                  formatter={(value, name) => [
                    name === "首字" ? `${Math.round(Number(value))}ms` : `${Number(value).toFixed(1)} tok/s`,
                    name,
                  ]}
                />
                <Line yAxisId="latency" type="monotone" dataKey="first_token_ms" name="首字" stroke="#6366f1" strokeWidth={2} dot={{ r: 2 }} />
                <Line yAxisId="speed" type="monotone" dataKey="output_tokens_per_sec" name="输出速度" stroke="#0891b2" strokeWidth={2} dot={{ r: 2 }} />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-48 flex items-center justify-center text-center text-sm text-default-500">暂无业务请求性能数据</div>
          )}
        </CardBody>
      </Card>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <h2 className="text-lg font-semibold">API 密钥</h2>
        <div className="flex flex-wrap gap-2 sm:gap-3">
          <Button
            variant="bordered"
            size="sm"
            isLoading={batchProbing}
            isDisabled={keys.length === 0}
            onPress={handleBatchProbe}
          >
            批量探测（{keys.length}）
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
      <DataTable columns={columns} data={keys} empty="暂未添加密钥" />
      <Card>
        <CardBody className="p-6">
          <h2 className="text-sm font-medium text-default-500 mb-4">近 7 天用量</h2>
          {timeseries.length > 0 ? (
            <ResponsiveContainer width="100%" height={240}>
              <AreaChart data={timeseries}>
                <XAxis dataKey="date" tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} />
                <YAxis tick={{ fill: "#6b7280", fontSize: 11 }} tickLine={false} axisLine={false} tickFormatter={formatTokens} />
                <Tooltip
                  contentStyle={{
                    background: "var(--heroui-background)",
                    border: "1px solid var(--heroui-default-200)",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                />
                <Area type="monotone" dataKey="input_tokens" stroke="#6366f1" fill="#6366f120" strokeWidth={2} name="输入" />
                <Area type="monotone" dataKey="output_tokens" stroke="#22d3ee" fill="#22d3ee20" strokeWidth={2} name="输出" />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-48 flex items-center justify-center text-default-500 text-sm">暂无数据</div>
          )}
        </CardBody>
      </Card>
      {showAddKey && (
        <KeyFormModal
          title="添加 API 密钥"
          fixedChannelId={channelId}
          onClose={() => setShowAddKey(false)}
          onSubmit={handleAddKey}
        />
      )}
      {showImportKeys && (
        <KeyImportModal
          fixedChannelId={channelId}
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
      <ConfirmDialog open={delKeyId !== null} title="删除密钥" message="这会永久移除该 API 密钥。" onConfirm={handleDeleteKey} onCancel={() => setDelKeyId(null)} danger />
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
              <p>将探测当前渠道的 {keys.length} 个密钥。</p>
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

function buildPerformanceData(logs: UsageLog[]): ChannelPerformancePoint[] {
  return [...logs]
    .reverse()
    .filter((log) => log.first_token_ms > 0 || log.output_tokens_per_sec > 0)
    .map((log) => ({
      time: new Date(log.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      first_token_ms: log.first_token_ms,
      output_tokens_per_sec: log.output_tokens_per_sec,
    }));
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
