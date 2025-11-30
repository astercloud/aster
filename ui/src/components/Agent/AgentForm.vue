<template>
  <div class="agent-form">
    <form @submit.prevent="handleSubmit" class="form-content">
      <!-- Name -->
      <div class="form-group">
        <label for="name" class="form-label">Agent 名称</label>
        <input
          id="name"
          v-model="formData.name"
          type="text"
          class="form-input"
          placeholder="输入 Agent 名称"
          required
        />
      </div>

      <!-- Description -->
      <div class="form-group">
        <label for="description" class="form-label">描述</label>
        <textarea
          id="description"
          v-model="formData.description"
          class="form-input"
          rows="3"
          placeholder="描述 Agent 的功能和用途"
        />
      </div>

      <!-- Template ID -->
      <div class="form-group">
        <label for="template" class="form-label">模板</label>
        <select
          id="template"
          v-model="formData.template_id"
          class="form-input"
          required
        >
          <option value="">选择模板</option>
          <option value="chat">聊天助手</option>
          <option value="writer">写作助手</option>
          <option value="coder">编程助手</option>
          <option value="analyst">数据分析师</option>
        </select>
      </div>

      <!-- Model Config -->
      <div class="form-section">
        <h3 class="section-title">模型配置</h3>

        <div class="form-row">
          <div class="form-group">
            <label for="provider" class="form-label">提供商</label>
            <select
              id="provider"
              v-model="formData.model_config.provider"
              class="form-input"
            >
              <option value="anthropic">Anthropic</option>
              <option value="openai">OpenAI</option>
              <option value="deepseek">DeepSeek</option>
            </select>
          </div>

          <div class="form-group">
            <label for="model" class="form-label">模型</label>
            <input
              id="model"
              v-model="formData.model_config.model"
              type="text"
              class="form-input"
              placeholder="claude-3-5-sonnet-20241022"
            />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label for="temperature" class="form-label">Temperature</label>
            <input
              id="temperature"
              v-model.number="formData.model_config.temperature"
              type="number"
              step="0.1"
              min="0"
              max="2"
              class="form-input"
            />
          </div>

          <div class="form-group">
            <label for="max_tokens" class="form-label">Max Tokens</label>
            <input
              id="max_tokens"
              v-model.number="formData.model_config.max_tokens"
              type="number"
              class="form-input"
            />
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="form-actions">
        <button
          type="button"
          @click="$emit('cancel')"
          class="btn-secondary"
        >
          取消
        </button>
        <button
          type="submit"
          :disabled="loading"
          class="btn-primary"
        >
          {{ loading ? '创建中...' : (agent ? '更新' : '创建') }}
        </button>
      </div>
    </form>
  </div>
</template>

<script lang="ts">
import { ref, watch, defineComponent } from 'vue';
import type { Agent } from '@/types';
import type { PropType } from 'vue';

export default defineComponent({
  name: 'AgentForm',
  props: {
    agent: {
      type: Object as PropType<Agent>,
      required: false,
      default: undefined
    },
    loading: {
      type: Boolean,
      default: false
    }
  },
  emits: {
    submit: (data: any) => true,
    cancel: () => true
  },
  setup(props, { emit }) {
    const formData = ref({
      name: '',
      description: '',
      template_id: '',
      model_config: {
        provider: 'anthropic',
        model: 'claude-3-5-sonnet-20241022',
        temperature: 1.0,
        max_tokens: 4096,
      },
      metadata: {},
    });

    // Load agent data if editing
    watch(() => props.agent, (agent) => {
      if (agent) {
        formData.value = {
          name: agent.name,
          description: agent.description || '',
          template_id: agent.metadata?.template_id || '',
          model_config: {
            provider: agent.metadata?.provider || 'anthropic',
            model: agent.metadata?.model || 'claude-3-5-sonnet-20241022',
            temperature: agent.metadata?.temperature || 1.0,
            max_tokens: agent.metadata?.max_tokens || 4096,
          },
          metadata: agent.metadata || {},
        };
      }
    }, { immediate: true });

    const handleSubmit = () => {
      emit('submit', formData.value);
    };

    return {
      formData,
      handleSubmit
    };
  }
});
</script>

<style scoped>
.agent-form {
  @apply max-w-2xl mx-auto;
}

.form-content {
  @apply space-y-6;
}

.form-section {
  @apply space-y-4 p-4 bg-surface dark:bg-surface-dark rounded-lg border border-border dark:border-border-dark;
}

.section-title {
  @apply text-lg font-semibold text-text dark:text-text-dark mb-4;
}

.form-group {
  @apply space-y-2;
}

.form-row {
  @apply grid grid-cols-2 gap-4;
}

.form-label {
  @apply block text-sm font-medium text-text dark:text-text-dark;
}

.form-input {
  @apply w-full px-3 py-2 bg-background dark:bg-background-dark border border-border dark:border-border-dark rounded-lg text-text dark:text-text-dark focus:outline-none focus:ring-2 focus:ring-primary dark:focus:ring-primary-light;
}

.form-actions {
  @apply flex justify-end gap-3 pt-4;
}

.btn-primary {
  @apply px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.btn-secondary {
  @apply px-4 py-2 bg-background dark:bg-background-dark hover:bg-border dark:hover:bg-border-dark text-text dark:text-text-dark border border-border dark:border-border-dark rounded-lg font-medium transition-colors;
}
</style>
