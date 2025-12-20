<template>
  <div
    v-if="hasThoughts"
    :class="[
      'thinking-block',
      pendingApproval ? 'thinking-block--approval' : '',
      isFinished ? 'thinking-block--finished' : ''
    ]"
  >
    <!-- 折叠头部 -->
    <button
      class="thinking-header"
      @click="toggleExpand"
    >
      <svg
        :class="['expand-icon', isExpanded ? 'expanded' : '']"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
      </svg>

      <svg
        :class="['status-icon', statusIconClass]"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
      </svg>

      <span class="thinking-title">思考过程 ({{ thoughtCount }} 步骤)</span>

      <!-- HITL 等待标签 -->
      <span v-if="pendingApproval" class="status-badge status-badge--approval">
        <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
        </svg>
        等待审批
      </span>

      <!-- 运行中标签 -->
      <span v-else-if="!isFinished" class="status-badge status-badge--running">
        <span class="pulse-dot"></span>
        运行中
      </span>
    </button>

    <!-- 思考详情 -->
    <div v-show="isExpanded" class="thinking-content">
      <TransitionGroup name="thought" tag="div" class="thoughts-list">
        <div
          v-for="event in processedThoughts"
          :key="event.id"
          :class="['thought-item', getBorderClass(event)]"
        >
          <!-- 标题行 -->
          <div class="thought-header">
            <ThinkingStageTag :event="event" />
            <span class="thought-time">
              {{ formatTime(event.timestamp) }}
            </span>
          </div>

          <div class="thought-body">
            <!-- 推理事件 -->
            <template v-if="event.type === 'reasoning'">
              <p class="reasoning-text">
                {{ event.reasoning }}
              </p>
              <!-- 决策结论 -->
              <div v-if="event.decision" class="decision-box">
                <span class="decision-icon">▶</span>
                <p class="decision-text">
                  {{ event.decision }}
                </p>
              </div>
            </template>

            <!-- 工具调用/结果 -->
            <template v-else-if="event.type === 'tool_call' || event.type === 'tool_result'">
              <ToolCallDisplay :event="event" />
            </template>

            <!-- 审批请求 -->
            <template v-else-if="event.type === 'approval_required'">
              <p v-if="event.reasoning" class="approval-reason">
                {{ event.reasoning }}
              </p>
            </template>
          </div>
        </div>
      </TransitionGroup>

      <!-- HITL 审批卡片 -->
      <ApprovalPanel
        v-if="pendingApproval"
        :pending-approval="pendingApproval"
        :is-loading="isApproving"
        @approve="handleApprove"
        @reject="handleReject"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import type { ThinkAloudEvent, ToolCallRecord } from '@/types/thinking'
import { isValidThinkAloudEvent } from '@/types/thinking'
import ThinkingStageTag from './ThinkingStageTag.vue'
import ToolCallDisplay from './ToolCallDisplay.vue'
import ApprovalPanel from './ApprovalPanel.vue'

const props = defineProps<{
  thoughts: ThinkAloudEvent[]
  isFinished: boolean
  pendingApproval?: ToolCallRecord
  isApproving?: boolean
}>()

const emit = defineEmits<{
  approve: [request: ToolCallRecord]
  reject: [request?: ToolCallRecord]
}>()

// 展开状态 - 运行中时默认展开，完成后才折叠
const isExpanded = ref(true)

// 计算：是否有思考事件
const hasThoughts = computed(() =>
  props.thoughts && props.thoughts.length > 0
)

// 计算：思考事件数量
const thoughtCount = computed(() =>
  props.thoughts?.length ?? 0
)

// 计算：处理后的思考事件（确保每个都有 ID）
const processedThoughts = computed(() => {
  if (!props.thoughts) return []

  return props.thoughts
    .filter(isValidThinkAloudEvent)
    .map((event, index) => ({
      ...event,
      id: event.id || `thought-${index}-${Date.now()}`
    }))
})

// 计算：状态图标样式类
const statusIconClass = computed(() => {
  if (props.pendingApproval) return 'status-icon--approval'
  if (props.isFinished) return 'status-icon--finished'
  return 'status-icon--running'
})

// 切换展开/折叠
const toggleExpand = () => {
  isExpanded.value = !isExpanded.value
}

// 监听状态变化，自动展开/折叠
watch(
  () => [props.pendingApproval, props.isFinished] as const,
  ([newPendingApproval, newIsFinished], oldValue) => {
    const oldPendingApproval = oldValue?.[0]
    const oldIsFinished = oldValue?.[1]

    // 核心规则：正在运行时必须展开
    if (!newIsFinished) {
      isExpanded.value = true
      return
    }

    // 有新的审批请求时展开
    if (newPendingApproval && !oldPendingApproval) {
      isExpanded.value = true
      return
    }

    // 刚刚完成时，延迟折叠
    if (newIsFinished && !oldIsFinished && !newPendingApproval) {
      setTimeout(() => {
        if (props.isFinished && !props.pendingApproval) {
          isExpanded.value = false
        }
      }, 3000)
    }
  },
  { immediate: true }
)

