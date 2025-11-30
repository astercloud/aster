<template>
  <div class="aster-chat flex flex-col h-full bg-background dark:bg-background-dark">
    <!-- Header -->
    <div v-if="config.showHeader" class="chat-header flex-shrink-0 px-4 py-3 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-full bg-primary/10 dark:bg-primary/20 flex items-center justify-center">
            <svg class="w-5 h-5 text-primary dark:text-primary-light" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
            </svg>
          </div>
          <div>
            <h3 class="text-sm font-semibold text-text dark:text-text-dark">{{ config.title || 'AI Agent' }}</h3>
            <p v-if="config.subtitle" class="text-xs text-secondary dark:text-secondary-dark">{{ config.subtitle }}</p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <div v-if="isConnected" class="flex items-center gap-1.5 text-xs text-secondary dark:text-secondary-dark">
            <div class="w-2 h-2 rounded-full bg-green-500 dark:bg-green-400"></div>
            <span class="hidden sm:inline">在线</span>
          </div>
          <div v-else class="flex items-center gap-1.5 text-xs text-secondary dark:text-secondary-dark">
            <div class="w-2 h-2 rounded-full bg-red-500 dark:bg-red-400 animate-pulse"></div>
            <span class="hidden sm:inline">离线</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Messages -->
    <MessageList
      ref="messageListRef"
      :messages="messages"
      :is-typing="isTyping"
      :config="config"
      class="flex-1"
    />

    <!-- Quick Replies -->
    <QuickReplies
      v-if="quickReplies.length > 0"
      :replies="quickReplies"
      @select="handleQuickReply"
      class="flex-shrink-0"
    />

    <!-- Composer -->
    <Composer
      v-model="inputText"
      :placeholder="config.placeholder"
      :disabled="!isConnected || isTyping"
      :enable-voice="config.enableVoice"
      :enable-image="config.enableImage"
      @send="handleSend"
      @voice="handleVoice"
      @image="handleImage"
      class="flex-shrink-0"
    />
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, watch, onMounted } from 'vue';
import MessageList from './MessageList.vue';
import QuickReplies from './QuickReplies.vue';
import Composer from './Composer.vue';
import { useChat } from '@/composables/useChat';
import type { ChatConfig, Message, QuickReply } from '@/types';

/**
 * Chat 组件 - AI Agent 聊天界面
 *
 * @example
 * ```vue
 * <Chat :config="chatConfig" @send="onSend" @error="onError" />
 * ```
 */
export default defineComponent({
  name: 'Chat',

  components: {
    MessageList,
    QuickReplies,
    Composer,
  },

  props: {
    /**
     * Chat 配置
     */
    config: {
      type: Object as () => ChatConfig,
      default: (): ChatConfig => ({
        showHeader: true,
        placeholder: '输入消息...',
        enableVoice: false,
        enableImage: false,
      }),
    },
  },

  emits: {
    /**
     * 发送消息事件
     */
    send: (message: Message) => true,
    /**
     * 接收消息事件
     */
    receive: (message: Message) => true,
    /**
     * 错误事件
     */
    error: (error: Error) => true,
  },

  setup(props, { emit }) {
    // Use chat composable
    const {
      messages,
      isTyping,
      isConnected,
      sendMessage,
      sendImage,
    } = useChat(props.config);

    const inputText = ref('');
    const messageListRef = ref<{ scrollToBottom: () => void } | null>(null);
    const quickReplies = ref<QuickReply[]>(props.config.quickReplies || []);

    // Handle send message
    const handleSend = async () => {
      if (!inputText.value.trim()) return;

      const text = inputText.value;
      inputText.value = '';

      try {
        await sendMessage(text);
        // Clear quick replies after first message
        if (messages.value.length === 0) {
          quickReplies.value = [];
        }
      } catch (error) {
        console.error('Send message error:', error);
        emit('error', error as Error);
      }
    };

    // Handle quick reply
    const handleQuickReply = async (reply: QuickReply) => {
      inputText.value = reply.text;
      await handleSend();
    };

    // Handle voice input
    const handleVoice = (blob: Blob) => {
      // TODO: Implement voice input
      console.log('Voice input:', blob);
    };

    // Handle image upload
    const handleImage = async (file: File) => {
      try {
        await sendImage(file);
      } catch (error) {
        console.error('Send image error:', error);
        emit('error', error as Error);
      }
    };

    // Auto scroll to bottom
    watch(messages, () => {
      messageListRef.value?.scrollToBottom();
    }, { deep: true });

    // Initialize
    onMounted(() => {
      // Show welcome message
      if (props.config.welcomeMessage && messages.value.length === 0) {
        // Welcome message will be added by useChat
      }
    });

    return {
      messages,
      isTyping,
      isConnected,
      inputText,
      messageListRef,
      quickReplies,
      handleSend,
      handleQuickReply,
      handleVoice,
      handleImage,
    };
  },
});
</script>

<style scoped>
.aster-chat {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  max-height: 100vh;
}
</style>
