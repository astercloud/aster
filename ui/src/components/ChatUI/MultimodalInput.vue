<template>
  <div class="multimodal-input">
    <!-- 图片预览 -->
    <div v-if="selectedImage" class="image-preview">
      <div class="preview-item">
        <img :src="selectedImage.preview" alt="Selected" class="preview-image" />
        <button class="preview-remove" @click="removeImage">
          <Icon type="close" size="sm" />
        </button>
      </div>
    </div>

    <!-- 输入区域 -->
    <div class="input-container">
      <!-- 工具栏 -->
      <div class="input-toolbar">
        <!-- 图片上传 -->
        <input v-if="enableImage" ref="fileInputRef" type="file" accept="image/*" class="hidden" @change="handleImageUpload" />
        <button v-if="enableImage" class="toolbar-button" title="上传图片" @click="fileInputRef?.click()">
          <Icon type="image" size="sm" />
        </button>

        <!-- 语音输入 -->
        <button v-if="enableVoice" :class="['toolbar-button', { 'toolbar-button-active': isListening }]" title="语音输入" @click="toggleVoice">
          <Icon type="mic" size="sm" />
        </button>

        <!-- 文件上传 -->
        <button v-if="enableFile" class="toolbar-button" title="上传文件" @click="handleFileClick">
          <Icon type="attach" size="sm" />
        </button>
      </div>

      <!-- 输入框 -->
      <textarea ref="textareaRef" v-model="inputValue" :placeholder="placeholder" :disabled="disabled" class="input-textarea" @keydown="handleKeyDown" @input="handleInput" />

      <!-- 发送按钮 -->
      <button class="send-button" :disabled="!canSend" @click="handleSend">
        <Icon type="send" size="sm" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from "vue";
import Icon from "./Icon.vue";

interface Props {
  modelValue?: string;
  placeholder?: string;
  disabled?: boolean;
  enableImage?: boolean;
  enableVoice?: boolean;
  enableFile?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: "",
  placeholder: "输入消息...",
  disabled: false,
  enableImage: true,
  enableVoice: true,
  enableFile: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  send: [data: { text: string; image?: { data: string; preview: string } }];
}>();

const inputValue = ref(props.modelValue);
const textareaRef = ref<HTMLTextAreaElement>();
const fileInputRef = ref<HTMLInputElement>();
const isListening = ref(false);
const selectedImage = ref<{ data: string; preview: string } | null>(null);

const canSend = computed(() => {
  return (inputValue.value.trim().length > 0 || selectedImage.value) && !props.disabled;
});

const handleInput = () => {
  emit("update:modelValue", inputValue.value);

  // 自动调整高度
  if (textareaRef.value) {
    textareaRef.value.style.height = "auto";
    textareaRef.value.style.height = `${textareaRef.value.scrollHeight}px`;
  }
};

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault();
    handleSend();
  }
};

const handleSend = () => {
  if (!canSend.value) return;

  emit("send", {
    text: inputValue.value.trim(),
    image: selectedImage.value || undefined,
  });

  // 清空输入
  inputValue.value = "";
  selectedImage.value = null;
  emit("update:modelValue", "");

  // 重置高度
  nextTick(() => {
    if (textareaRef.value) {
      textareaRef.value.style.height = "auto";
    }
  });
};

const handleImageUpload = (e: Event) => {
  const target = e.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = (e) => {
    const result = e.target?.result as string;
    const base64Data = result?.split(",")[1] ?? "";
    selectedImage.value = {
      preview: result ?? "",
      data: base64Data,
    };
  };
  reader.readAsDataURL(file);

  // 重置 input
  target.value = "";
};

const removeImage = () => {
  selectedImage.value = null;
};

const toggleVoice = () => {
  if (isListening.value) {
    isListening.value = false;
    return;
  }

  const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
  if (!SpeechRecognition) {
    alert("您的浏览器不支持语音输入，请尝试 Chrome。");
    return;
  }

  const recognition = new SpeechRecognition();
  recognition.lang = "zh-CN";
  recognition.interimResults = false;
  recognition.maxAlternatives = 1;

  isListening.value = true;

  recognition.onresult = (event: any) => {
    const text = event.results[0][0].transcript;
    inputValue.value += text;
    emit("update:modelValue", inputValue.value);
    isListening.value = false;
  };

  recognition.onerror = (event: any) => {
    console.error("Voice Error", event.error);
    isListening.value = false;
  };

  recognition.onend = () => {
    isListening.value = false;
  };

  recognition.start();
};

const handleFileClick = () => {
  console.log("File upload not implemented");
};
</script>

<style scoped>
.multimodal-input {
  @apply space-y-2;
}

.image-preview {
  @apply px-4 pt-4 flex items-start gap-2;
}

.preview-item {
  @apply relative;
}

.preview-item:hover .preview-remove {
  @apply opacity-100;
}

.preview-image {
  @apply w-16 h-16 object-cover rounded-lg border border-gray-200 dark:border-gray-700;
}

.preview-remove {
  @apply absolute -top-2 -right-2 bg-red-500 hover:bg-red-600 text-white rounded-full p-0.5 shadow-md opacity-0 transition-opacity;
}

.input-container {
  @apply flex items-end gap-2 p-2;
}

.input-toolbar {
  @apply flex items-center gap-1 text-gray-400 dark:text-gray-500;
}

.toolbar-button {
  @apply p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors hover:text-gray-900 dark:hover:text-white;
}

.toolbar-button-active {
  @apply text-red-500 dark:text-red-400 animate-pulse;
}

.input-textarea {
  @apply flex-1 px-4 py-2 bg-transparent text-sm focus:outline-none placeholder:text-gray-400 dark:placeholder:text-gray-500 font-medium resize-none;
  max-height: 120px;
  min-height: 40px;
}

.send-button {
  @apply p-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors;
}

.hidden {
  @apply sr-only;
}
</style>
