package server

import (
	"github.com/astercloud/aster/pkg/a2a"
	"github.com/astercloud/aster/pkg/core"
	"github.com/astercloud/aster/server/handlers"
	"github.com/gin-gonic/gin"
)

// registerAgentRoutes registers all agent-related routes
func (s *Server) registerAgentRoutes(rg *gin.RouterGroup) {
	// Create agent handler
	h := handlers.NewAgentHandler(s.store, s.deps.AgentDeps)

	agents := rg.Group("/agents")
	{
		agents.POST("", h.Create)
		agents.GET("", h.List)
		agents.GET("/:id", h.Get)
		agents.PATCH("/:id", h.Update)
		agents.DELETE("/:id", h.Delete)
		agents.POST("/:id/run", h.Run)
		agents.POST("/:id/send", h.Send)
		agents.POST("/chat", h.Chat)
		agents.POST("/chat/stream", h.StreamChat)
		agents.GET("/:id/status", h.GetStatus)
		agents.GET("/:id/stats", h.GetStats)
		agents.POST("/:id/resume", h.Resume)
	}
}

// registerWebSocketRoutes registers WebSocket routes
// Deprecated: WebSocket routes are now registered in registerRoutes
// func (s *Server) registerWebSocketRoutes(rg *gin.RouterGroup) {
// 	// Create WebSocket handler
// 	h := handlers.NewWebSocketHandler(s.store, s.deps.AgentDeps, s.agentRegistry)
//
// 	ws := rg.Group("/ws")
// 	{
// 		ws.GET("", h.HandleWebSocket)
// 		ws.GET("/stats", h.GetStats)
// 	}
// }

// registerMemoryRoutes registers all memory-related routes
func (s *Server) registerMemoryRoutes(rg *gin.RouterGroup) {
	// Create memory handler
	h := handlers.NewMemoryHandler(s.store)

	memory := rg.Group("/memory")
	{
		// Working memory
		working := memory.Group("/working")
		{
			working.POST("", h.CreateWorkingMemory)
			working.GET("", h.ListWorkingMemory)
			working.GET("/:id", h.GetWorkingMemory)
			working.PATCH("/:id", h.UpdateWorkingMemory)
			working.DELETE("/:id", h.DeleteWorkingMemory)
			working.POST("/clear", h.ClearWorkingMemory)
		}

		// Semantic memory
		semantic := memory.Group("/semantic")
		{
			semantic.POST("", h.CreateSemanticMemory)
			semantic.POST("/search", h.SearchSemanticMemory)
		}

		// Provenance
		memory.GET("/provenance/:id", h.GetProvenance)

		// Consolidation
		memory.POST("/consolidate", h.ConsolidateMemory)
	}
}

// registerSessionRoutes registers all session-related routes
func (s *Server) registerSessionRoutes(rg *gin.RouterGroup) {
	// Create session handler
	h := handlers.NewSessionHandler(s.store)

	sessions := rg.Group("/sessions")
	{
		sessions.POST("", h.Create)
		sessions.GET("", h.List)
		sessions.GET("/:id", h.Get)
		sessions.PATCH("/:id", h.Update)
		sessions.DELETE("/:id", h.Delete)
		sessions.GET("/:id/messages", h.GetMessages)
		sessions.GET("/:id/checkpoints", h.GetCheckpoints)
		sessions.POST("/:id/resume", h.Resume)
		sessions.GET("/:id/stats", h.GetStats)
	}
}

// registerWorkflowRoutes registers all workflow-related routes
func (s *Server) registerWorkflowRoutes(rg *gin.RouterGroup) {
	// Create workflow handler
	h := handlers.NewWorkflowHandler(s.store)

	workflows := rg.Group("/workflows")
	{
		workflows.POST("", h.Create)
		workflows.GET("", h.List)
		workflows.GET("/:id", h.Get)
		workflows.PATCH("/:id", h.Update)
		workflows.DELETE("/:id", h.Delete)
		workflows.POST("/:id/execute", h.Execute)
		workflows.POST("/:id/suspend", h.Suspend)
		workflows.POST("/:id/resume", h.Resume)
		workflows.GET("/:id/executions", h.GetExecutions)
		workflows.GET("/:id/executions/:eid", h.GetExecutionDetails)
	}
}

