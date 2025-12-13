import { useCallback, useMemo, useState } from 'react';
import {
  ReactFlow,
  Controls,
  Background,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  MarkerType,
  BackgroundVariant,
  Panel,
} from '@xyflow/react';
import type { Node, Edge, Connection } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import {
  Bot,
  Wrench,
  MessageSquare,
  XCircle,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useAgents } from '../hooks/useApi';
import { cn } from '../lib/utils';
import type { AgentRecord } from '../api/types';

// Custom node component for Agent
function AgentNode({ data }: { data: AgentNodeData }) {
  const statusColor = data.status === 'active'
    ? 'border-[var(--color-success)]'
    : data.status === 'disabled'
      ? 'border-[var(--color-warning)]'
      : 'border-[var(--color-text-muted)]';

  return (
    <div
      className={cn(
        'px-4 py-3 rounded-lg bg-[var(--color-surface)] border-2 shadow-lg min-w-[180px]',
        statusColor
      )}
    >
      <div className="flex items-center gap-2 mb-2">
        <div className="w-8 h-8 rounded-lg bg-[var(--color-primary)]/10 flex items-center justify-center">
          <Bot className="w-4 h-4 text-[var(--color-primary)]" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="font-medium text-[var(--color-text-primary)] text-sm truncate">
            {data.name}
          </div>
          <div className="text-xs text-[var(--color-text-muted)] font-mono">
            {data.templateId}
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="flex items-center gap-3 text-xs">
        {data.toolCount > 0 && (
          <div className="flex items-center gap-1 text-[var(--color-text-muted)]">
            <Wrench className="w-3 h-3" />
            <span>{data.toolCount}</span>
          </div>
        )}
        {data.middlewareCount > 0 && (
          <div className="flex items-center gap-1 text-[var(--color-text-muted)]">
            <MessageSquare className="w-3 h-3" />
            <span>{data.middlewareCount}</span>
          </div>
        )}
        {data.model && (
          <div className="text-[var(--color-text-muted)] truncate flex-1">
            {data.model}
          </div>
        )}
      </div>
    </div>
  );
}

// Custom node for Tools
function ToolNode({ data }: { data: ToolNodeData }) {
  return (
    <div className="px-3 py-2 rounded-lg bg-[var(--color-surface-elevated)] border border-[var(--color-border)] shadow min-w-[120px]">
      <div className="flex items-center gap-2">
        <Wrench className="w-4 h-4 text-[var(--color-secondary)]" />
        <span className="text-sm text-[var(--color-text-secondary)]">{data.name}</span>
      </div>
    </div>
  );
}

// Custom node for Middleware
function MiddlewareNode({ data }: { data: MiddlewareNodeData }) {
  return (
    <div className="px-3 py-2 rounded-lg bg-[var(--color-primary)]/10 border border-[var(--color-primary)]/30 shadow min-w-[120px]">
      <div className="flex items-center gap-2">
        <MessageSquare className="w-4 h-4 text-[var(--color-primary)]" />
        <span className="text-sm text-[var(--color-primary)]">{data.name}</span>
      </div>
    </div>
  );
}

interface AgentNodeData {
  name: string;
  templateId: string;
  status: string;
  model?: string;
  toolCount: number;
  middlewareCount: number;
}

interface ToolNodeData {
  name: string;
}

interface MiddlewareNodeData {
  name: string;
}

const nodeTypes = {
  agent: AgentNode,
  tool: ToolNode,
  middleware: MiddlewareNode,
};

