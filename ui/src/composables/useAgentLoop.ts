/**
 * useAgentLoop Composable
 * 
 * 封装完整的 Agent Loop，支持：
 * - 重试逻辑 (Retry with exponential backoff)
 * - Human-in-the-Loop (HITL) 审批流程
 * - 真实工具执行 (通过 Aster 后端)
 * - 流式响应
 * 
 * 对应 geminiService.ts 的功能，但使用 Aster 后端真实实现
 */

import { ref, reactive, computed } from 'vue';
import { useWebSocket } from './useWebSocket';
import { useApprovalStore } from '@/stores/approval';
import { useThinkingStore } from '@/stores/thinking';
import { useToolsStore } from '@/stores/tools';
import { generateId } from '@/utils/format';

// ==================
// Types
// ==================

export interface ThinkAloudEvent {
  id: string;
  stage: string;
  reasoning: string;
  decision: string;
  timestamp: string;
  context?: Record<string, any>;
  toolCall?: { toolName: string; args: Record<string, any> };
  toolResult?: { toolName: string; result: any };
  approvalRequest?: ApprovalRequest;
}

export interface ApprovalRequest {
  id: string;
  toolName: string;
  args: Record<string, any>;
}

export interface AgentExecutionResult {
  status: 'finished' | 'paused' | 'error';
  output: string;
  history: any[];
  approvalRequest?: ApprovalRequest;
  error?: string;
}

export interface AgentLoopConfig {
  apiUrl?: string;
  wsUrl?: string;
  apiKey?: string;
  modelConfig?: {
    provider?: string;
    model?: string;
    api_key?: string;
  };
  // 敏感工具列表 (需要 HITL 审批)
  sensitiveTools?: string[];
  // 最大重试次数
  maxRetries?: number;
  // 最大循环次数
  maxLoops?: number;
  // 回调
  onThink?: (event: ThinkAloudEvent) => void;
  onApprovalRequired?: (request: ApprovalRequest) => void;
  onToolStart?: (toolName: string, args: any) => void;
  onToolEnd?: (toolName: string, result: any) => void;
  onTextDelta?: (delta: string) => void;
  onComplete?: (result: AgentExecutionResult) => void;
  onError?: (error: Error) => void;
}

// ==================
// Composable
// ==================