// registerToolRoutes registers all tool-related routes
func (s *Server) registerToolRoutes(rg *gin.RouterGroup) {
	// Create tool handler
	h := handlers.NewToolHandler(s.store)
	rt := handlers.NewToolRuntimeHandler(s.store, s.agentRegistry)

	tools := rg.Group("/tools")
	{
		tools.POST("", h.Create)
		tools.GET("", h.List)
		tools.GET("/:id", h.Get)
		tools.PATCH("/:id", h.Update)
		tools.DELETE("/:id", h.Delete)
		tools.POST("/:id/execute", h.Execute)
	}

	toolCalls := rg.Group("/tool-calls")
	{
		toolCalls.GET("/running", rt.ListRunning)
		toolCalls.GET("/:id/status", rt.GetStatus)
		toolCalls.GET("/:id/result", rt.GetResult)
	}
}

// registerMiddlewareRoutes registers all middleware-related routes
func (s *Server) registerMiddlewareRoutes(rg *gin.RouterGroup) {
	// Create middleware handler
	h := handlers.NewMiddlewareHandler(s.store)

	middlewares := rg.Group("/middlewares")
	{
		// Basic CRUD
		middlewares.POST("", h.Create)
		middlewares.GET("", h.List)
		middlewares.GET("/:id", h.Get)
		middlewares.PATCH("/:id", h.Update)
		middlewares.DELETE("/:id", h.Delete)

		// Management operations
		middlewares.POST("/:id/enable", h.Enable)
		middlewares.POST("/:id/disable", h.Disable)
		middlewares.POST("/:id/reload", h.Reload)
		middlewares.GET("/:id/stats", h.GetStats)

		// Registry
		registry := middlewares.Group("/registry")
		{
			registry.GET("", h.ListRegistry)
			registry.POST("/:id/install", h.Install)
			registry.DELETE("/:id/uninstall", h.Uninstall)
			registry.GET("/:id/info", h.GetInfo)
			registry.POST("/reload-all", h.ReloadAll)
		}
	}
}

// registerTelemetryRoutes registers all telemetry-related routes
func (s *Server) registerTelemetryRoutes(rg *gin.RouterGroup) {
	// Create telemetry handler
	h := handlers.NewTelemetryHandler(s.store)

	telemetry := rg.Group("/telemetry")
	{
		// Metrics
		telemetry.POST("/metrics", h.RecordMetric)
		telemetry.GET("/metrics", h.ListMetrics)

		// Traces
		telemetry.POST("/traces", h.RecordTrace)
		telemetry.POST("/traces/query", h.QueryTraces)

		// Logs
		telemetry.POST("/logs", h.RecordLog)
		telemetry.POST("/logs/query", h.QueryLogs)
	}
}

// registerEvalRoutes registers all eval-related routes
func (s *Server) registerEvalRoutes(rg *gin.RouterGroup) {
	// Create eval handler
	h := handlers.NewEvalHandler(s.store)

	eval := rg.Group("/eval")
	{
		// Evaluation runs
		eval.POST("/text", h.RunTextEval)
		eval.POST("/session", h.RunSessionEval)
		eval.POST("/batch", h.RunBatchEval)
		eval.POST("/custom", h.RunCustomEval)

		// Evaluation management
		evals := eval.Group("/evals")
		{
			evals.GET("", h.ListEvals)
			evals.GET("/:id", h.GetEval)
			evals.DELETE("/:id", h.DeleteEval)
		}

		// Benchmarks
		benchmarks := eval.Group("/benchmarks")
		{
			benchmarks.POST("", h.CreateBenchmark)
			benchmarks.GET("", h.ListBenchmarks)
			benchmarks.GET("/:id", h.GetBenchmark)
			benchmarks.DELETE("/:id", h.DeleteBenchmark)
			benchmarks.POST("/:id/run", h.RunBenchmark)
		}
	}
}

