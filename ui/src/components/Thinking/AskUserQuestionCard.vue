<template>
  <div class="ask-user-card">
    <div class="card-header">
      <h3 class="card-title">💬 Agent 有问题想问您</h3>
    </div>

    <div class="questions-container">
      <div
        v-for="(question, index) in questions"
        :key="index"
        class="question-item"
      >
        <div class="question-header">
          <span class="question-chip">{{ question.header }}</span>
          <h4 class="question-text">{{ question.question }}</h4>
        </div>

        <!-- 单选模式 -->
        <div v-if="!question.multi_select" class="options-single">
          <label
            v-for="option in question.options"
            :key="option.label"
            class="option-item"
            :class="{ selected: answers[index] === option.label }"
          >
            <input
              type="radio"
              :name="`question-${index}`"
              :value="option.label"
              v-model="answers[index]"
              class="option-radio"
            />
            <div class="option-content">
              <span class="option-label">{{ option.label }}</span>
              <p class="option-description">{{ option.description }}</p>
            </div>
          </label>
        </div>

        <!-- 多选模式 -->
        <div v-else class="options-multi">
          <label
            v-for="option in question.options"
            :key="option.label"
            class="option-item"
            :class="{ selected: multiAnswers[index]?.includes(option.label) }"
          >
            <input
              type="checkbox"
              :value="option.label"
              v-model="multiAnswers[index]"
              class="option-checkbox"
            />
            <div class="option-content">
              <span class="option-label">{{ option.label }}</span>
              <p class="option-description">{{ option.description }}</p>
            </div>
          </label>
        </div>

        <!-- Other 选项 -->
        <div class="other-option">
          <label class="other-label">
            <input
              type="checkbox"
              v-model="otherEnabled[index]"
              class="other-checkbox"
            />
            <span>其他（自定义输入）</span>
          </label>
          <input
            v-if="otherEnabled[index]"
            v-model="otherAnswers[index]"
            placeholder="请输入您的答案..."
            class="other-input"
          />
        </div>
      </div>
    </div>

    <div class="card-actions">
      <button
        @click="handleSubmit"
        class="btn-submit"
        :disabled="!isValid || answered"
      >
        {{ answered ? '已提交' : '提交回答' }}
      </button>
      <span v-if="!isValid" class="validation-hint">
        请至少选择一个选项或填写自定义答案
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, watch } from 'vue';
import type { Question } from '@/types/message';
import type { PropType } from 'vue';

export default defineComponent({
  name: 'AskUserQuestionCard',
  props: {
    requestId: {
      type: String,
      required: true,
    },
    questions: {
      type: Array as PropType<Question[]>,
      required: true,
    },
    answered: {
      type: Boolean,
      default: false,
    },
  },
  emits: {
    submit: (payload: { requestId: string; answers: Record<string, any> }) => {
      return typeof payload.requestId === 'string' && typeof payload.answers === 'object';
    },
  },
  setup(props, { emit }) {
    // 单选答案
    const answers = ref<Record<number, string>>({});

    // 多选答案
    const multiAnswers = ref<Record<number, string[]>>({});

    // Other 选项开关
    const otherEnabled = ref<Record<number, boolean>>({});

    // Other 选项答案
    const otherAnswers = ref<Record<number, string>>({});

    // 初始化多选答案数组
    props.questions.forEach((q, index) => {
      if (q.multi_select && !multiAnswers.value[index]) {
        multiAnswers.value[index] = [];
      }
      otherEnabled.value[index] = false;
      otherAnswers.value[index] = '';
    });

    // 验证所有问题是否都有答案
    const isValid = computed(() => {
      return props.questions.every((q, index) => {
        // 如果启用了 Other 并且有内容，则有效
        if (otherEnabled.value[index] && otherAnswers.value[index]?.trim()) {
          return true;
        }

        // 多选模式：至少选择一个
        if (q.multi_select) {
          return (multiAnswers.value[index]?.length ?? 0) > 0;
        }

        // 单选模式：必须选择一个
        return !!answers.value[index];
      });
    });

    // 当启用 Other 时，清空其他选项
    watch(
      otherEnabled,
      (newVal) => {
        Object.keys(newVal).forEach((indexStr) => {
          const index = parseInt(indexStr);
          if (newVal[index] && props.questions[index]) {
            // 清空单选或多选
            if (props.questions[index]!.multi_select) {
              multiAnswers.value[index] = [];
            } else {
              delete answers.value[index];
            }
          }
        });
      },
      { deep: true }
    );

    // 当选择其他选项时，禁用 Other
    watch(
      [answers, multiAnswers],
      () => {
        props.questions.forEach((q, index) => {
          if (q.multi_select) {
            if ((multiAnswers.value[index]?.length ?? 0) > 0) {
              otherEnabled.value[index] = false;
              otherAnswers.value[index] = '';
            }
          } else {
            if (answers.value[index]) {
              otherEnabled.value[index] = false;
              otherAnswers.value[index] = '';
            }
          }
        });
      },
      { deep: true }
    );

    const handleSubmit = () => {
      if (!isValid.value || props.answered) return;

      const finalAnswers: Record<string, any> = {};

      props.questions.forEach((q, index) => {
        // 优先使用 Other 答案
        if (otherEnabled.value[index] && otherAnswers.value[index]?.trim()) {
          finalAnswers[index] = otherAnswers.value[index];
        } else if (q.multi_select) {
          finalAnswers[index] = multiAnswers.value[index] || [];
        } else {
          finalAnswers[index] = answers.value[index];
        }
      });

      emit('submit', {
        requestId: props.requestId,
        answers: finalAnswers,
      });
    };

    return {
      answers,
      multiAnswers,
      otherEnabled,
      otherAnswers,
      isValid,
      handleSubmit,
    };
  },
});
</script>

