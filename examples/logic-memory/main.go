// Package main demonstrates the Logic Memory framework for Aster SDK.
//
// Logic Memory enables AI agents to:
// - Learn user preferences from interactions
// - Remember behavioral patterns across sessions
// - Apply learned knowledge to personalize responses
//
// This example shows:
// 1. Basic setup with InMemoryStore
// 2. Recording and retrieving memories
// 3. Processing events with PatternMatcher
// 4. Middleware integration for automatic capture and injection
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/memory/logic"
	"github.com/astercloud/aster/pkg/middleware"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Aster SDK Logic Memory Example ===")
	fmt.Println()

	// Example 1: Basic Usage
	basicUsageExample(ctx)

	// Example 2: Event Processing with PatternMatcher
	eventProcessingExample(ctx)

	// Example 3: Middleware Integration
	middlewareExample(ctx)

	// Example 4: Memory Consolidation
	consolidationExample(ctx)

	// Example 5: Memory Pruning
	pruningExample(ctx)

	fmt.Println("\n=== All examples completed ===")
}

// basicUsageExample demonstrates basic Logic Memory operations
func basicUsageExample(ctx context.Context) {
	fmt.Println("--- Example 1: Basic Usage ---")

	// 1. Create a store (InMemory for this example)
	store := logic.NewInMemoryStore()

	// 2. Create the manager
	manager, err := logic.NewManager(&logic.ManagerConfig{
		Store:           store,
		ConfidenceBoost: 0.1, // Boost confidence by 10% on repeated patterns
	})
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Close() }()

	// 3. Record a memory manually
	mem := &logic.LogicMemory{
		Namespace:   "user:alice",
		Scope:       logic.ScopeUser,
		Type:        "preference",
		Key:         "writing_tone",
		Value:       "casual",
		Description: "Alice prefers a casual, friendly writing tone",
		Provenance: &memory.MemoryProvenance{
			SourceType: memory.SourceUserInput,
			Confidence: 0.8,
			Sources:    []string{"user_feedback:2024-01-15"},
		},
	}

	err = manager.RecordMemory(ctx, mem)
	if err != nil {
		log.Fatalf("Failed to record memory: %v", err)
	}
	fmt.Printf("Recorded memory: %s = %v (confidence: %.0f%%)\n",
		mem.Key, mem.Value, mem.Provenance.Confidence*100)

	// 4. Retrieve memories
	memories, err := manager.RetrieveMemories(ctx, "user:alice",
		logic.WithTopK(5),
		logic.WithMinConfidence(0.5),
	)
	if err != nil {
		log.Fatalf("Failed to retrieve memories: %v", err)
	}

	fmt.Printf("Retrieved %d memories for user:alice\n", len(memories))
	for _, m := range memories {
		fmt.Printf("  - %s: %s (%.0f%% confidence)\n",
			m.Key, m.Description, m.Provenance.Confidence*100)
	}

	// 5. Record the same memory again to boost confidence
	mem2 := &logic.LogicMemory{
		Namespace:   "user:alice",
		Scope:       logic.ScopeUser,
		Type:        "preference",
		Key:         "writing_tone",
		Value:       "casual",
		Description: "Alice prefers a casual, friendly writing tone",
		Provenance: &memory.MemoryProvenance{
			SourceType: memory.SourceUserInput,
			Confidence: 0.8,
		},
	}
	err = manager.RecordMemory(ctx, mem2)
	if err != nil {
		log.Fatalf("Failed to record memory: %v", err)
	}

	// Check boosted confidence
	updated, _ := manager.GetMemory(ctx, "user:alice", "writing_tone")
	fmt.Printf("After re-recording: confidence boosted to %.0f%%\n",
		updated.Provenance.Confidence*100)

	fmt.Println()
}

// eventProcessingExample demonstrates automatic memory capture from events
func eventProcessingExample(ctx context.Context) {
	fmt.Println("--- Example 2: Event Processing ---")

	store := logic.NewInMemoryStore()

	// Create a custom PatternMatcher
	matcher := &WritingPatternMatcher{}

	manager, err := logic.NewManager(&logic.ManagerConfig{
		Store:    store,
		Matchers: []logic.PatternMatcher{matcher},
	})
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Close() }()

	// Simulate a user revision event (user edited AI-generated content)
	event := logic.Event{
		Type:   "user_revision",
		Source: "user:bob",
		Data: map[string]any{
			"original": "However, this approach has several disadvantages.",
			"revised":  "But this approach has some problems.",
		},
		Timestamp: time.Now(),
	}

	// Process the event - PatternMatcher will identify patterns
	err = manager.ProcessEvent(ctx, event)
	if err != nil {
		log.Fatalf("Failed to process event: %v", err)
	}
	fmt.Println("Processed user_revision event")

	// Check if any memories were created
	memories, _ := manager.RetrieveMemories(ctx, "user:bob")
	fmt.Printf("Created %d memories from event\n", len(memories))
	for _, m := range memories {
		fmt.Printf("  - %s: %s (%.0f%% confidence)\n",
			m.Key, m.Description, m.Provenance.Confidence*100)
	}

	fmt.Println()
}