// 获取边框颜色类
const getBorderClass = (event: ThinkAloudEvent): string => {
  if (event.type === 'approval_required' || event.stage === '人工介入') {
    return 'thought-item--approval'
  }
  if (event.type === 'tool_call') {
    return 'thought-item--tool-call'
  }
  if (event.type === 'tool_result') {
    return 'thought-item--tool-result'
  }
  return 'thought-item--reasoning'
}

// 格式化时间
const formatTime = (timestamp: string): string => {
  if (!timestamp) return ''
  try {
    return new Date(timestamp).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  } catch {
    return ''
  }
}

// 处理批准
const handleApprove = (request: ToolCallRecord) => {
  emit('approve', request)
}

// 处理拒绝
const handleReject = (request: ToolCallRecord) => {
  emit('reject', request)
}

// 清理
onUnmounted(() => {
  // 清理可能的定时器
})
</script>

<style scoped>
.thinking-block {
  margin-bottom: 16px;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  background: #f8fafc;
  overflow: hidden;
  transition: all 0.3s ease;
}

.thinking-block--approval {
  border-color: #fcd34d;
  background: #fffbeb;
  box-shadow: 0 2px 8px rgba(251, 191, 36, 0.15);
}

.thinking-block--finished {
  opacity: 0.85;
}

/* 头部 */
.thinking-header {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  font-size: 12px;
  font-weight: 500;
  color: #475569;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background 0.2s;
}

.thinking-header:hover {
  background: rgba(0, 0, 0, 0.03);
}

.expand-icon {
  width: 14px;
  height: 14px;
  color: #94a3b8;
  transition: transform 0.2s;
  flex-shrink: 0;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.status-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.status-icon--running {
  color: #3b82f6;
  animation: pulse 2s ease-in-out infinite;
}

.status-icon--approval {
  color: #f59e0b;
  animation: pulse 1.5s ease-in-out infinite;
}

.status-icon--finished {
  color: #94a3b8;
}

.thinking-title {
  flex: 1;
  text-align: left;
}

/* 状态标签 */
.status-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 9999px;
}

.status-badge--approval {
  color: #d97706;
  background: #fef3c7;
  border: 1px solid #fde68a;
  animation: pulse 1.5s ease-in-out infinite;
}

.status-badge--running {
  color: #3b82f6;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
}

.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #3b82f6;
  animation: pulse 1.5s ease-in-out infinite;
}

/* 内容区 */
.thinking-content {
  padding: 4px 12px 12px;
  border-top: 1px solid rgba(0, 0, 0, 0.05);
  background: rgba(255, 255, 255, 0.5);
}

.thoughts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 思考项 */
.thought-item {
  position: relative;
  padding-left: 16px;
  margin-left: 4px;
  border-left: 2px solid #e2e8f0;
  transition: all 0.3s ease;
  min-width: 0;
  overflow: hidden;
}

.thought-item--reasoning {
  border-left-color: #cbd5e1;
}

.thought-item--tool-call {
  border-left-color: #a5b4fc;
}

.thought-item--tool-result {
  border-left-color: #6ee7b7;
}

.thought-item--approval {
  border-left-color: #fcd34d;
}

.thought-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  gap: 8px;
  flex-wrap: nowrap;
  min-width: 0;
}

.thought-time {
  font-size: 10px;
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
}

.thought-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

/* 推理文本 */
.reasoning-text {
  font-size: 12px;
  color: #475569;
  line-height: 1.6;
  margin: 0;
}

/* 决策框 */
.decision-box {
  display: flex;
  gap: 6px;
  align-items: flex-start;
  margin-top: 4px;
  background: #f1f5f9;
  border-radius: 6px;
  padding: 8px 10px;
}

.decision-icon {
  color: #3b82f6;
  font-size: 12px;
  margin-top: 1px;
  flex-shrink: 0;
}

.decision-text {
  font-size: 12px;
  color: #1e293b;
  font-weight: 500;
  margin: 0;
  line-height: 1.5;
}

/* 审批原因 */
.approval-reason {
  font-size: 10px;
  color: #64748b;
  font-style: italic;
  margin: 0;
}

/* 过渡动画 */
.thought-enter-active,
.thought-leave-active {
  transition: all 0.3s ease;
}

.thought-enter-from {
  opacity: 0;
  transform: translateX(-10px);
}

.thought-leave-to {
  opacity: 0;
  transform: translateX(10px);
}

.thought-move {
  transition: transform 0.3s ease;
}

/* 脉冲动画 */
@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>
