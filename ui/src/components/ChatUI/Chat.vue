<template>
  <div class="chatui-container">
    <!-- 消息列表 -->
    <div ref="messagesRef" class="chatui-messages">
      <div
        v-for="msg in messages"
        :key="msg.id"
        :class="['chatui-message', `chatui-message-${msg.position}`]"
      >
        <!-- 气泡消息 -->
        <Bubble
          v-if="msg.type === 'text'"
          :content="msg.content ?? ''"
          :position="msg.position"
          :status="msg.status"
          :avatar="msg.user?.avatar"
        />
        
        <!-- 思考过程块 -->
        <ThinkingBlock
          v-else-if="msg.type === 'thinking' && msg.thinkingSteps?.length"
          :message-id="msg.id"
          :steps="msg.thinkingSteps"
          :is-active="msg.isThinkingActive"
        />
        
        <!-- 简单思考气泡（无步骤时） -->
        <ThinkBubble
          v-else-if="msg.type === 'thinking'"
        />
        
        <!-- 打字中指示器 -->
        <TypingBubble
          v-else-if="msg.type === 'typing'"
        />
        
        <!-- 卡片消息 -->
        <Card
          v-else-if="msg.type === 'card'"
          :title="msg.card?.title ?? ''"
          :content="msg.card?.content ?? ''"
          :actions="msg.card?.actions"
          @action="handleCardAction"
        />
        
        <!-- 文件卡片 -->
        <FileCard
          v-else-if="msg.type === 'file' && msg.file"
          :file="msg.file"
        />
      </div>
      
      <!-- 滚动锚点 -->
      <div ref="scrollAnchor" class="scroll-anchor"></div>
    </div>

    <!-- 输入区域 -->
    <div class="chatui-composer">
      <div class="composer-input-wrapper">
        <!-- 工具栏 -->
        <div class="composer-toolbar">
          <Button
            v-for="tool in toolbar"
            :key="tool.icon"
            :icon="tool.icon"
            variant="text"
            @click="tool.onClick"
          />
        </div>
        
        <!-- 输入框 -->
        <div class="composer-input">
          <textarea
            ref="inputRef"
            v-model="inputValue"
            :placeholder="placeholder"
            :disabled="disabled"
            class="composer-textarea"
            @keydown="handleKeyDown"
            @input="handleInput"
          />
        </div>
        
        <!-- 发送按钮 -->
        <Button
          icon="send"
          :disabled="!canSend"
          @click="handleSend"
        />
      </div>
      
      <!-- 快捷回复 -->
      <div v-if="quickReplies.length > 0" class="quick-replies">
        <button
          v-for="reply in quickReplies"
          :key="reply.name"
          class="quick-reply-btn"
          @click="handleQuickReply(reply)"
        >
          {{ reply.name }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue';
import Bubble from './Bubble.vue';
import ThinkBubble from './ThinkBubble.vue';
import TypingBubble from './TypingBubble.vue';
import Card from './Card.vue';
import FileCard from './FileCard.vue';
import Button from './Button.vue';
import ThinkingBlock from '@/components/Thinking/ThinkingBlock.vue';

interface ThinkingStep {
  id?: string;
  type: 'reasoning' | 'decision' | 'tool_call' | 'tool_result' | 'approval';
  content?: string;
  tool?: { name: string; args: any };
  result?: any;
  timestamp: number;
}

interface Message {
  id: string;
  type: 'text' | 'thinking' | 'typing' | 'card' | 'file';
  content?: string;
  position: 'left' | 'right';
  status?: 'pending' | 'sent' | 'error';
  user?: {
    avatar?: string;
    name?: string;
  };
  card?: {
    title: string;
    content: string;
    actions?: Array<{ text: string; value: string }>;
  };
  file?: {
    name: string;
    size: number;
    url: string;
  };
  // Thinking 相关字段
  thinkingSteps?: ThinkingStep[];
  isThinkingActive?: boolean;
}

interface QuickReply {
  name: string;
  value?: string;
  icon?: string;
}

interface Props {
  messages?: Message[];
  placeholder?: string;
  disabled?: boolean;
  quickReplies?: QuickReply[];
  toolbar?: Array<{ icon: string; onClick: () => void }>;
}

const props = withDefaults(defineProps<Props>(), {
  messages: () => [],
  placeholder: '输入消息...',
  disabled: false,
  quickReplies: () => [],
  toolbar: () => [],
});

const emit = defineEmits<{
  send: [message: { type: string; content: string }];
  quickReply: [reply: QuickReply];
  cardAction: [action: { value: string }];
  askUserSubmit: [payload: { requestId: string; answers: Record<string, any> }];
}>();

const inputValue = ref('');
const inputRef = ref<HTMLTextAreaElement>();
const messagesRef = ref<HTMLDivElement>();
const scrollAnchor = ref<HTMLDivElement>();

const canSend = computed(() => {
  return inputValue.value.trim().length > 0 && !props.disabled;
});

const handleSend = () => {
  if (!canSend.value) return;
  
  emit('send', {
    type: 'text',
    content: inputValue.value.trim(),
  });
  
  inputValue.value = '';
  nextTick(() => {
    inputRef.value?.focus();
  });
};

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    handleSend();
  }
};

const handleInput = () => {
  // 自动调整输入框高度
  if (inputRef.value) {
    inputRef.value.style.height = 'auto';
    inputRef.value.style.height = `${inputRef.value.scrollHeight}px`;
  }
};

const handleQuickReply = (reply: QuickReply) => {
  emit('quickReply', reply);
};

const handleCardAction = (action: { value: string }) => {
  emit('cardAction', action);
};

// 自动滚动到底部
watch(() => props.messages, () => {
  nextTick(() => {
    scrollAnchor.value?.scrollIntoView({ behavior: 'smooth' });
  });
}, { deep: true });
</script>

<style scoped>
.chatui-container {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark;
}

.chatui-messages {
  @apply flex-1 overflow-y-auto px-4 py-6 space-y-4;
  scroll-behavior: smooth;
}

.chatui-message {
  @apply flex;
}

.chatui-message-left {
  @apply justify-start;
}

.chatui-message-right {
  @apply justify-end;
}

.chatui-composer {
  @apply border-t border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.composer-input-wrapper {
  @apply flex items-end gap-2 p-4;
}

.composer-toolbar {
  @apply flex gap-1;
}

.composer-input {
  @apply flex-1 min-w-0;
}

.composer-textarea {
  @apply w-full px-4 py-2 bg-gray-50 dark:bg-surface-dark/50 border border-border dark:border-border-dark rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-primary dark:text-text-dark;
  max-height: 120px;
  min-height: 40px;
}

.quick-replies {
  @apply flex gap-2 px-4 pb-4 overflow-x-auto;
}

.quick-reply-btn {
  @apply px-4 py-2 bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-full text-sm font-medium hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors whitespace-nowrap;
}

.scroll-anchor {
  @apply h-px;
}
</style>