// WritingPatternMatcher is a custom PatternMatcher for detecting writing preferences
type WritingPatternMatcher struct{}

func (m *WritingPatternMatcher) SupportedEventTypes() []string {
	return []string{"user_revision", "user_feedback"}
}

func (m *WritingPatternMatcher) MatchEvent(ctx context.Context, event logic.Event) ([]*logic.LogicMemory, error) {
	if event.Type != "user_revision" {
		return nil, nil
	}

	original, _ := event.Data["original"].(string)
	revised, _ := event.Data["revised"].(string)

	var memories []*logic.LogicMemory

	// Simple pattern: formal to casual language
	// "However" -> "But" indicates preference for simpler language
	if containsWord(original, "However") && containsWord(revised, "But") {
		memories = append(memories, &logic.LogicMemory{
			Namespace:   event.Source,
			Scope:       logic.ScopeUser,
			Type:        "preference",
			Key:         "casual_language",
			Value:       true,
			Description: "Prefers casual language over formal expressions",
			Provenance: &memory.MemoryProvenance{
				SourceType: memory.SourceUserInput,
				Confidence: 0.7,
				Sources:    []string{fmt.Sprintf("revision:%d", time.Now().Unix())},
			},
		})
	}

	// Pattern: simplifying words
	if containsWord(original, "disadvantages") && containsWord(revised, "problems") {
		memories = append(memories, &logic.LogicMemory{
			Namespace:   event.Source,
			Scope:       logic.ScopeUser,
			Type:        "preference",
			Key:         "simple_vocabulary",
			Value:       true,
			Description: "Prefers simple, everyday vocabulary",
			Provenance: &memory.MemoryProvenance{
				SourceType: memory.SourceUserInput,
				Confidence: 0.65,
			},
		})
	}

	return memories, nil
}

