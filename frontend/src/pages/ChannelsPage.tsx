import { useEffect, useState } from "react";
import { listChannels, createChannel, deleteChannel, updateChannel } from "../api/client";
import { DataTable, ConfirmDialog } from "../components/common";
import type { Channel } from "../types";
import { useNavigate } from "react-router-dom";
import {
  Button,
  Chip,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Input,
  Link,
} from "@heroui/react";

export function ChannelsPage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [delId, setDelId] = useState<number | null>(null);
  const navigate = useNavigate();

  const load = () => listChannels().then(setChannels).catch(() => {});
  useEffect(() => { load(); }, []);

  const handleCreate = async (data: Partial<Channel>) => {
    await createChannel(data);
    setShowCreate(false);
    load();
  };

  const handleDelete = async () => {
    if (delId) await deleteChannel(delId);
    setDelId(null);
    load();
  };

  const toggleActive = async (ch: Channel) => {
    await updateChannel(ch.id, { ...ch, is_active: !ch.is_active });
    load();
  };

  const columns = [
    { key: "id", label: "ID" },
    {
      key: "name",
      label: "名称",
      render: (ch: Channel) => (
        <Link
          as="button"
          onPress={() => navigate(`/channels/${ch.id}`)}
          size="sm"
          className="font-medium"
        >
          {ch.name}
        </Link>
      ),
    },
    { key: "base_url", label: "基础地址" },
    { key: "probe_model", label: "探测模型", render: (ch: Channel) => ch.probe_model || "未设置" },
    { key: "description", label: "描述" },
    {
      key: "is_active",
      label: "状态",
      render: (ch: Channel) => (
        <Chip
          as="button"
          size="sm"
          variant="flat"
          color={ch.is_active ? "success" : "danger"}
          onClick={() => toggleActive(ch)}
          className="cursor-pointer"
        >
          {ch.is_active ? "启用" : "停用"}
        </Chip>
      ),
    },
    {
      key: "actions",
      label: "",
      render: (ch: Channel) => (
        <Button size="sm" variant="light" color="danger" onPress={() => setDelId(ch.id)}>
          删除
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-4 md:space-y-6">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl md:text-2xl font-bold">渠道</h1>
        <Button color="primary" onPress={() => setShowCreate(true)}>
          + 新建渠道
        </Button>
      </div>
      <DataTable columns={columns} data={channels} empty="暂无渠道" />
      <CreateChannelModal isOpen={showCreate} onClose={() => setShowCreate(false)} onCreate={handleCreate} />
      <ConfirmDialog open={delId !== null} title="删除渠道" message="这会永久删除该渠道及其所有密钥。" onConfirm={handleDelete} onCancel={() => setDelId(null)} danger />
    </div>
  );
}

function CreateChannelModal({ isOpen, onClose, onCreate }: { isOpen: boolean; onClose: () => void; onCreate: (d: Partial<Channel>) => void }) {
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [desc, setDesc] = useState("");
  const [probeModel, setProbeModel] = useState("");

  return (
    <Modal isOpen={isOpen} onOpenChange={(open) => !open && onClose()} size="md">
      <ModalContent>
        <ModalHeader>新建渠道</ModalHeader>
        <ModalBody className="gap-4">
          <Input label="渠道名称" value={name} onValueChange={setName} />
          <Input label="基础地址" placeholder="https://api.anthropic.com" value={baseUrl} onValueChange={setBaseUrl} />
          <Input label="探测模型 ID" placeholder="claude-3-5-haiku-latest（可选）" value={probeModel} onValueChange={setProbeModel} />
          <Input label="描述" placeholder="可选" value={desc} onValueChange={setDesc} />
        </ModalBody>
        <ModalFooter>
          <Button variant="flat" onPress={onClose}>取消</Button>
          <Button
            color="primary"
            onPress={() => onCreate({ name, base_url: baseUrl, description: desc, probe_model: probeModel.trim() })}
            isDisabled={!name || !baseUrl}
          >
            创建
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
