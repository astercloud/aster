/**
 * useMessage Composable
 * 消息处理和格式化逻辑
 */

import { computed } from "vue";
import type { Message } from "@/types";
import { formatTime, truncate } from "@/utils/format";
import { extractPlainText, hasCodeBlock } from "@/utils/markdown";

export function useMessage(message: Message) {
  // 获取纯文本内容
  const plainText = computed(() => {
    if (message.type === "text") {
      return extractPlainText(message.content.text);
    }
    return "";
  });

  // 检查是否包含代码
  const hasCode = computed(() => {
    if (message.type === "text") {
      return hasCodeBlock(message.content.text);
    }
    return false;
  });

  // 格式化时间
  const formattedTime = computed(() => {
    return formatTime(message.createdAt, "time");
  });

  // 格式化相对时间
  const relativeTime = computed(() => {
    return formatTime(message.createdAt, "relative");
  });

  // 消息预览（用于通知等）
  const preview = computed(() => {
    return truncate(plainText.value, 50);
  });

  // 是否可以复制
  const canCopy = computed(() => {
    return message.type === "text" && message.content.text.length > 0;
  });

  // 是否可以重试
  const canRetry = computed(() => {
    return message.status === "error" && message.role === "user";
  });

  // 是否可以删除
  const canDelete = computed(() => {
    return true; // 所有消息都可以删除
  });

  return {
    plainText,
    hasCode,
    formattedTime,
    relativeTime,
    preview,
    canCopy,
    canRetry,
    canDelete,
  };
}

/**
 * useMessageList Composable
 * 消息列表处理逻辑
 */
export function useMessageList(messages: Message[]) {
  // 按日期分组
  const groupedByDate = computed(() => {
    const groups = new Map<string, Message[]>();

    messages.forEach((message) => {
      const date = formatMessageDate(message.createdAt);
      if (!groups.has(date)) {
        groups.set(date, []);
      }
      groups.get(date)!.push(message);
    });

    return Array.from(groups.entries()).map(([date, msgs]) => ({
      date,
      messages: msgs,
    }));
  });

  // 按发送者分组（连续消息合并）
  const groupedBySender = computed(() => {
    const groups: Array<{ sender: string; messages: Message[] }> = [];
    let currentSender = "";
    let currentGroup: Message[] = [];

    messages.forEach((message) => {
      const sender = message.role;

      if (sender !== currentSender) {
        if (currentGroup.length > 0) {
          groups.push({ sender: currentSender, messages: currentGroup });
        }
        currentSender = sender;
        currentGroup = [message];
      } else {
        currentGroup.push(message);
      }
    });

    if (currentGroup.length > 0) {
      groups.push({ sender: currentSender, messages: currentGroup });
    }

    return groups;
  });

  // 搜索消息
  const searchMessages = (query: string) => {
    if (!query.trim()) return messages;

    const lowerQuery = query.toLowerCase();
    return messages.filter((message) => {
      if (message.type === "text") {
        return message.content.text.toLowerCase().includes(lowerQuery);
      }
      return false;
    });
  };

  // 过滤消息
  const filterByRole = (role: "user" | "assistant" | "system") => {
    return messages.filter((m) => m.role === role);
  };

  const filterByType = (type: Message["type"]) => {
    return messages.filter((m) => m.type === type);
  };

  // 统计信息
  const stats = computed(() => {
    return {
      total: messages.length,
      user: messages.filter((m) => m.role === "user").length,
      assistant: messages.filter((m) => m.role === "assistant").length,
      system: messages.filter((m) => m.role === "system").length,
    };
  });

  return {
    groupedByDate,
    groupedBySender,
    searchMessages,
    filterByRole,
    filterByType,
    stats,
  };
}

// 辅助函数：格式化消息日期
function formatMessageDate(timestamp: number): string {
  const date = new Date(timestamp);
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (date.toDateString() === today.toDateString()) {
    return "今天";
  } else if (date.toDateString() === yesterday.toDateString()) {
    return "昨天";
  } else {
    return date.toLocaleDateString("zh-CN", {
      month: "long",
      day: "numeric",
    });
  }
}