func containsWord(text, word string) bool {
	return len(text) > 0 && len(word) > 0 &&
		(text == word || (len(text) >= len(word) &&
			(text[:len(word)] == word || text[len(text)-len(word):] == word ||
				contains(text, word))))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// middlewareExample demonstrates LogicMemoryMiddleware integration
func middlewareExample(ctx context.Context) {
	fmt.Println("--- Example 3: Middleware Integration ---")

	store := logic.NewInMemoryStore()

	// Pre-populate some memories
	_ = store.Save(ctx, &logic.LogicMemory{
		ID:          "mem-1",
		Namespace:   "user:charlie",
		Scope:       logic.ScopeUser,
		Type:        "preference",
		Key:         "response_format",
		Value:       "bullet_points",
		Description: "Charlie prefers bullet-point lists over paragraphs",
		Provenance: &memory.MemoryProvenance{
			SourceType: memory.SourceUserInput,
			Confidence: 0.85,
		},
	})

	manager, _ := logic.NewManager(&logic.ManagerConfig{Store: store})

	// Create the middleware
	mw, err := middleware.NewLogicMemoryMiddleware(&middleware.LogicMemoryMiddlewareConfig{
		Manager: manager,

		// Capture configuration
		EnableCapture: true,
		AsyncCapture:  true, // Non-blocking capture

		// Injection configuration
		EnableInjection: true,
		MaxMemories:     5,
		MinConfidence:   0.6,
		InjectionPoint:  "system_prompt_end",
	})
	if err != nil {
		log.Fatalf("Failed to create middleware: %v", err)
	}

	fmt.Printf("Middleware created: %s (priority: %d)\n", mw.Name(), mw.Priority())
	fmt.Printf("Configuration: %+v\n", mw.GetConfig())

	// Manually capture events
	mw.CaptureUserFeedback("user:charlie", "Great response!", 5, nil)
	mw.CaptureUserRevision("user:charlie",
		"The results are as follows:",
		"Here are the results:",
		nil,
	)

	fmt.Println("Events captured via middleware")

	// Get available tools
	tools := mw.Tools()
	fmt.Printf("Available tools: %d\n", len(tools))
	for _, t := range tools {
		fmt.Printf("  - %s: %s\n", t.Name(), t.Description())
	}

	fmt.Println()
}

// consolidationExample demonstrates memory consolidation
func consolidationExample(ctx context.Context) {
	fmt.Println("--- Example 4: Memory Consolidation ---")

	store := logic.NewInMemoryStore()

	// Create some similar memories that should be consolidated
	memories := []*logic.LogicMemory{
		{
			ID:          "1",
			Namespace:   "user:diana",
			Type:        "preference",
			Key:         "tone_casual_1",
			Description: "User prefers casual tone",
			Provenance:  &memory.MemoryProvenance{Confidence: 0.7},
		},
		{
			ID:          "2",
			Namespace:   "user:diana",
			Type:        "preference",
			Key:         "tone_casual_2",
			Description: "User likes informal language",
			Provenance:  &memory.MemoryProvenance{Confidence: 0.8},
		},
		{
			ID:          "3",
			Namespace:   "user:diana",
			Type:        "behavior",
			Key:         "format_preference",
			Description: "User prefers lists",
			Provenance:  &memory.MemoryProvenance{Confidence: 0.9},
		},
	}

	// Save memories to store
	for _, m := range memories {
		_ = store.Save(ctx, m)
	}

	fmt.Printf("Created %d memories\n", len(memories))

	// Create consolidation engine
	engine := logic.NewConsolidationEngine(store, &logic.ConsolidationConfig{
		SimilarityThreshold: 0.6,
		MinGroupSize:        2,
		MergeStrategy:       logic.MergeStrategyKeepHighestConfidence,
	})

	// Run consolidation
	result, err := engine.Consolidate(ctx, "user:diana")
	if err != nil {
		log.Fatalf("Failed to consolidate: %v", err)
	}

	fmt.Printf("Consolidation result:\n")
	fmt.Printf("  - Total memories: %d\n", result.TotalMemories)
	fmt.Printf("  - Groups merged: %d\n", result.MergedGroups)
	fmt.Printf("  - Memories deleted: %d\n", result.DeletedMemories)
	fmt.Printf("  - Duration: %v\n", result.EndTime.Sub(result.StartTime))

	fmt.Println()
}

// pruningExample demonstrates memory pruning
func pruningExample(ctx context.Context) {
	fmt.Println("--- Example 5: Memory Pruning ---")

	store := logic.NewInMemoryStore()
	manager, _ := logic.NewManager(&logic.ManagerConfig{Store: store})

	// Create memories with varying quality
	now := time.Now()
	memories := []*logic.LogicMemory{
		{
			Namespace:    "user:eve",
			Key:          "high_quality",
			Description:  "High confidence, recently accessed",
			Provenance:   &memory.MemoryProvenance{Confidence: 0.9},
			AccessCount:  10,
			LastAccessed: now,
			CreatedAt:    now.Add(-24 * time.Hour),
		},
		{
			Namespace:    "user:eve",
			Key:          "low_confidence",
			Description:  "Low confidence memory",
			Provenance:   &memory.MemoryProvenance{Confidence: 0.3},
			AccessCount:  1,
			LastAccessed: now.Add(-48 * time.Hour),
			CreatedAt:    now.Add(-72 * time.Hour),
		},
		{
			Namespace:    "user:eve",
			Key:          "old_unused",
			Description:  "Old and rarely accessed",
			Provenance:   &memory.MemoryProvenance{Confidence: 0.6},
			AccessCount:  0,
			LastAccessed: now.Add(-168 * time.Hour), // 7 days ago
			CreatedAt:    now.Add(-240 * time.Hour), // 10 days ago
		},
	}

	for _, m := range memories {
		_ = store.Save(ctx, m)
	}

	fmt.Printf("Created %d memories\n", len(memories))

	// Define pruning criteria
	criteria := logic.PruneCriteria{
		MinConfidence:   0.5,             // Remove if confidence < 50%
		SinceLastAccess: 72 * time.Hour,  // Remove if not accessed in 3 days
		MinAccessCount:  1,               // Minimum access count to keep
		MaxAge:          168 * time.Hour, // Maximum age: 7 days
	}

	// Execute pruning
	count, err := manager.PruneMemories(ctx, criteria)
	if err != nil {
		log.Fatalf("Failed to prune: %v", err)
	}

	fmt.Printf("Pruned %d memories\n", count)

	// Check remaining memories
	remaining, _ := manager.RetrieveMemories(ctx, "user:eve")
	fmt.Printf("Remaining memories: %d\n", len(remaining))
	for _, m := range remaining {
		fmt.Printf("  - %s (confidence: %.0f%%, accesses: %d)\n",
			m.Key, m.Provenance.Confidence*100, m.AccessCount)
	}

	fmt.Println()
}
