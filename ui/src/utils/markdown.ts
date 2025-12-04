/**
 * Markdown 渲染工具
 */

import { marked } from "marked";

// 配置 marked
marked.setOptions({
  breaks: true,
  gfm: true,
});

/**
 * 渲染 Markdown 为 HTML
 */
export function renderMarkdown(content: string): string {
  try {
    return marked.parse(content) as string;
  } catch (error) {
    console.error("Markdown render error:", error);
    return content;
  }
}

/**
 * 提取纯文本（去除 Markdown 标记）
 */
export function extractPlainText(markdown: string): string {
  // 移除代码块
  let text = markdown.replace(/```[\s\S]*?```/g, "");
  // 移除行内代码
  text = text.replace(/`[^`]+`/g, "");
  // 移除链接
  text = text.replace(/\[([^\]]+)\]\([^)]+\)/g, "$1");
  // 移除图片
  text = text.replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1");
  // 移除标题标记
  text = text.replace(/^#+\s+/gm, "");
  // 移除加粗和斜体
  text = text.replace(/[*_]{1,2}([^*_]+)[*_]{1,2}/g, "$1");
  // 移除列表标记
  text = text.replace(/^[-*+]\s+/gm, "");

  return text.trim();
}

/**
 * 检测是否包含代码块
 */
export function hasCodeBlock(markdown: string): boolean {
  return /```[\s\S]*?```/.test(markdown);
}

/**
 * 提取代码块
 */
export function extractCodeBlocks(markdown: string): Array<{ language: string; code: string }> {
  const regex = /```(\w+)?\n([\s\S]*?)```/g;
  const blocks: Array<{ language: string; code: string }> = [];

  let match;
  while ((match = regex.exec(markdown)) !== null) {
    blocks.push({
      language: match[1] || "text",
      code: (match[2] ?? "").trim(),
    });
  }

  return blocks;
}