// Layout helper - simple grid layout
function layoutNodes(agents: AgentRecord[]): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  const agentSpacing = 350;
  const toolSpacing = 150;
  const middlewareSpacing = 150;

  agents.forEach((agent, agentIndex) => {
    const agentX = 100 + (agentIndex % 3) * agentSpacing;
    const agentY = 100 + Math.floor(agentIndex / 3) * 400;

    const agentName = (agent.metadata?.name as string) || agent.id.slice(0, 8);

    // Agent node
    nodes.push({
      id: agent.id,
      type: 'agent',
      position: { x: agentX, y: agentY },
      data: {
        name: agentName,
        templateId: agent.config.template_id,
        status: agent.status,
        model: agent.config.model_config?.model,
        toolCount: agent.config.tools?.length || 0,
        middlewareCount: agent.config.middlewares?.length || 0,
      },
    });

    // Tool nodes
    const tools = agent.config.tools || [];
    tools.slice(0, 5).forEach((tool, toolIndex) => {
      const toolId = `${agent.id}-tool-${toolIndex}`;
      const toolX = agentX - 100 + toolIndex * toolSpacing;
      const toolY = agentY + 120;

      nodes.push({
        id: toolId,
        type: 'tool',
        position: { x: toolX, y: toolY },
        data: { name: tool },
      });

      edges.push({
        id: `${agent.id}-to-${toolId}`,
        source: agent.id,
        target: toolId,
        type: 'smoothstep',
        animated: true,
        style: { stroke: 'var(--color-secondary)' },
        markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--color-secondary)' },
      });
    });

    // Middleware nodes
    const middlewares = agent.config.middlewares || [];
    middlewares.slice(0, 4).forEach((middleware, mwIndex) => {
      const mwId = `${agent.id}-mw-${mwIndex}`;
      const mwX = agentX - 50 + mwIndex * middlewareSpacing;
      const mwY = agentY - 80;

      nodes.push({
        id: mwId,
        type: 'middleware',
        position: { x: mwX, y: mwY },
        data: { name: middleware },
      });

      edges.push({
        id: `${mwId}-to-${agent.id}`,
        source: mwId,
        target: agent.id,
        type: 'smoothstep',
        style: { stroke: 'var(--color-primary)', strokeDasharray: '5,5' },
        markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--color-primary)' },
      });
    });
  });

  return { nodes, edges };
}

