/**
 * 演示数据配置
 * UI 独立的演示数据，不依赖后端
 *
 * 使用场景：
 * 1. DEMO_MODE=true: UI 使用本地演示数据（默认，适合展示）
 * 2. DEMO_MODE=false: UI 连接真实后端 API（适合开发/生产）
 */

export const DEMO_MODE = import.meta.env.VITE_DEMO_MODE !== "false"; // 默认启用演示模式

// 演示工作流数据
export const demoWorkflows = [
  {
    id: "demo-wf-1",
    name: "客户服务工作流",
    description: "自动处理客户咨询和问题",
    status: "idle",
    version: "1.0.0",
    steps: [
      { id: "s1", name: "接收客户咨询", status: "pending", type: "task", config: {} },
      { id: "s2", name: "分析问题类型", status: "pending", type: "task", config: {} },
      { id: "s3", name: "生成解决方案", status: "pending", type: "task", config: {} },
      { id: "s4", name: "发送回复", status: "pending", type: "task", config: {} },
    ],
    createdAt: "2024-11-20",
  },
  {
    id: "demo-wf-2",
    name: "内容生成工作流",
    description: "自动生成营销内容和文案",
    status: "idle",
    version: "1.0.0",
    steps: [
      { id: "s1", name: "分析目标受众", status: "pending", type: "task", config: {} },
      { id: "s2", name: "生成内容大纲", status: "pending", type: "task", config: {} },
      { id: "s3", name: "撰写正文", status: "pending", type: "task", config: {} },
      { id: "s4", name: "审核优化", status: "pending", type: "task", config: {} },
    ],
    createdAt: "2024-11-19",
  },
  {
    id: "demo-wf-3",
    name: "数据分析工作流",
    description: "分析业务数据并生成报告",
    status: "completed",
    version: "1.0.0",
    steps: [
      { id: "s1", name: "收集数据", status: "completed", type: "task", config: {} },
      { id: "s2", name: "数据清洗", status: "completed", type: "task", config: {} },
      { id: "s3", name: "统计分析", status: "completed", type: "task", config: {} },
      { id: "s4", name: "生成报告", status: "completed", type: "task", config: {} },
    ],
    createdAt: "2024-11-18",
  },
];

// 演示房间数据
export const demoRooms = [
  {
    id: "demo-room-1",
    name: "产品开发团队",
    description: "产品经理、设计师、开发者协作",
    members: 5,
    agents: ["产品 Agent", "设计 Agent", "开发 Agent"],
    status: "active",
    maxMembers: 10,
    createdAt: "2024-11-20",
  },
  {
    id: "demo-room-2",
    name: "营销策划组",
    description: "营销团队协作制定推广策略",
    members: 3,
    agents: ["营销 Agent", "文案 Agent"],
    status: "active",
    maxMembers: 8,
    createdAt: "2024-11-19",
  },
  {
    id: "demo-room-3",
    name: "客户支持中心",
    description: "客服团队协作处理客户问题",
    members: 8,
    agents: ["客服 Agent", "技术支持 Agent", "售后 Agent"],
    status: "active",
    maxMembers: 15,
    createdAt: "2024-11-18",
  },
];

// 演示 Agent 数据
export const demoAgents = [
  {
    id: "demo-agent-1",
    name: "写作助手",
    description: "帮助你创作优质内容",
    status: "idle",
    template_id: "chat",
    model_config: {
      provider: "anthropic",
      model: "claude-3-5-sonnet",
    },
    createdAt: "2024-11-20",
  },
  {
    id: "demo-agent-2",
    name: "代码助手",
    description: "协助编写和优化代码",
    status: "idle",
    template_id: "chat",
    model_config: {
      provider: "openai",
      model: "gpt-4",
    },
    createdAt: "2024-11-19",
  },
  {
    id: "demo-agent-3",
    name: "数据分析师",
    description: "分析数据并生成洞察",
    status: "idle",
    template_id: "chat",
    model_config: {
      provider: "deepseek",
      model: "deepseek-chat",
    },
    createdAt: "2024-11-18",
  },
];
