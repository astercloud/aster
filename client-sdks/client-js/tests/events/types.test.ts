/**
 * 事件类型测试
 */

import { describe, it, expect } from "vitest";
import {
  ProgressThinkingEvent,
  ProgressTextChunkEvent,
  ProgressToolStartEvent,
  ProgressToolEndEvent,
  ProgressDoneEvent,
  ProgressErrorEvent,
  ControlToolApprovalRequestEvent,
  ControlToolApprovalResponseEvent,
  ControlPauseEvent,
  ControlResumeEvent,
  MonitorTokenUsageEvent,
  MonitorLatencyEvent,
  MonitorCostEvent,
  MonitorComplianceEvent,
  isProgressEvent,
  isControlEvent,
  isMonitorEvent,
  isEventType,
  StreamEvent,
} from "../../src/events/types";

describe("Event Types", () => {
  describe("Progress Events", () => {
    it("should create thinking event", () => {
      const event: ProgressThinkingEvent = {
        type: "thinking",
        channel: "progress",
        data: { content: "Analyzing the question..." },
      };

      expect(event.type).toBe("thinking");
      expect(event.channel).toBe("progress");
      expect(event.data.content).toBe("Analyzing the question...");
    });

    it("should create text chunk event", () => {
      const event: ProgressTextChunkEvent = {
        type: "text_chunk",
        channel: "progress",
        data: { delta: "Hello", text: "Hello" },
      };

      expect(event.type).toBe("text_chunk");
      expect(event.data.delta).toBe("Hello");
    });

    it("should create tool start event", () => {
      const event: ProgressToolStartEvent = {
        type: "tool_start",
        channel: "progress",
        data: {
          toolName: "calculator",
          params: { operation: "add", numbers: [1, 2] },
        },
      };

      expect(event.type).toBe("tool_start");
      expect(event.data.toolName).toBe("calculator");
    });

    it("should create tool end event", () => {
      const event: ProgressToolEndEvent = {
        type: "tool_end",
        channel: "progress",
        data: { toolName: "calculator", result: 3 },
      };

      expect(event.type).toBe("tool_end");
      expect(event.data.result).toBe(3);
    });

    it("should create done event", () => {
      const event: ProgressDoneEvent = {
        type: "done",
        channel: "progress",
        data: { text: "Task completed" },
      };

      expect(event.type).toBe("done");
      expect(event.data.text).toBe("Task completed");
    });

    it("should create error event", () => {
      const event: ProgressErrorEvent = {
        type: "error",
        channel: "progress",
        data: { error: "Something went wrong" },
      };

      expect(event.type).toBe("error");
      expect(event.data.error).toBe("Something went wrong");
    });
  });

  describe("Control Events", () => {
    it("should create tool approval request event", () => {
      const event: ControlToolApprovalRequestEvent = {
        type: "tool_approval_request",
        channel: "control",
        data: {
          approvalId: "approval-123",
          toolName: "file_delete",
          params: { path: "/important/file.txt" },
        },
      };

      expect(event.type).toBe("tool_approval_request");
      expect(event.data.approvalId).toBe("approval-123");
    });

    it("should create tool approval response event", () => {
      const event: ControlToolApprovalResponseEvent = {
        type: "tool_approval_response",
        channel: "control",
        data: { approvalId: "approval-123", approved: true },
      };

      expect(event.type).toBe("tool_approval_response");
      expect(event.data.approved).toBe(true);
    });

    it("should create pause event", () => {
      const event: ControlPauseEvent = {
        type: "pause",
        channel: "control",
        data: { reason: "User requested pause" },
      };

      expect(event.type).toBe("pause");
      expect(event.data.reason).toBe("User requested pause");
    });

    it("should create resume event", () => {
      const event: ControlResumeEvent = {
        type: "resume",
        channel: "control",
        data: { timestamp: "2024-01-01T00:00:00Z" },
      };

      expect(event.type).toBe("resume");
      expect(event.data.timestamp).toBe("2024-01-01T00:00:00Z");
    });
  });

  describe("Monitor Events", () => {
    it("should create token usage event", () => {
      const event: MonitorTokenUsageEvent = {
        type: "token_usage",
        channel: "monitor",
        data: {
          promptTokens: 100,
          completionTokens: 50,
          totalTokens: 150,
        },
      };

      expect(event.type).toBe("token_usage");
      expect(event.data.totalTokens).toBe(150);
    });

    it("should create latency event", () => {
      const event: MonitorLatencyEvent = {
        type: "latency",
        channel: "monitor",
        data: { latencyMs: 250, operation: "chat" },
      };

      expect(event.type).toBe("latency");
      expect(event.data.latencyMs).toBe(250);
    });

    it("should create cost event", () => {
      const event: MonitorCostEvent = {
        type: "cost",
        channel: "monitor",
        data: { cost: 0.05, currency: "USD" },
      };

      expect(event.type).toBe("cost");
      expect(event.data.cost).toBe(0.05);
    });

    it("should create compliance event", () => {
      const event: MonitorComplianceEvent = {
        type: "compliance",
        channel: "monitor",
        data: { passed: true, details: "All checks passed" },
      };

      expect(event.type).toBe("compliance");
      expect(event.data.passed).toBe(true);
    });
  });

  describe("Type Guards", () => {
    it("should identify progress events", () => {
      const event: StreamEvent = {
        type: "thinking",
        channel: "progress",
        data: { content: "test" },
      };

      expect(isProgressEvent(event)).toBe(true);
      expect(isControlEvent(event)).toBe(false);
      expect(isMonitorEvent(event)).toBe(false);
    });

    it("should identify control events", () => {
      const event: StreamEvent = {
        type: "pause",
        channel: "control",
        data: { reason: "test" },
      };

      expect(isProgressEvent(event)).toBe(false);
      expect(isControlEvent(event)).toBe(true);
      expect(isMonitorEvent(event)).toBe(false);
    });

    it("should identify monitor events", () => {
      const event: StreamEvent = {
        type: "token_usage",
        channel: "monitor",
        data: { promptTokens: 10, completionTokens: 5, totalTokens: 15 },
      };

      expect(isProgressEvent(event)).toBe(false);
      expect(isControlEvent(event)).toBe(false);
      expect(isMonitorEvent(event)).toBe(true);
    });

    it("should identify specific event types", () => {
      const thinkingEvent: StreamEvent = {
        type: "thinking",
        channel: "progress",
        data: { content: "test" },
      };

      const pauseEvent: StreamEvent = {
        type: "pause",
        channel: "control",
        data: { reason: "test" },
      };

      expect(isEventType(thinkingEvent, "thinking")).toBe(true);
      expect(isEventType(thinkingEvent, "pause")).toBe(false);
      expect(isEventType(pauseEvent, "pause")).toBe(true);
      expect(isEventType(pauseEvent, "thinking")).toBe(false);
    });
  });
});
