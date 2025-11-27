<template>
  <div class="ask-user-tool">
    <div class="ask-header">
      <Icon type="help-circle" size="sm" />
      <span>需要您的确认</span>
    </div>

    <div class="questions-container">
      <div
        v-for="(question, qIndex) in questions"
        :key="qIndex"
        class="question-block"
      >
        <div class="question-header">
          <span class="question-label">{{ question.header }}</span>
        </div>
        <div class="question-text">{{ question.question }}</div>

        <div class="options-list">
          <label
            v-for="(option, oIndex) in question.options"
            :key="oIndex"
            :class="['option-item', { 
              'selected': isSelected(qIndex, oIndex),
              'multi-select': question.multi_select 
            }]"
          >
            <input
              v-if="question.multi_select"
              type="checkbox"
              :checked="isSelected(qIndex, oIndex)"
              @change="toggleOption(qIndex, oIndex)"
              class="option-input"
            />
            <input
              v-else
              type="radio"
              :name="`question-${qIndex}`"
              :checked="isSelected(qIndex, oIndex)"
              @change="selectOption(qIndex, oIndex)"
              class="option-input"
            />
            <div class="option-content">
              <span class="option-label">{{ option.label }}</span>
              <span class="option-description">{{ option.description }}</span>
            </div>
          </label>

          <!-- Other 选项 -->
          <label
            :class="['option-item', 'other-option', { 
              'selected': isOtherSelected(qIndex) 
            }]"
          >
            <input
              type="radio"
              :name="`question-${qIndex}`"
              :checked="isOtherSelected(qIndex)"
              @change="selectOther(qIndex)"
              class="option-input"
            />
            <div class="option-content">
              <span class="option-label">其他</span>
              <input
                v-if="isOtherSelected(qIndex)"
                v-model="otherInputs[qIndex]"
                type="text"
                placeholder="请输入..."
                class="other-input"
                @click.stop
              />
              <span v-else class="option-description">提供自定义答案</span>
            </div>
          </label>
        </div>
      </div>
    </div>

    <div class="actions">
      <button
        class="submit-btn"
        :disabled="!canSubmit"
        @click="submitAnswers"
      >
        确认
      </button>
      <button
        class="skip-btn"
        @click="skipQuestions"
      >
        跳过
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import Icon from '../ChatUI/Icon.vue';
import type { Question } from '@/types';

interface Props {
  requestId: string;
  questions: Question[];
  answered?: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  answer: [requestId: string, answers: Record<string, any>];
  skip: [requestId: string];
}>();

// 存储选中的答案
const selectedAnswers = ref<Record<number, number | number[]>>({});
const otherInputs = ref<Record<number, string>>({});
const otherSelected = ref<Record<number, boolean>>({});

const isSelected = (qIndex: number, oIndex: number) => {
  const answer = selectedAnswers.value[qIndex];
  if (Array.isArray(answer)) {
    return answer.includes(oIndex);
  }
  return answer === oIndex;
};

const isOtherSelected = (qIndex: number) => {
  return otherSelected.value[qIndex] === true;
};

const selectOption = (qIndex: number, oIndex: number) => {
  selectedAnswers.value[qIndex] = oIndex;
  otherSelected.value[qIndex] = false;
};

const toggleOption = (qIndex: number, oIndex: number) => {
  const current = selectedAnswers.value[qIndex];
  if (Array.isArray(current)) {
    const idx = current.indexOf(oIndex);
    if (idx >= 0) {
      current.splice(idx, 1);
    } else {
      current.push(oIndex);
    }
  } else {
    selectedAnswers.value[qIndex] = [oIndex];
  }
  otherSelected.value[qIndex] = false;
};

const selectOther = (qIndex: number) => {
  otherSelected.value[qIndex] = true;
  selectedAnswers.value[qIndex] = -1;
};

const canSubmit = computed(() => {
  return props.questions.every((_, qIndex) => {
    if (otherSelected.value[qIndex]) {
      return otherInputs.value[qIndex]?.trim();
    }
    const answer = selectedAnswers.value[qIndex];
    if (Array.isArray(answer)) {
      return answer.length > 0;
    }
    return answer !== undefined && answer >= 0;
  });
});

const submitAnswers = () => {
  const answers: Record<string, any> = {};
  
  props.questions.forEach((question, qIndex) => {
    if (otherSelected.value[qIndex]) {
      answers[qIndex] = { type: 'other', value: otherInputs.value[qIndex] };
    } else {
      const answer = selectedAnswers.value[qIndex];
      if (Array.isArray(answer)) {
        answers[qIndex] = answer.map(i => question.options[i].label);
      } else if (answer !== undefined && answer >= 0) {
        answers[qIndex] = question.options[answer].label;
      }
    }
  });

  emit('answer', props.requestId, answers);
};

const skipQuestions = () => {
  emit('skip', props.requestId);
};
</script>

<style scoped>
.ask-user-tool {
  @apply bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4;
}

.ask-header {
  @apply flex items-center gap-2 text-amber-700 dark:text-amber-300 font-medium mb-3;
}

.questions-container {
  @apply space-y-4;
}

.question-block {
  @apply bg-white dark:bg-gray-800 rounded-lg p-3 border border-gray-200 dark:border-gray-700;
}

.question-header {
  @apply mb-2;
}

.question-label {
  @apply text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide;
}

.question-text {
  @apply text-sm text-gray-800 dark:text-gray-200 mb-3;
}

.options-list {
  @apply space-y-2;
}

.option-item {
  @apply flex items-start gap-3 p-2 rounded-lg border border-gray-200 dark:border-gray-600 cursor-pointer transition-colors;
  @apply hover:bg-gray-50 dark:hover:bg-gray-700;
}

.option-item.selected {
  @apply bg-blue-50 dark:bg-blue-900/30 border-blue-300 dark:border-blue-700;
}

.option-input {
  @apply mt-1 flex-shrink-0;
}

.option-content {
  @apply flex-1;
}

.option-label {
  @apply block text-sm font-medium text-gray-800 dark:text-gray-200;
}

.option-description {
  @apply block text-xs text-gray-500 dark:text-gray-400 mt-0.5;
}

.other-input {
  @apply mt-1 w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded;
  @apply bg-white dark:bg-gray-700 text-gray-800 dark:text-gray-200;
  @apply focus:outline-none focus:ring-2 focus:ring-blue-500;
}

.actions {
  @apply flex gap-2 mt-4;
}

.submit-btn {
  @apply px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium rounded-lg transition-colors;
  @apply disabled:opacity-50 disabled:cursor-not-allowed;
}

.skip-btn {
  @apply px-4 py-2 text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-sm rounded-lg transition-colors;
}
</style>
