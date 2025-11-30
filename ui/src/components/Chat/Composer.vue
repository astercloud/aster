<template>
  <div class="composer">
    <div class="composer-inner">
      <!-- Toolbar (optional) -->
      <div v-if="enableVoice || enableImage" class="composer-toolbar">
        <button
          v-if="enableImage"
          @click="handleImageClick"
          class="toolbar-button"
          title="上传图片"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
          </svg>
        </button>

        <button
          v-if="enableVoice"
          @click="toggleVoice"
          :class="['toolbar-button', isRecording && 'recording']"
          title="语音输入"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z"></path>
          </svg>
        </button>
      </div>

      <!-- Input Area -->
      <div class="composer-input-wrapper">
        <textarea
          ref="textareaRef"
          v-model="localValue"
          :placeholder="placeholder"
          :disabled="disabled"
          @keydown="handleKeyDown"
          @input="handleInput"
          class="composer-input"
          rows="1"
        />

        <!-- Character Count -->
        <div v-if="showCharCount && maxLength" class="char-count">
          {{ localValue.length }} / {{ maxLength }}
        </div>
      </div>

      <!-- Send Button -->
      <button
        @click="handleSend"
        :disabled="!canSend"
        class="send-button"
        title="发送 (Enter)"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"></path>
        </svg>
      </button>
    </div>

    <!-- Hidden File Input -->
    <input
      ref="fileInputRef"
      type="file"
      accept="image/*"
      class="hidden"
      @change="handleFileChange"
    />
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, watch, nextTick } from 'vue';

export default defineComponent({
  name: 'Composer',

  props: {
    modelValue: {
      type: String,
      required: true,
    },
    placeholder: {
      type: String,
      default: '输入消息...',
    },
    disabled: {
      type: Boolean,
      default: false,
    },
    enableVoice: {
      type: Boolean,
      default: false,
    },
    enableImage: {
      type: Boolean,
      default: false,
    },
    maxLength: {
      type: Number,
      default: undefined,
    },
    showCharCount: {
      type: Boolean,
      default: false,
    },
  },

  emits: {
    'update:modelValue': (value: string) => true,
    send: () => true,
    voice: (blob: Blob) => true,
    image: (file: File) => true,
  },

  setup(props, { emit, expose }) {
    const textareaRef = ref<HTMLTextAreaElement>();
    const fileInputRef = ref<HTMLInputElement>();
    const localValue = ref(props.modelValue);
    const isRecording = ref(false);

    // Can send if has content and not disabled
    const canSend = computed(() => {
      return localValue.value.trim().length > 0 && !props.disabled;
    });

    // Handle input
    function handleInput() {
      emit('update:modelValue', localValue.value);
      adjustTextareaHeight();
    }

    // Handle key down
    function handleKeyDown(e: KeyboardEvent) {
      // Enter to send (Shift+Enter for new line)
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        if (canSend.value) {
          handleSend();
        }
      }
    }

    // Handle send
    function handleSend() {
      if (!canSend.value) return;
      emit('send');
    }

    // Adjust textarea height
    function adjustTextareaHeight() {
      if (!textareaRef.value) return;

      textareaRef.value.style.height = 'auto';
      const newHeight = Math.min(textareaRef.value.scrollHeight, 120); // Max 120px
      textareaRef.value.style.height = `${newHeight}px`;
    }

    // Handle image upload
    function handleImageClick() {
      fileInputRef.value?.click();
    }

    function handleFileChange(e: Event) {
      const target = e.target as HTMLInputElement;
      const file = target.files?.[0];
      if (file) {
        emit('image', file);
        // Reset input
        target.value = '';
      }
    }

    // Handle voice input
    function toggleVoice() {
      if (isRecording.value) {
        stopRecording();
      } else {
        startRecording();
      }
    }

    function startRecording() {
      // TODO: Implement voice recording
      isRecording.value = true;
      console.log('Start recording...');
    }

    function stopRecording() {
      // TODO: Implement voice recording
      isRecording.value = false;
      console.log('Stop recording...');
    }

    // Watch modelValue changes from parent
    watch(() => props.modelValue, (newValue) => {
      localValue.value = newValue;
      nextTick(() => {
        adjustTextareaHeight();
      });
    });

    // Focus input
    function focus() {
      textareaRef.value?.focus();
    }

    expose({
      focus,
    });

    return {
      textareaRef,
      fileInputRef,
      localValue,
      isRecording,
      canSend,
      handleInput,
      handleKeyDown,
      handleSend,
      handleImageClick,
      handleFileChange,
      toggleVoice,
    };
  },
});
</script>

<style scoped>
.composer {
  @apply p-4 border-t border-border bg-surface;
}

.composer-inner {
  @apply max-w-4xl mx-auto flex items-end gap-2;
}

.composer-toolbar {
  @apply flex items-center gap-1 pb-2;
}

.toolbar-button {
  @apply p-2 text-secondary hover:text-primary hover:bg-background rounded-lg transition-colors;
}

.toolbar-button.recording {
  @apply text-red-500 animate-pulse;
}

.composer-input-wrapper {
  @apply flex-1 relative;
}

.composer-input {
  @apply w-full px-4 py-3 bg-background border border-border rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all;
  @apply text-sm leading-relaxed text-text dark:text-text-dark;
  min-height: 44px;
  max-height: 120px;
  color: #e5e7eb;
}

.composer-input:disabled {
  @apply opacity-50 cursor-not-allowed;
}

.composer-input::placeholder {
  @apply text-secondary;
}

.char-count {
  @apply absolute bottom-2 right-2 text-xs text-secondary pointer-events-none;
}

.send-button {
  @apply p-3 bg-primary text-white rounded-lg hover:bg-primary-hover disabled:opacity-50 disabled:cursor-not-allowed transition-all flex-shrink-0;
}

.send-button:not(:disabled):active {
  @apply scale-95;
}
</style>