// registerMCPRoutes registers all MCP-related routes
func (s *Server) registerMCPRoutes(rg *gin.RouterGroup) {
	// Create MCP handler
	h := handlers.NewMCPHandler(s.store)

	mcp := rg.Group("/mcp")
	{
		servers := mcp.Group("/servers")
		{
			servers.POST("", h.Create)
			servers.GET("", h.List)
			servers.GET("/:id", h.Get)
			servers.PATCH("/:id", h.Update)
			servers.DELETE("/:id", h.Delete)
			servers.POST("/:id/connect", h.Connect)
			servers.POST("/:id/disconnect", h.Disconnect)
		}
	}
}

// registerSystemRoutes registers all system-related routes
func (s *Server) registerSystemRoutes(rg *gin.RouterGroup) {
	// Create system handler
	h := handlers.NewSystemHandler(s.store)

	system := rg.Group("/system")
	{
		// Configuration management
		config := system.Group("/config")
		{
			config.GET("", h.ListConfig)
			config.GET("/:key", h.GetConfig)
			config.PUT("/:key", h.UpdateConfig)
			config.DELETE("/:key", h.DeleteConfig)
		}

		// System operations
		system.GET("/info", h.GetInfo)
		system.GET("/health", h.GetHealth)
		system.GET("/stats", h.GetStats)
		system.POST("/reload", h.Reload)
		system.POST("/gc", h.RunGC)
		system.POST("/backup", h.Backup)
	}

	// Pool routes
	s.registerPoolRoutes(rg)

	// Room routes
	s.registerRoomRoutes(rg)
}

// registerPoolRoutes registers pool-related routes
func (s *Server) registerPoolRoutes(rg *gin.RouterGroup) {
	h := handlers.NewPoolHandler(s.store, s.deps.AgentDeps)

	pool := rg.Group("/pool")
	{
		pool.POST("/agents", h.CreateAgent)
		pool.GET("/agents", h.ListAgents)
		pool.GET("/agents/:id", h.GetAgent)
		pool.POST("/agents/:id/resume", h.ResumeAgent)
		pool.DELETE("/agents/:id", h.RemoveAgent)
		pool.GET("/stats", h.GetStats)
	}
}

// registerRoomRoutes registers room-related routes
func (s *Server) registerRoomRoutes(rg *gin.RouterGroup) {
	// Create a pool for rooms
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: s.deps.AgentDeps,
		MaxAgents:    100,
	})

	h := handlers.NewRoomHandler(s.store, pool)

	rooms := rg.Group("/rooms")
	{
		rooms.POST("", h.Create)
		rooms.GET("", h.List)
		rooms.GET("/:id", h.Get)
		rooms.DELETE("/:id", h.Delete)
		rooms.POST("/:id/join", h.Join)
		rooms.POST("/:id/leave", h.Leave)
		rooms.POST("/:id/say", h.Say)
		rooms.GET("/:id/members", h.GetMembers)
		rooms.GET("/:id/history", h.GetHistory)
	}
}

// registerA2ARoutes registers A2A protocol routes
func (s *Server) registerA2ARoutes(rg *gin.RouterGroup) {
	if s.a2aServer == nil {
		return // A2A not initialized
	}

	// 创建 A2A handler
	h := a2a.NewHandler(s.a2aServer)

	// 注册路由 (直接在 v1 group 上注册以获取正确的路径)
	h.RegisterRoutes(rg)
}