export function AgentTopology() {
  const { data: agents, isLoading, error, refetch, isFetching } = useAgents();
  const [selectedAgent, setSelectedAgent] = useState<AgentRecord | null>(null);

  const { nodes: initialNodes, edges: initialEdges } = useMemo(() => {
    return layoutNodes(agents || []);
  }, [agents]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Update nodes/edges when agents change
  useMemo(() => {
    const { nodes: newNodes, edges: newEdges } = layoutNodes(agents || []);
    setNodes(newNodes);
    setEdges(newEdges);
  }, [agents, setNodes, setEdges]);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      if (node.type === 'agent') {
        const agent = agents?.find((a) => a.id === node.id);
        setSelectedAgent(agent || null);
      }
    },
    [agents]
  );

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Agent Topology"
        subtitle="多 Agent 拓扑可视化"
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
      />

      <div className="flex-1 relative">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin w-8 h-8 border-2 border-[var(--color-primary)] border-t-transparent rounded-full" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-full text-[var(--color-error)]">
            <XCircle className="w-12 h-12 mb-4" />
            <p>加载失败: {(error as Error).message}</p>
          </div>
        ) : agents?.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-[var(--color-text-muted)]">
            <Bot className="w-12 h-12 mb-4" />
            <p>暂无 Agent，请先创建 Agent</p>
          </div>
        ) : (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            nodeTypes={nodeTypes}
            fitView
            attributionPosition="bottom-left"
            className="bg-[var(--color-background)]"
          >
            <Controls
              className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg"
              showZoom={true}
              showFitView={true}
              showInteractive={false}
            />
            <MiniMap
              className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg"
              nodeStrokeColor="var(--color-border)"
              nodeColor="var(--color-primary)"
              maskColor="rgba(0, 0, 0, 0.1)"
            />
            <Background
              variant={BackgroundVariant.Dots}
              gap={20}
              size={1}
              color="var(--color-border)"
            />

            {/* Legend Panel */}
            <Panel position="top-right" className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-3 m-2">
              <h4 className="text-sm font-medium text-[var(--color-text-primary)] mb-2">图例</h4>
              <div className="space-y-2 text-xs">
                <div className="flex items-center gap-2">
                  <div className="w-4 h-4 rounded bg-[var(--color-primary)]/10 border border-[var(--color-primary)]" />
                  <span className="text-[var(--color-text-muted)]">Agent</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-4 h-4 rounded bg-[var(--color-surface-elevated)] border border-[var(--color-border)]" />
                  <span className="text-[var(--color-text-muted)]">Tool</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-4 h-4 rounded bg-[var(--color-primary)]/10 border border-[var(--color-primary)]/30" />
                  <span className="text-[var(--color-text-muted)]">Middleware</span>
                </div>
              </div>
            </Panel>
          </ReactFlow>
        )}

        {/* Agent Detail Sidebar */}
        {selectedAgent && (
          <div className="absolute right-0 top-0 bottom-0 w-80 bg-[var(--color-surface)] border-l border-[var(--color-border)] shadow-lg overflow-auto">
            <div className="p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold text-[var(--color-text-primary)]">Agent 详情</h3>
                <button
                  onClick={() => setSelectedAgent(null)}
                  className="p-1 rounded hover:bg-[var(--color-surface-elevated)] transition-colors"
                >
                  <XCircle className="w-4 h-4 text-[var(--color-text-muted)]" />
                </button>
              </div>

              {/* Basic Info */}
              <div className="space-y-3">
                <div>
                  <div className="text-xs text-[var(--color-text-muted)] mb-1">名称</div>
                  <div className="text-sm text-[var(--color-text-primary)]">
                    {(selectedAgent.metadata?.name as string) || selectedAgent.id}
                  </div>
                </div>

                <div>
                  <div className="text-xs text-[var(--color-text-muted)] mb-1">ID</div>
                  <div className="text-sm text-[var(--color-text-secondary)] font-mono">
                    {selectedAgent.id}
                  </div>
                </div>

                <div>
                  <div className="text-xs text-[var(--color-text-muted)] mb-1">模板</div>
                  <div className="text-sm text-[var(--color-text-secondary)]">
                    {selectedAgent.config.template_id}
                  </div>
                </div>

                {selectedAgent.config.model_config?.model && (
                  <div>
                    <div className="text-xs text-[var(--color-text-muted)] mb-1">模型</div>
                    <div className="text-sm text-[var(--color-text-secondary)]">
                      {selectedAgent.config.model_config.model}
                    </div>
                  </div>
                )}

                {/* Tools */}
                {selectedAgent.config.tools && selectedAgent.config.tools.length > 0 && (
                  <div>
                    <div className="text-xs text-[var(--color-text-muted)] mb-2">
                      工具 ({selectedAgent.config.tools.length})
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {selectedAgent.config.tools.map((tool, idx) => (
                        <span
                          key={idx}
                          className="px-2 py-0.5 text-xs bg-[var(--color-secondary)]/10 text-[var(--color-secondary)] rounded"
                        >
                          {tool}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {/* Middlewares */}
                {selectedAgent.config.middlewares && selectedAgent.config.middlewares.length > 0 && (
                  <div>
                    <div className="text-xs text-[var(--color-text-muted)] mb-2">
                      中间件 ({selectedAgent.config.middlewares.length})
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {selectedAgent.config.middlewares.map((mw, idx) => (
                        <span
                          key={idx}
                          className="px-2 py-0.5 text-xs bg-[var(--color-primary)]/10 text-[var(--color-primary)] rounded"
                        >
                          {mw}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {/* Status */}
                <div>
                  <div className="text-xs text-[var(--color-text-muted)] mb-1">状态</div>
                  <div className={cn(
                    'text-sm font-medium',
                    selectedAgent.status === 'active'
                      ? 'text-[var(--color-success)]'
                      : selectedAgent.status === 'disabled'
                        ? 'text-[var(--color-warning)]'
                        : 'text-[var(--color-text-muted)]'
                  )}>
                    {selectedAgent.status === 'active' ? '运行中' :
                      selectedAgent.status === 'disabled' ? '已禁用' : '已归档'}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
