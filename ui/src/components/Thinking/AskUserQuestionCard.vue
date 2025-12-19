<template>
  <div class="ask-user-card">
    <!-- Header -->
    <div class="card-header">
      <div
        class="header-icon"
        :class="isSubmitted && isSubmitting ? 'submitting' : (isSubmitted ? 'submitted' : (isCompleted ? 'completed' : 'pending'))"
      >
        <div v-if="isSubmitted && isSubmitting" class="loading-spinner"></div>
        <svg v-else-if="isSubmitted || isCompleted" class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
        </svg>
        <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
      </div>
      <div class="header-content">
        <h3 class="header-title">
          {{ isSubmitted ? '回答已提交' : (isCompleted ? '已完成回答' : '请回答以下问题') }}
        </h3>
        <p class="header-desc">
          {{ isSubmitted && isSubmitting ? '正在提交回答...' : (isSubmitted ? '正在等待 AI 处理...' : 'AI 需要更多信息来继续') }}
        </p>
      </div>
    </div>

    <!-- Questions -->
    <div class="questions-list">
      <div
        v-for="(question, idx) in questions"
        :key="idx"
        class="question-card"
      >
        <!-- Question Header -->
        <div v-if="question.header" class="question-header">
          {{ question.header }}
        </div>

        <!-- Question Text -->
        <h4
          class="question-text"
          :class="{ 'answered': isQuestionAnswered(idx) }"
        >
          {{ question.question }}
        </h4>

        <!-- Options (if available) -->
        <div v-if="question.options && question.options.length > 0" class="options-list">
          <button
            v-for="option in question.options"
            :key="option.label"
            class="option-btn"
            :class="{
              'selected': getSelectedValue(idx) === option.label,
              'disabled': isQuestionAnswered(idx) && getSelectedValue(idx) !== option.label
            }"
            :disabled="isQuestionAnswered(idx) || isSubmitted"
            @click="handleOptionSelect(idx, option.label)"
          >
            <div
              class="option-label"
              :class="{ 'selected': getSelectedValue(idx) === option.label }"
            >
              <svg
                v-if="getSelectedValue(idx) === option.label"
                class="w-4 h-4"
                fill="currentColor"
                viewBox="0 0 24 24"
              >
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
              </svg>
              <span v-else>{{ option.label.charAt(0).toUpperCase() }}</span>
            </div>
            <span class="option-text">{{ option.label }}</span>
          </button>
        </div>

        <!-- Text Input (if no options) -->
        <div v-else class="text-input-wrapper">
          <textarea
            v-model="textInputs[idx]"
            class="text-input"
            :class="{ 'answered': isQuestionAnswered(idx) }"
            :disabled="isQuestionAnswered(idx)"
            placeholder="请输入您的回答..."
            rows="3"
            @keydown.ctrl.enter="handleTextSubmit(idx)"
          />
          <button
            v-if="!isQuestionAnswered(idx)"
            class="submit-btn"
            :disabled="!textInputs[idx]?.trim()"
            @click="handleTextSubmit(idx)"
          >
            提交
          </button>
          <div v-else class="answered-badge">
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
            </svg>
            <span>已回答</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

export interface AskUserOption {
  label: string
  description?: string
}

export interface AskUserQuestion {
  question: string
  header?: string
  options?: AskUserOption[]
  multi_select?: boolean
}

export interface AskUserRequest {
  requestId: string
  questions: AskUserQuestion[]
  timestamp?: string
  answers?: Record<string, string>
}

interface Props {
  request: AskUserRequest
  isSubmitting?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isSubmitting: false
})

const emit = defineEmits<{
  (e: 'answer', requestId: string, answers: Record<string, string>): void
}>()

// 简化：使用 questions 作为 computed
const questions = computed(() => props.request.questions || [])

// Track answers per question
const answers = ref<Record<number, string>>({})
const textInputs = ref<Record<number, string>>({})
const isSubmitted = ref(false)

// 从历史记录恢复选中状态
const initFromHistory = () => {
  if (props.request.answers) {
    const restoredAnswers: Record<number, string> = {}
    props.request.questions.forEach((q, idx) => {
      const answer = props.request.answers?.[q.question]
      if (answer) {
        restoredAnswers[idx] = answer
      }
    })
    if (Object.keys(restoredAnswers).length > 0) {
      answers.value = restoredAnswers
      isSubmitted.value = true
    }
  }
}

