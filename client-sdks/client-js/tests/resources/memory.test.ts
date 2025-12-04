/**
 * Memory 资源类测试
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { MemoryResource } from "../../src/resources/memory";
import type { ClientOptions } from "../../src/resources/base";

describe("MemoryResource", () => {
  let memory: MemoryResource;
  let mockFetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    // Mock fetch
    mockFetch = vi.fn();

    const options: ClientOptions = {
      baseUrl: "http://localhost:8080",
      fetchImpl: mockFetch as any,
    };

    memory = new MemoryResource(options);
  });

  describe("Working Memory", () => {
    describe("set", () => {
      it("should set working memory with default scope", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () => JSON.stringify({}),
        });

        await memory.working.set("user_preference", {
          theme: "dark",
          language: "zh-CN",
        });

        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/working",
          expect.objectContaining({
            method: "PUT",
            body: JSON.stringify({
              key: "user_preference",
              value: { theme: "dark", language: "zh-CN" },
              scope: "thread",
            }),
          }),
        );
      });

      it("should set working memory with custom scope", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () => JSON.stringify({}),
        });

        await memory.working.set(
          "global_config",
          { version: "1.0" },
          {
            scope: "resource",
            ttl: 3600,
          },
        );

        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/working",
          expect.objectContaining({
            method: "PUT",
            body: JSON.stringify({
              key: "global_config",
              value: { version: "1.0" },
              scope: "resource",
              ttl: 3600,
            }),
          }),
        );
      });

      it("should set working memory with JSON Schema", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () => JSON.stringify({}),
        });

        await memory.working.set(
          "validated_data",
          { count: 42 },
          {
            schema: {
              type: "object",
              properties: {
                count: { type: "number" },
              },
              required: ["count"],
            },
          },
        );

        const call = mockFetch.mock.calls[0];
        const body = JSON.parse(call[1].body);
        expect(body.schema).toBeDefined();
        expect(body.schema.type).toBe("object");
      });
    });

    describe("get", () => {
      it("should get working memory value", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () =>
            JSON.stringify({
              value: { theme: "dark" },
            }),
        });

        const result = await memory.working.get("user_preference");

        expect(result).toEqual({ theme: "dark" });
        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/working/user_preference",
          expect.objectContaining({
            method: "GET",
          }),
        );
      });

      it("should get working memory with scope", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () =>
            JSON.stringify({
              value: { version: "1.0" },
            }),
        });

        await memory.working.get("global_config", "resource");

        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/working/global_config?scope=resource",
          expect.any(Object),
        );
      });
    });

    describe("delete", () => {
      it("should delete working memory", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 204,
          text: async () => "",
        });

        await memory.working.delete("user_preference");

        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/working/user_preference",
          expect.objectContaining({
            method: "DELETE",
          }),
        );
      });

      it("should delete working memory with scope", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 204,
          text: async () => "",
        });

        await memory.working.delete("global_config", "resource");

        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/working/global_config?scope=resource",
          expect.any(Object),
        );
      });
    });

    describe("list", () => {
      it("should list all working memory", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () =>
            JSON.stringify({
              items: [
                {
                  key: "pref1",
                  value: "val1",
                  scope: "thread",
                  createdAt: "2024-01-01",
                },
                {
                  key: "pref2",
                  value: "val2",
                  scope: "thread",
                  createdAt: "2024-01-01",
                },
              ],
            }),
        });

        const result = await memory.working.list();

        expect(result).toEqual({
          pref1: "val1",
          pref2: "val2",
        });
      });

      it("should list working memory by scope", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () =>
            JSON.stringify({
              items: [
                {
                  key: "config1",
                  value: "val1",
                  scope: "resource",
                  createdAt: "2024-01-01",
                },
              ],
            }),
        });

        await memory.working.list("resource");

        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/working?scope=resource",
          expect.any(Object),
        );
      });
    });

    describe("clear", () => {
      it("should clear all working memory", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 204,
          text: async () => "",
        });

        await memory.working.clear();

        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/working",
          expect.objectContaining({
            method: "DELETE",
          }),
        );
      });
    });
  });

  describe("Semantic Memory", () => {
    describe("search", () => {
      it("should search semantic memory", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () =>
            JSON.stringify({
              chunks: [
                {
                  id: "chunk-1",
                  content: "Paris is the capital of France",
                  score: 0.95,
                  timestamp: "2024-01-01",
                },
              ],
              total: 1,
            }),
        });

        const results = await memory.semantic.search("capital of France");

        expect(results).toHaveLength(1);
        expect(results[0].content).toBe("Paris is the capital of France");
        expect(results[0].score).toBe(0.95);
      });

      it("should search with options", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () => JSON.stringify({ chunks: [], total: 0 }),
        });

        await memory.semantic.search("test query", {
          limit: 5,
          threshold: 0.8,
          filter: { category: "geography" },
        });

        const call = mockFetch.mock.calls[0];
        const body = JSON.parse(call[1].body);
        expect(body.limit).toBe(5);
        expect(body.threshold).toBe(0.8);
        expect(body.filter).toEqual({ category: "geography" });
      });
    });

    describe("store", () => {
      it("should store memory chunk", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 200,
          text: async () => JSON.stringify({ chunkId: "chunk-123" }),
        });

        const chunkId = await memory.semantic.store(
          "Paris is the capital of France",
          { source: "wikipedia", category: "geography" },
        );

        expect(chunkId).toBe("chunk-123");

        const call = mockFetch.mock.calls[0];
        const body = JSON.parse(call[1].body);
        expect(body.content).toBe("Paris is the capital of France");
        expect(body.metadata).toEqual({
          source: "wikipedia",
          category: "geography",
        });
      });
    });

    describe("delete", () => {
      it("should delete memory chunk", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 204,
          text: async () => "",
        });

        await memory.semantic.delete("chunk-123");

        expect(mockFetch).toHaveBeenCalledWith(
          "http://localhost:8080/v1/memory/semantic/chunk-123",
          expect.objectContaining({
            method: "DELETE",
          }),
        );
      });

      it("should batch delete memory chunks", async () => {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 204,
          text: async () => "",
        });

        await memory.semantic.deleteBatch(["chunk-1", "chunk-2", "chunk-3"]);

        const call = mockFetch.mock.calls[0];
        const body = JSON.parse(call[1].body);
        expect(body.chunkIds).toEqual(["chunk-1", "chunk-2", "chunk-3"]);
      });
    });
  });

  describe("Provenance", () => {
    it("should get memory provenance", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({
            memoryId: "mem-123",
            provenance: {
              source: "user_input",
              confidence: 0.95,
              timestamp: "2024-01-01T00:00:00Z",
            },
          }),
      });

      const result = await memory.getProvenance("mem-123");

      expect(result.memoryId).toBe("mem-123");
      expect(result.provenance.source).toBe("user_input");
      expect(result.provenance.confidence).toBe(0.95);
    });

    it("should get memory lineage", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({
            lineage: [
              {
                source: "user_input",
                confidence: 1.0,
                timestamp: "2024-01-01",
              },
              {
                source: "inference",
                confidence: 0.9,
                timestamp: "2024-01-02",
                parentId: "mem-1",
              },
            ],
          }),
      });

      const lineage = await memory.getLineage("mem-123");

      expect(lineage).toHaveLength(2);
      expect(lineage[0].source).toBe("user_input");
      expect(lineage[1].source).toBe("inference");
    });
  });

  describe("Consolidation", () => {
    it("should trigger memory consolidation", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({
            jobId: "job-123",
            status: "pending",
            startedAt: "2024-01-01T00:00:00Z",
          }),
      });

      const result = await memory.consolidate({
        strategy: "summarize",
        llmProvider: "anthropic",
      });

      expect(result.jobId).toBe("job-123");
      expect(result.status).toBe("pending");
    });

    it("should get consolidation status", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({
            jobId: "job-123",
            status: "completed",
            progress: 100,
          }),
      });

      const status = await memory.getConsolidationStatus("job-123");

      expect(status.status).toBe("completed");
      expect(status.progress).toBe(100);
    });

    it("should cancel consolidation", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
        text: async () => "",
      });

      await memory.cancelConsolidation("job-123");

      expect(mockFetch).toHaveBeenCalledWith(
        "http://localhost:8080/v1/memory/consolidation/job-123",
        expect.objectContaining({
          method: "DELETE",
        }),
      );
    });
  });

  describe("Stats and Management", () => {
    it("should get memory stats", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({
            workingMemory: {
              threadCount: 10,
              resourceCount: 5,
              totalSize: 1024,
            },
            semanticMemory: {
              chunkCount: 100,
              totalSize: 10240,
            },
          }),
      });

      const stats = await memory.getStats();

      expect(stats.workingMemory.threadCount).toBe(10);
      expect(stats.semanticMemory.chunkCount).toBe(100);
    });

    it("should require confirmation to clear all", async () => {
      await expect(memory.clearAll()).rejects.toThrow("Must confirm");
    });

    it("should clear all memory with confirmation", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
        text: async () => "",
      });

      await memory.clearAll(true);

      const call = mockFetch.mock.calls[0];
      const body = JSON.parse(call[1].body);
      expect(body.confirm).toBe(true);
    });
  });
});
