<template>
  <div :class="['thinking-block rounded-lg border overflow-hidden transition-all', hasApproval ? 'border-amber-300 bg-amber-50' : 'border-border bg-surface/50']">
    <!-- Header -->
    <button @click="isExpanded = !isExpanded" class="w-full flex items-center gap-2 px-3 py-2 text-xs font-medium text-secondary hover:bg-background/50 transition-colors">
      <svg :class="['w-4 h-4 transition-transform', isExpanded ? 'rotate-90' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
      </svg>

      <svg :class="['w-4 h-4', hasApproval ? 'text-amber-500 animate-pulse' : isFinished ? 'text-secondary' : 'text-primary animate-pulse']" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
        ></path>
      </svg>

      <span>思考中...</span>

      <span v-if="hasApproval" class="ml-auto flex items-center gap-1 text-amber-600 font-bold animate-pulse bg-amber-100 px-2 py-0.5 rounded-full border border-amber-200">
        <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
        </svg>
        等待审批
      </span>

      <span v-else-if="!isFinished" class="ml-auto flex items-center gap-1 text-primary text-[10px] bg-primary/5 px-2 py-0.5 rounded-full border border-primary/10">
        <span class="w-1.5 h-1.5 rounded-full bg-primary animate-pulse"></span>
        运行中
      </span>
    </button>

    <!-- Content -->
    <div v-if="isExpanded" class="px-3 pb-3 pt-1 space-y-3 border-t border-border/50 bg-white/50">
      <div
        v-for="(thought, idx) in thoughts"
        :key="thought.id || idx"
        :class="['relative pl-4 border-l-2 ml-1 transition-all', thought.approvalRequest ? 'border-amber-400' : thought.toolCall ? 'border-indigo-300' : thought.toolResult ? 'border-emerald-300' : 'border-border']"
      >
        <!-- Stage Badge -->
        <div class="flex items-center justify-between mb-1">
          <span
            :class="[
              'text-[10px] font-bold px-1.5 py-0.5 rounded flex items-center gap-1',
              thought.approvalRequest ? 'bg-amber-100 text-amber-700' : thought.toolCall ? 'bg-indigo-100 text-indigo-700' : thought.toolResult ? 'bg-emerald-100 text-emerald-700' : 'bg-stone-100 text-stone-600',
            ]"
          >
            {{ thought.stage }}
          </span>
          <span class="text-[10px] text-secondary">
            {{ formatTime(thought.timestamp) }}
          </span>
        </div>

        <!-- Reasoning -->
        <p v-if="thought.reasoning" class="text-xs text-secondary leading-relaxed mb-2">
          {{ thought.reasoning }}
        </p>

        <!-- Decision -->
        <p v-if="thought.decision" class="text-xs text-primary font-medium">
          {{ thought.decision }}
        </p>

        <!-- Tool Call -->
        <div v-if="thought.toolCall" class="mt-2 bg-stone-900 rounded-md overflow-hidden">
          <div class="flex items-center gap-2 px-2 py-1.5 bg-stone-800 border-b border-stone-700">
            <svg class="w-3 h-3 text-indigo-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path>
            </svg>
            <span class="text-[10px] font-mono font-bold text-indigo-100">{{ thought.toolCall.name }}</span>
          </div>
          <pre class="p-2 text-[10px] font-mono text-stone-300 overflow-x-auto">{{ JSON.stringify(thought.toolCall.args, null, 2) }}</pre>
        </div>

        <!-- Tool Result -->
        <div v-if="thought.toolResult" class="mt-2 bg-emerald-50 border border-emerald-200 rounded-md overflow-hidden">
          <div class="flex items-center gap-2 px-2 py-1.5 bg-emerald-100/50 border-b border-emerald-200">
            <svg class="w-3 h-3 text-emerald-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
            </svg>
            <span class="text-[10px] font-bold text-emerald-800">Output</span>
          </div>
          <pre class="p-2 text-[10px] font-mono text-stone-600 overflow-x-auto">{{ JSON.stringify(thought.toolResult.result, null, 2) }}</pre>
        </div>

        <!-- Approval Request -->
        <div v-if="thought.approvalRequest" class="mt-3 bg-amber-50 border border-amber-200 rounded-lg p-3">
          <div class="flex items-start gap-2 mb-2">
            <svg class="w-5 h-5 text-amber-600 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
            </svg>
            <div>
              <h4 class="text-sm font-bold text-amber-900">需要人工审批</h4>
              <p class="text-xs text-amber-700 mt-0.5">
                Agent 请求执行敏感操作 <span class="font-mono font-bold bg-amber-100 px-1 rounded">{{ thought.approvalRequest.toolName }}</span>
              </p>
            </div>
          </div>

          <div class="bg-white border border-amber-200 rounded-md mb-3 overflow-hidden">
            <div class="bg-amber-50/50 px-2 py-1.5 border-b border-amber-100 text-xs font-semibold text-amber-800">参数详情</div>
            <pre class="p-2 text-[10px] font-mono text-stone-600 overflow-x-auto">{{ JSON.stringify(thought.approvalRequest.args, null, 2) }}</pre>
          </div>

          <div class="flex gap-2">
            <button @click="$emit('approve', thought.approvalRequest.id)" class="flex-1 flex items-center justify-center gap-2 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold py-2 rounded-md transition-colors">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
              </svg>
              批准执行
            </button>
            <button
              @click="$emit('reject', thought.approvalRequest.id)"
              class="flex-1 flex items-center justify-center gap-2 bg-white hover:bg-red-50 text-red-600 border border-red-200 hover:border-red-300 text-xs font-bold py-2 rounded-md transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
              拒绝
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import type { ThinkingEvent } from "@/types";

interface Props {
  thoughts: ThinkingEvent[];
  isFinished: boolean;
}

const props = defineProps<Props>();

defineEmits<{
  approve: [requestId: string];
  reject: [requestId: string];
}>();

const isExpanded = ref(false);

const hasApproval = computed(() => {
  return props.thoughts.some((t) => t.approvalRequest);
});

watch(
  () => props.isFinished,
  (finished) => {
    if (finished) {
      isExpanded.value = false;
    }
  },
);

watch(hasApproval, (has) => {
  if (has) {
    isExpanded.value = true;
  }
});

const formatTime = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};
</script>