// 重置状态
const resetState = () => {
  answers.value = {}
  textInputs.value = {}
  isSubmitted.value = false
}

// 初始化
initFromHistory()

// 监听 request 变化，重置状态
watch(() => props.request, (newRequest, oldRequest) => {
  if (newRequest?.requestId !== oldRequest?.requestId) {
    resetState()
    initFromHistory()
  }
}, { deep: true, immediate: false })

const isCompleted = computed(() => {
  return questions.value.every((_, idx) => answers.value[idx])
})

const handleOptionSelect = (questionIdx: number, value: string) => {
  if (isSubmitted.value) return
  if (answers.value[questionIdx]) return

  answers.value = { ...answers.value, [questionIdx]: value }
  checkAndSubmit()
}

const handleTextSubmit = (questionIdx: number) => {
  if (isSubmitted.value) return
  const text = textInputs.value[questionIdx]?.trim()
  if (!text || answers.value[questionIdx]) return

  answers.value = { ...answers.value, [questionIdx]: text }
  checkAndSubmit()
}

const checkAndSubmit = () => {
  if (isCompleted.value && !isSubmitted.value) {
    isSubmitted.value = true
    const formattedAnswers: Record<string, string> = {}
    questions.value.forEach((q, idx) => {
      const answer = answers.value[idx]
      if (answer) {
        formattedAnswers[q.question] = answer
      }
    })
    emit('answer', props.request.requestId, formattedAnswers)
  }
}

const isQuestionAnswered = (idx: number) => {
  return !!answers.value[idx]
}

const getSelectedValue = (idx: number) => {
  return answers.value[idx]
}
</script>

<style scoped>
.ask-user-card {
  width: 100%;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border: 1px solid #bae6fd;
  border-radius: 12px;
  padding: 16px;
  margin: 8px 0 16px;
}

.card-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}

.header-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-top: 2px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  transition: all 0.2s;
}

.header-icon.pending {
  background: white;
  border: 1px solid #bae6fd;
  color: #0ea5e9;
}

.header-icon.completed {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #10b981;
}

.header-icon.submitting {
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  color: #3b82f6;
}

.header-icon.submitted {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #10b981;
  animation: pulse 1.5s ease-in-out infinite;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(59, 130, 246, 0.2);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.header-content {
  flex: 1;
}

.header-title {
  font-size: 15px;
  font-weight: 700;
  color: #0c4a6e;
  margin: 0 0 4px;
  line-height: 1.3;
}

.header-desc {
  font-size: 12px;
  color: #0369a1;
  margin: 0;
}

.questions-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.question-card {
  background: white;
  border: 1px solid #e0f2fe;
  border-radius: 10px;
  padding: 14px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

.question-header {
  font-size: 11px;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
}

.question-text {
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  margin: 0 0 12px;
  line-height: 1.5;
  transition: color 0.2s;
}

.question-text.answered {
  color: #9ca3af;
}

.options-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.option-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 20px;
  font-size: 13px;
  color: #0369a1;
  cursor: pointer;
  transition: all 0.15s;
}

.option-btn:hover:not(:disabled) {
  background: #0ea5e9;
  color: white;
  border-color: #0ea5e9;
}

.option-btn.selected {
  background: #0ea5e9;
  color: white;
  border-color: #0ea5e9;
}

.option-btn.disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.option-label {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  background: rgba(255, 255, 255, 0.3);
  transition: all 0.15s;
}

.option-label.selected {
  background: rgba(255, 255, 255, 0.9);
  color: #0ea5e9;
}

.option-text {
  font-weight: 500;
}

.text-input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.text-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  outline: none;
  transition: border-color 0.15s;
  font-family: inherit;
}

.text-input:focus {
  border-color: #0ea5e9;
}

.text-input.answered {
  background: #f9fafb;
  color: #6b7280;
}

.text-input::placeholder {
  color: #9ca3af;
}

.submit-btn {
  align-self: flex-end;
  padding: 8px 16px;
  background: #0ea5e9;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.submit-btn:hover:not(:disabled) {
  background: #0284c7;
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.answered-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  align-self: flex-end;
  color: #10b981;
  font-size: 13px;
  font-weight: 500;
}
</style>