// registerDashboardRoutes registers all dashboard-related routes (with auth).
// Reserved for future use when dashboard authentication is enabled.
//
//nolint:unused
func (s *Server) registerDashboardRoutes(rg *gin.RouterGroup) {
	// Create dashboard handler with agent registry
	h := handlers.NewDashboardHandlerWithRegistry(s.agentRegistry, s.store)

	// Create dashboard event stream handler
	eventHandler := handlers.NewDashboardEventHandler(s.agentRegistry)

	dashboard := rg.Group("/dashboard")
	{
		// Overview
		dashboard.GET("/overview", h.GetOverview)

		// Traces
		dashboard.GET("/traces", h.ListTraces)
		dashboard.GET("/traces/:id", h.GetTrace)

		// Metrics
		metrics := dashboard.Group("/metrics")
		{
			metrics.GET("/tokens", h.GetTokenUsage)
			metrics.GET("/costs", h.GetCosts)
			metrics.GET("/performance", h.GetPerformance)
		}

		// Events
		events := dashboard.Group("/events")
		{
			events.GET("", h.GetRecentEvents)
			events.GET("/since/:cursor", h.GetEventsSince)
			// WebSocket endpoint for real-time event streaming
			events.GET("/stream", eventHandler.HandleEventStream)
			events.GET("/stats", eventHandler.GetStats)
		}

		// Insights
		dashboard.GET("/insights", h.GetInsights)

		// Pricing configuration
		pricing := dashboard.Group("/pricing")
		{
			pricing.GET("", h.GetPricing)
			pricing.PUT("", h.UpdatePricing)
		}

		// Sessions
		sessions := dashboard.Group("/sessions")
		{
			sessions.GET("", h.ListSessions)
			sessions.GET("/:id", h.GetSession)
		}
	}
}

// registerDashboardRoutesNoAuth registers dashboard routes without authentication (for Studio UI)
func (s *Server) registerDashboardRoutesNoAuth(dashboard *gin.RouterGroup) {
	// Create dashboard handler with agent registry
	h := handlers.NewDashboardHandlerWithRegistry(s.agentRegistry, s.store)

	// Create dashboard event stream handler
	eventHandler := handlers.NewDashboardEventHandler(s.agentRegistry)

	// Overview
	dashboard.GET("/overview", h.GetOverview)

	// Traces
	dashboard.GET("/traces", h.ListTraces)
	dashboard.GET("/traces/:id", h.GetTrace)

	// Metrics
	metrics := dashboard.Group("/metrics")
	{
		metrics.GET("/tokens", h.GetTokenUsage)
		metrics.GET("/costs", h.GetCosts)
		metrics.GET("/performance", h.GetPerformance)
	}

	// Events
	events := dashboard.Group("/events")
	{
		events.GET("", h.GetRecentEvents)
		events.GET("/since/:cursor", h.GetEventsSince)
		// WebSocket endpoint for real-time event streaming
		events.GET("/stream", eventHandler.HandleEventStream)
		events.GET("/stats", eventHandler.GetStats)
	}

	// Insights
	dashboard.GET("/insights", h.GetInsights)

	// Pricing configuration
	pricing := dashboard.Group("/pricing")
	{
		pricing.GET("", h.GetPricing)
		pricing.PUT("", h.UpdatePricing)
	}

	// Sessions
	sessions := dashboard.Group("/sessions")
	{
		sessions.GET("", h.ListSessions)
		sessions.GET("/:id", h.GetSession)
	}
}

// registerRemoteAgentRoutes registers remote agent routes
func (s *Server) registerRemoteAgentRoutes(rg *gin.RouterGroup) {
	// Create remote agent handler with store for persistence
	h := handlers.NewRemoteAgentHandler(s.agentRegistry, s.store)

	remoteAgents := rg.Group("/remote-agents")
	{
		remoteAgents.GET("/connect", h.HandleConnect)
		remoteAgents.GET("/stats", h.GetStats)
	}
}
