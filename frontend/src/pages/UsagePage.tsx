import { useEffect, useState } from "react";
import { listUsage, getUsageModels, listChannels, listAllKeys } from "../api/client";
import type { UsageLog, Channel, APIKey } from "../types";
import {
  Card,
  CardBody,
  Button,
  Chip,
  Spinner,
  Select,
  SelectItem,
  Input,
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Pagination,
} from "@heroui/react";

export function UsagePage() {
  const [logs, setLogs] = useState<UsageLog[]>([]);
  const [total, setTotal] = useState(0);
  const [models, setModels] = useState<string[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [page, setPage] = useState(1);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [filters, setFilters] = useState({
    channel_id: "",
    api_key_id: "",
    model: "",
    success: "",
    status: "",
    start_date: "",
    end_date: "",
  });

  const load = () => {
    const f: Record<string, unknown> = { page, page_size: 20 };
    if (filters.channel_id) f.channel_id = Number(filters.channel_id);
    if (filters.api_key_id) f.api_key_id = Number(filters.api_key_id);
    if (filters.model) f.model = filters.model;
    if (filters.success === "true") f.success = true;
    if (filters.success === "false") f.success = false;
    if (filters.status) f.status = filters.status;
    if (filters.start_date) f.start_date = filters.start_date;
    if (filters.end_date) f.end_date = filters.end_date;
    listUsage(f as any).then((r) => { setLogs(r.data || []); setTotal(r.total); }).catch(() => {});
  };

  useEffect(() => {
    load();
    getUsageModels().then(setModels).catch(() => {});
    listChannels().then(setChannels).catch(() => {});
    listAllKeys().then(setKeys).catch(() => {});
  }, [page, filters]);

  const channelMap = new Map(channels.map((c) => [c.id, c.name]));
  const keyMap = new Map(keys.map((k) => [k.id, k.alias || `${k.key_value.slice(0, 8)}...`]));

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
  };
  const formatMs = (n?: number) => (n && n > 0 ? `${Math.round(n)}ms` : "-");
  const formatSpeed = (n?: number) => (n && n > 0 ? `${n.toFixed(1)} tok/s` : "-");

  const totalPages = Math.ceil(total / 20);

  const tableColumns = [
    { key: "id", label: "ID" },
    { key: "channel_id", label: "渠道" },
    { key: "api_key_id", label: "Key" },
    { key: "model", label: "模型" },
    { key: "status", label: "状态" },
    { key: "latency_ms", label: "延迟" },
    { key: "performance", label: "性能" },
    { key: "created_at", label: "时间" },
    { key: "expand", label: "" },
  ];

  return (
    <div className="space-y-4 md:space-y-6">
      <div>
        <h1 className="text-xl md:text-2xl font-bold">用量</h1>
        <p className="text-sm text-default-500 mt-1">统计所有请求的 API 用量，包括代理请求和探测请求。</p>
      </div>
      <div className="flex flex-wrap gap-2 md:gap-3">
        <Select
          label="渠道"
          placeholder="全部渠道"
          selectedKeys={filters.channel_id ? new Set([filters.channel_id]) : new Set()}
          onSelectionChange={(keys) => {
            const key = Array.from(keys)[0] as string;
            setFilters({ ...filters, channel_id: key || "" });
            setPage(1);
          }}
          className="min-w-[120px] flex-1"
        >
          {channels.map((c) => <SelectItem key={String(c.id)}>{c.name}</SelectItem>)}
        </Select>
        <Select
          label="模型"
          placeholder="全部模型"
          selectedKeys={filters.model ? new Set([filters.model]) : new Set()}
          onSelectionChange={(keys) => {
            const key = Array.from(keys)[0] as string;
            setFilters({ ...filters, model: key || "" });
            setPage(1);
          }}
          className="min-w-[120px] flex-1"
        >
          {models.map((m) => <SelectItem key={m}>{m}</SelectItem>)}
        </Select>
        <Select
          label="状态"
          placeholder="全部状态"
          selectedKeys={filters.status ? new Set([filters.status]) : new Set()}
          onSelectionChange={(keys) => {
            const key = Array.from(keys)[0] as string;
            setFilters({ ...filters, status: key || "" });
            setPage(1);
          }}
          className="min-w-[100px] flex-1"
        >
          <SelectItem key="pending">进行中</SelectItem>
          <SelectItem key="success">成功</SelectItem>
          <SelectItem key="failed">失败</SelectItem>
        </Select>
        <Select
          label="结果"
          placeholder="全部结果"
          selectedKeys={filters.success ? new Set([filters.success]) : new Set()}
          onSelectionChange={(keys) => {
            const key = Array.from(keys)[0] as string;
            setFilters({ ...filters, success: key || "" });
            setPage(1);
          }}
          className="min-w-[100px] flex-1"
        >
          <SelectItem key="true">成功</SelectItem>
          <SelectItem key="false">失败</SelectItem>
        </Select>
        <Input
          type="date"
          label="开始日期"
          value={filters.start_date}
          onValueChange={(v) => { setFilters({ ...filters, start_date: v }); setPage(1); }}
          className="min-w-[130px] flex-1"
        />
        <Input
          type="date"
          label="结束日期"
          value={filters.end_date}
          onValueChange={(v) => { setFilters({ ...filters, end_date: v }); setPage(1); }}
          className="min-w-[130px] flex-1"
        />
      </div>
      <Table aria-label="Usage logs table">
        <TableHeader>
          {tableColumns.map((c) => (
            <TableColumn key={c.key}>{c.label}</TableColumn>
          ))}
        </TableHeader>
        <TableBody emptyContent="暂无用量记录">
          {logs.flatMap((log) => {
            const expanded = expandedId === log.id;
            const isPending = log.status === "pending";
            const mainRow = (
              <TableRow key={log.id}>
                <TableCell>{log.id}</TableCell>
                <TableCell>{channelMap.get(log.channel_id) || `#${log.channel_id}`}</TableCell>
                <TableCell>{keyMap.get(log.api_key_id) || `#${log.api_key_id}`}</TableCell>
                <TableCell>{log.model || "-"}</TableCell>
                <TableCell>
                  <StatusBadge status={log.status} />
                </TableCell>
                <TableCell>{isPending ? "-" : `${log.latency_ms}ms`}</TableCell>
                <TableCell>
                  {isPending ? (
                    <span className="text-default-500">-</span>
                  ) : (
                    <>
                      <div>{formatMs(log.first_token_ms)}</div>
                      <div className="text-xs text-default-500">{formatSpeed(log.output_tokens_per_sec)}</div>
                    </>
                  )}
                </TableCell>
                <TableCell>{new Date(log.created_at).toLocaleString()}</TableCell>
                <TableCell>
                  <Button
                    size="sm"
                    variant="bordered"
                    onPress={() => setExpandedId(expanded ? null : log.id)}
                  >
                    {expanded ? "收起" : "详情"}
                  </Button>
                </TableCell>
              </TableRow>
            );
            if (!expanded) return [mainRow];
            const detailRow = (
              <TableRow key={`${log.id}-detail`}>
                <TableCell colSpan={tableColumns.length}>
                  <div className="grid gap-3 text-xs sm:grid-cols-2 lg:grid-cols-4">
                    <DetailItem label="输入令牌" value={isPending ? "-" : formatTokens(log.input_tokens)} />
                    <DetailItem label="输出令牌" value={isPending ? "-" : formatTokens(log.output_tokens)} />
                    <DetailItem label="缓存写" value={isPending ? "-" : formatTokens(log.cache_creation_tokens)} />
                    <DetailItem label="缓存读" value={isPending ? "-" : formatTokens(log.cache_read_tokens)} />
                    <DetailItem label="请求结果" value={isPending ? "进行中" : log.success ? `${log.status_code}` : `错误 ${log.status_code}`} tone={isPending ? undefined : log.success ? "success" : "danger"} />
                    <DetailItem label="流式" value={isPending ? "-" : log.is_stream ? "是" : "否"} />
                    <DetailItem label="端点" value={log.endpoint || "-"} />
                    <DetailItem label="错误信息" value={isPending ? "-" : log.error_message || "-"} tone={!isPending && log.error_message ? "danger" : undefined} />
                  </div>
                </TableCell>
              </TableRow>
            );
            return [mainRow, detailRow];
          })}
        </TableBody>
      </Table>
      {totalPages > 1 && (
        <div className="flex justify-center">
          <Pagination
            total={totalPages}
            page={page}
            onChange={setPage}
            showControls
          />
        </div>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  if (status === "pending") {
    return (
      <span className="inline-flex items-center gap-1.5 text-warning">
        <Spinner size="sm" color="warning" />
        <span className="text-xs font-medium">请求中</span>
      </span>
    );
  }
  if (status === "failed") {
    return <Chip size="sm" variant="flat" color="danger">失败</Chip>;
  }
  return <Chip size="sm" variant="flat" color="success">成功</Chip>;
}

function DetailItem({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "success" | "danger";
}) {
  const toneClass = tone === "success" ? "text-success" : tone === "danger" ? "text-danger" : "text-foreground";

  return (
    <Card>
      <CardBody className="px-3 py-2">
        <div className="text-default-500">{label}</div>
        <div className={`mt-1 break-all font-medium ${toneClass}`}>{value}</div>
      </CardBody>
    </Card>
  );
}