<style scoped>
.ask-user-card {
  @apply mt-4 bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20
         border border-blue-200 dark:border-blue-800 rounded-xl shadow-lg overflow-hidden;
}

.card-header {
  @apply px-6 py-4 bg-blue-100 dark:bg-blue-900/30 border-b border-blue-200 dark:border-blue-800;
}

.card-title {
  @apply text-lg font-bold text-blue-900 dark:text-blue-100;
}

.questions-container {
  @apply p-6 space-y-6;
}

.question-item {
  @apply space-y-3;
}

.question-header {
  @apply mb-4;
}

.question-chip {
  @apply inline-block px-3 py-1 bg-blue-200 dark:bg-blue-800 text-blue-800 dark:text-blue-200
         text-xs font-bold rounded-full mb-2;
}

.question-text {
  @apply text-base font-bold text-gray-900 dark:text-gray-100;
}

.options-single,
.options-multi {
  @apply space-y-2;
}

.option-item {
  @apply flex items-start gap-3 p-4 border-2 border-blue-200 dark:border-blue-700
         rounded-lg cursor-pointer transition-all duration-200
         hover:border-blue-400 dark:hover:border-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20;
}

.option-item.selected {
  @apply border-blue-500 dark:border-blue-400 bg-blue-100 dark:bg-blue-900/30;
}

.option-radio,
.option-checkbox {
  @apply mt-1 flex-shrink-0 w-4 h-4 cursor-pointer;
}

.option-content {
  @apply flex-1;
}

.option-label {
  @apply font-semibold text-gray-900 dark:text-gray-100 block mb-1;
}

.option-description {
  @apply text-sm text-gray-700 dark:text-gray-300;
}

.other-option {
  @apply mt-3 p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-600;
}

.other-label {
  @apply flex items-center gap-2 cursor-pointer mb-2;
}

.other-checkbox {
  @apply w-4 h-4 cursor-pointer;
}

.other-label span {
  @apply text-sm font-medium text-gray-700 dark:text-gray-300;
}

.other-input {
  @apply w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
         bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
         focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
         transition-colors;
}

.card-actions {
  @apply px-6 py-4 bg-gray-50 dark:bg-gray-800/50 border-t border-blue-200 dark:border-blue-800
         flex items-center justify-between;
}

.btn-submit {
  @apply px-6 py-2 bg-blue-600 text-white font-bold rounded-lg
         hover:bg-blue-700 active:bg-blue-800
         disabled:bg-gray-300 disabled:text-gray-500 disabled:cursor-not-allowed
         transition-colors duration-200 shadow-md hover:shadow-lg;
}

.validation-hint {
  @apply text-xs text-gray-500 dark:text-gray-400 italic;
}
</style>