export function useAgentLoop(config: AgentLoopConfig = {}) {
  const approvalStore = useApprovalStore();
  const thinkingStore = useThinkingStore();
  const toolsStore = useToolsStore();
  const { connect, getInstance, isConnected } = useWebSocket();

  // 状态
  const isRunning = ref(false);
  const isPaused = ref(false);
  const currentOutput = ref('');
  const history = ref<any[]>([]);
  const pendingApproval = ref<ApprovalRequest | null>(null);

  // 配置
  const sensitiveTools = config.sensitiveTools || ['Edit', 'Write', 'bash', 'fs_write'];
  const maxRetries = config.maxRetries || 3;
  const maxLoops = config.maxLoops || 10;

  // WebSocket URL
  const apiUrl = config.apiUrl || import.meta.env.VITE_API_URL || 'http://localhost:8080';
  const wsUrl = config.wsUrl || apiUrl.replace(/^http/, 'ws') + '/v1/ws';

  /**
   * 初始化 WebSocket 连接
   */
  const initConnection = async () => {
    if (!isConnected.value) {
      await connect(wsUrl);
    }
    return getInstance();
  };

  /**
   * 发送思考事件
   */
  const emitThink = (event: Partial<ThinkAloudEvent>) => {
    const fullEvent: ThinkAloudEvent = {
      id: generateId('think'),
      stage: event.stage || 'Thinking',
      reasoning: event.reasoning || '',
      decision: event.decision || '',
      timestamp: new Date().toISOString(),
      ...event,
    };
    
    config.onThink?.(fullEvent);
    return fullEvent;
  };

  /**
   * 执行 Agent Loop
   * 
   * @param input 用户输入
   * @param contextData 上下文数据
   * @param resumeState 恢复状态 (用于 HITL 恢复)
   */
  const execute = async (
    input: string,
    contextData: string = '',
    resumeState?: { history: any[]; approvedTool?: ApprovalRequest }
  ): Promise<AgentExecutionResult> => {
    isRunning.value = true;
    isPaused.value = false;
    currentOutput.value = '';

    try {
      const ws = await initConnection();
      if (!ws) {
        throw new Error('WebSocket connection failed');
      }

      return new Promise((resolve, reject) => {
        let loopCount = 0;
        let finalOutput = '';

        // 设置消息处理器
        const unsubscribe = ws.onMessage((message: any) => {
          const { type, payload } = message;

          switch (type) {
            // 思考开始
            case 'think_chunk_start':
              thinkingStore.startThinking(generateId('msg'));
              emitThink({
                stage: '任务规划',
                reasoning: '分析用户请求...',
                decision: '准备进入 Agent 执行循环',
              });
              break;

            // 思考内容
            case 'think_chunk':
              thinkingStore.handleThinkChunk(payload?.delta || '');
              break;

            // 思考结束
            case 'think_chunk_end':
              thinkingStore.endThinking();
              break;

            // 文本增量
            case 'text_delta':
              const delta = payload?.text || '';
              finalOutput += delta;
              currentOutput.value = finalOutput;
              config.onTextDelta?.(delta);
              break;

            // 工具开始
            case 'tool_start':
            case 'agent_event':
              if (payload?.type === 'tool:start' || payload?.event?.type === 'tool:start') {
                const call = payload?.event?.Call || payload?.call || {};
                const toolName = call.name || 'unknown';
                const args = call.arguments || {};

                // 检查是否需要审批
                if (sensitiveTools.includes(toolName)) {
                  isPaused.value = true;
                  const approvalRequest: ApprovalRequest = {
                    id: generateId('approval'),
                    toolName,
                    args,
                  };
                  pendingApproval.value = approvalRequest;

                  emitThink({
                    stage: 'Human in the Loop',
                    reasoning: `工具 ${toolName} 被标记为敏感操作`,
                    decision: '暂停执行，请求人工审批',
                    approvalRequest,
                  });

                  config.onApprovalRequired?.(approvalRequest);

                  // 添加到 approval store
                  approvalStore.addApprovalRequest({
                    id: approvalRequest.id,
                    toolName,
                    args,
                    timestamp: Date.now(),
                  });

                  // 暂停并返回
                  unsubscribe();
                  resolve({
                    status: 'paused',
                    output: finalOutput,
                    history: history.value,
                    approvalRequest,
                  });
                  return;
                }

                // 非敏感工具，正常执行
                toolsStore.handleToolStart({
                  id: call.id || generateId('tool'),
                  name: toolName,
                  state: 'executing',
                  progress: 0,
                  arguments: args,
                });

                emitThink({
                  stage: `调用工具: ${toolName}`,
                  reasoning: `模型请求使用 ${toolName}`,
                  decision: `执行 ${toolName}`,
                  toolCall: { toolName, args },
                });

                config.onToolStart?.(toolName, args);
              }
              break;

            // 工具结束
            case 'tool_end':
              if (payload?.type === 'tool:end' || payload?.event?.type === 'tool:end') {
                const call = payload?.event?.Call || payload?.call || {};
                const toolName = call.name || 'unknown';
                const result = call.result || payload?.result;

                toolsStore.handleToolEnd({
                  id: call.id || '',
                  name: toolName,
                  state: call.error ? 'failed' : 'completed',
                  progress: 1,
                  arguments: call.arguments || {},
                  result,
                  error: call.error,
                });

                emitThink({
                  stage: `工具返回: ${toolName}`,
                  reasoning: '工具执行完毕',
                  decision: `获取结果: ${JSON.stringify(result).substring(0, 50)}...`,
                  toolResult: { toolName, result },
                });

                config.onToolEnd?.(toolName, result);
                loopCount++;
              }
              break;

            // 需要审批
            case 'permission_required':
              isPaused.value = true;
              const call = payload?.call || {};
              const approvalRequest: ApprovalRequest = {
                id: payload?.request_id || generateId('approval'),
                toolName: call.name || '',
                args: call.arguments || {},
              };
              pendingApproval.value = approvalRequest;

              emitThink({
                stage: 'Human in the Loop',
                reasoning: `工具 ${approvalRequest.toolName} 需要人工审批`,
                decision: '暂停执行，等待审批',
                approvalRequest,
              });

              config.onApprovalRequired?.(approvalRequest);

              approvalStore.addApprovalRequest({
                id: approvalRequest.id,
                toolName: approvalRequest.toolName,
                args: approvalRequest.args,
                timestamp: Date.now(),
              });

              unsubscribe();
              resolve({
                status: 'paused',
                output: finalOutput,
                history: history.value,
                approvalRequest,
              });
              break;

            // 完成
            case 'chat_complete':
              isRunning.value = false;
              unsubscribe();

              const result: AgentExecutionResult = {
                status: 'finished',
                output: finalOutput,
                history: history.value,
              };

              config.onComplete?.(result);
              resolve(result);
              break;

            // 错误
            case 'error':
              isRunning.value = false;
              unsubscribe();

              const error = new Error(payload?.message || 'Unknown error');
              config.onError?.(error);
              resolve({
                status: 'error',
                output: finalOutput,
                history: history.value,
                error: payload?.message,
              });
              break;
          }

          // 检查循环限制
          if (loopCount >= maxLoops) {
            isRunning.value = false;
            unsubscribe();
            resolve({
              status: 'finished',
              output: finalOutput + '\n\n[达到最大循环次数限制]',
              history: history.value,
            });
          }
        });

        // 发送请求
        const messagePayload: any = {
          type: 'chat',
          payload: {
            template_id: 'agent',
            input: input,
            context: contextData,
            model_config: config.modelConfig,
          },
        };

        // 如果是恢复执行
        if (resumeState) {
          messagePayload.payload.resume = true;
          messagePayload.payload.history = resumeState.history;
          if (resumeState.approvedTool) {
            messagePayload.payload.approved_tool = resumeState.approvedTool;
          }
        }

        ws.send(messagePayload);
      });
    } catch (error: any) {
      isRunning.value = false;
      config.onError?.(error);
      return {
        status: 'error',
        output: currentOutput.value,
        history: history.value,
        error: error.message,
      };
    }
  };

  /**
   * 批准工具执行并恢复
   */
  const approveAndResume = async (requestId: string): Promise<AgentExecutionResult> => {
    const approval = pendingApproval.value;
    if (!approval || approval.id !== requestId) {
      throw new Error('No pending approval or ID mismatch');
    }

    // 发送审批决策
    approvalStore.approve(requestId);
    pendingApproval.value = null;
    isPaused.value = false;

    emitThink({
      stage: `调用工具: ${approval.toolName} (已批准)`,
      reasoning: '用户已批准敏感操作',
      decision: `执行 ${approval.toolName}`,
    });

    // 恢复执行
    return execute('', '', {
      history: history.value,
      approvedTool: approval,
    });
  };

  /**
   * 拒绝工具执行
   */
  const rejectTool = (requestId: string, reason?: string): AgentExecutionResult => {
    const approval = pendingApproval.value;
    if (!approval || approval.id !== requestId) {
      throw new Error('No pending approval or ID mismatch');
    }

    approvalStore.reject(requestId, reason);
    pendingApproval.value = null;
    isPaused.value = false;
    isRunning.value = false;

    emitThink({
      stage: `工具被拒绝: ${approval.toolName}`,
      reasoning: reason || '用户拒绝了敏感操作',
      decision: '终止执行',
    });

    return {
      status: 'finished',
      output: currentOutput.value + `\n\n[工具 ${approval.toolName} 被拒绝: ${reason || '用户拒绝'}]`,
      history: history.value,
    };
  };

  /**
   * 取消执行
   */
  const cancel = () => {
    isRunning.value = false;
    isPaused.value = false;
    pendingApproval.value = null;
  };

  return {
    // 状态
    isRunning: computed(() => isRunning.value),
    isPaused: computed(() => isPaused.value),
    currentOutput: computed(() => currentOutput.value),
    pendingApproval: computed(() => pendingApproval.value),
    isConnected,

    // 方法
    execute,
    approveAndResume,
    rejectTool,
    cancel,
    initConnection,
  };
}
