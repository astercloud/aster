/**
 * AsterUI - Universal AI Agent Interface
 *
 * @version 1.0.0
 * @author Kiro AI
 * @license MIT
 */

// ============================================
// Core Chat Components (Phase 1)
// ============================================
export { default as Chat } from "./components/Chat/Chat.vue";
export { default as MessageList } from "./components/Chat/MessageList.vue";
export { default as MessageBubble } from "./components/Chat/MessageBubble.vue";
export { default as Composer } from "./components/Chat/Composer.vue";
export { default as QuickReplies } from "./components/Chat/QuickReplies.vue";

// ============================================
// Message Type Components (Phase 2)
// ============================================
export { default as ImageMessage } from "./components/Message/ImageMessage.vue";
export { default as CardMessage } from "./components/Message/CardMessage.vue";
export { default as ListMessage } from "./components/Message/ListMessage.vue";
export { default as ThinkingMessage } from "./components/Message/ThinkingMessage.vue";
export { default as SystemMessage } from "./components/Message/SystemMessage.vue";

// ============================================
// Common Components (Phase 2)
// ============================================
export { default as Modal } from "./components/Common/Modal.vue";
export { default as LoadingSpinner } from "./components/Common/LoadingSpinner.vue";
export { default as ErrorBoundary } from "./components/Common/ErrorBoundary.vue";
export { default as EmptyState } from "./components/Common/EmptyState.vue";
export { default as NotificationContainer } from "./components/Common/NotificationContainer.vue";

// ============================================
// Agent Components (Phase 2)
// ============================================
export { default as AgentList } from "./components/Agent/AgentList.vue";
export { default as AgentCard } from "./components/Agent/AgentCard.vue";
export { default as AgentForm } from "./components/Agent/AgentForm.vue";

// ============================================
// Room Components (Phase 2)
// ============================================
export { default as RoomList } from "./components/Room/RoomList.vue";

// ============================================
// Workflow Components (Phase 2)
// ============================================
export { default as WorkflowList } from "./components/Workflow/WorkflowList.vue";
export { default as WorkflowCard } from "./components/Workflow/WorkflowCard.vue";

// ============================================
// Functional Composables (Phase 1)
// ============================================
export { useChat } from "./composables/useChat";
export { useMessage, useMessageList } from "./composables/useMessage";
export { useScroll } from "./composables/useScroll";
export { useDarkMode } from "./composables/useDarkMode";

// ============================================
// Utility Composables (Phase 2)
// ============================================
export { useLocalStorage } from "./composables/useLocalStorage";
export { useDebounce, useDebounceFn } from "./composables/useDebounce";
export { useThrottle, useThrottleFn } from "./composables/useThrottle";
export { useNotification } from "./composables/useNotification";
export { useClipboard } from "./composables/useClipboard";
export { useWebSocket } from "./composables/useWebSocket";

// ============================================
// Utility Functions
// ============================================
export { formatTime, truncate, formatFileSize } from "./utils/format";
export { renderMarkdown, extractPlainText, hasCodeBlock } from "./utils/markdown";

// ============================================
// Type Definitions
// ============================================
export type {
  // Message Types
  Message,
  TextMessage,
  ImageMessage as ImageMessageType,
  CardMessage as CardMessageType,
  ListMessage as ListMessageType,
  ThinkingMessage as ThinkingMessageType,
  SystemMessage as SystemMessageType,
  MessageType,
  MessageRole,
  MessageStatus,

  // Chat Types
  ChatConfig,
  QuickReply,

  // Agent Types
  Agent,

  // Room Types
  Room,
  RoomMember,

  // Workflow Types
  Workflow,
  WorkflowStep,

  // API Types
  ApiResponse,
} from "./types";

// ============================================
// Version Info
// ============================================
export const version = "0.15.0";
export const name = "AsterUI";

// ============================================
// Project Components
// ============================================
export { default as ProjectCard } from "./components/Project/ProjectCard.vue";
export { default as ProjectList } from "./components/Project/ProjectList.vue";
