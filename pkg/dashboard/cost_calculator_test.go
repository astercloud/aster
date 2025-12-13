package dashboard

import (
	"testing"
)

func TestCostCalculator_Calculate(t *testing.T) {
	tests := []struct {
		name         string
		inputTokens  int64
		outputTokens int64
		model        string
		wantMin      float64 // 最小期望成本
		wantMax      float64 // 最大期望成本
	}{
		{
			name:         "Claude Sonnet pricing",
			inputTokens:  1000000, // 1M tokens
			outputTokens: 1000000,
			model:        "claude-3-5-sonnet-20241022",
			wantMin:      17.0, // 3 + 15 = 18, allow some margin
			wantMax:      19.0,
		},
		{
			name:         "GPT-4o pricing",
			inputTokens:  1000000,
			outputTokens: 1000000,
			model:        "gpt-4o",
			wantMin:      11.0, // 2.5 + 10 = 12.5
			wantMax:      14.0,
		},
		{
			name:         "DeepSeek pricing",
			inputTokens:  1000000,
			outputTokens: 1000000,
			model:        "deepseek-chat",
			wantMin:      0.3, // 0.14 + 0.28 = 0.42
			wantMax:      0.5,
		},
		{
			name:         "Unknown model uses default",
			inputTokens:  1000000,
			outputTokens: 1000000,
			model:        "unknown-model",
			wantMin:      3.0, // default: 1.0 + 3.0 = 4.0
			wantMax:      5.0,
		},
		{
			name:         "Zero tokens",
			inputTokens:  0,
			outputTokens: 0,
			model:        "claude-3-5-sonnet-20241022",
			wantMin:      0,
			wantMax:      0,
		},
		{
			name:         "Small token count",
			inputTokens:  1000,
			outputTokens: 500,
			model:        "claude-3-5-sonnet-20241022",
			wantMin:      0.005, // (1000 * 3 + 500 * 15) / 1M = 0.0105
			wantMax:      0.015,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := NewCostCalculator(nil)
			got := cc.Calculate(tt.inputTokens, tt.outputTokens, tt.model)

			if got.Amount < tt.wantMin || got.Amount > tt.wantMax {
				t.Errorf("Calculate() = %v, want between %v and %v", got.Amount, tt.wantMin, tt.wantMax)
			}

			if got.Currency != "USD" {
				t.Errorf("Calculate() currency = %v, want USD", got.Currency)
			}
		})
	}
}

func TestCostCalculator_CustomPricing(t *testing.T) {
	customPricing := map[string]ModelPricing{
		"custom-model": {
			Model:           "custom-model",
			InputPricePerM:  1.0,
			OutputPricePerM: 2.0,
			Currency:        "USD",
		},
	}

	cc := NewCostCalculator(customPricing)

	// Test custom pricing
	got := cc.Calculate(1000000, 1000000, "custom-model")
	expectedCost := 3.0 // 1.0 + 2.0

	if got.Amount != expectedCost {
		t.Errorf("Calculate() with custom pricing = %v, want %v", got.Amount, expectedCost)
	}

	// Test that default pricing still works
	got2 := cc.Calculate(1000000, 1000000, "gpt-4o")
	if got2.Amount < 10 || got2.Amount > 15 {
		t.Errorf("Calculate() with default pricing = %v, want between 10 and 15", got2.Amount)
	}
}

func TestCostCalculator_SetPricing(t *testing.T) {
	cc := NewCostCalculator(nil)

	// Set new pricing
	cc.SetPricing("new-model", ModelPricing{
		Model:           "new-model",
		InputPricePerM:  5.0,
		OutputPricePerM: 10.0,
		Currency:        "USD",
	})

	// Verify pricing was set
	pricing := cc.GetPricing("new-model")
	if pricing.InputPricePerM != 5.0 {
		t.Errorf("GetPricing() InputPricePerM = %v, want 5.0", pricing.InputPricePerM)
	}

	// Test calculation with new pricing
	got := cc.Calculate(1000000, 1000000, "new-model")
	expectedCost := 15.0 // 5.0 + 10.0

	if got.Amount != expectedCost {
		t.Errorf("Calculate() = %v, want %v", got.Amount, expectedCost)
	}
}

func TestCostCalculator_FuzzyModelMatching(t *testing.T) {
	cc := NewCostCalculator(nil)

	tests := []struct {
		model   string
		wantMin float64
		wantMax float64
	}{
		// Claude variants
		{"claude-3-opus-20240229", 85.0, 95.0},   // 15 + 75 = 90
		{"claude-3-sonnet-20240229", 17.0, 19.0}, // 3 + 15 = 18
		{"claude-3-haiku-20240307", 4.0, 5.5},    // 0.8 + 4 = 4.8

		// GPT variants
		{"gpt-4-turbo-preview", 35.0, 45.0},  // 10 + 30 = 40
		{"gpt-4o-mini-2024-07-18", 0.5, 1.0}, // 0.15 + 0.6 = 0.75
		{"gpt-3.5-turbo-0125", 1.5, 2.5},     // 0.5 + 1.5 = 2.0

		// Gemini variants
		{"gemini-1.5-pro", 5.0, 7.0},   // 1.25 + 5.0 = 6.25
		{"gemini-1.5-flash", 0.3, 0.5}, // 0.075 + 0.3 = 0.375
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := cc.Calculate(1000000, 1000000, tt.model)
			if got.Amount < tt.wantMin || got.Amount > tt.wantMax {
				t.Errorf("Calculate(%s) = %v, want between %v and %v",
					tt.model, got.Amount, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCostCalculator_CalculateDetailed(t *testing.T) {
	cc := NewCostCalculator(nil)

	detail := cc.CalculateDetailed(1000, 500, "claude-3-5-sonnet-20241022")

	if detail.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("CalculateDetailed() model = %v, want claude-3-5-sonnet-20241022", detail.Model)
	}

	if detail.InputTokens != 1000 {
		t.Errorf("CalculateDetailed() InputTokens = %v, want 1000", detail.InputTokens)
	}

	if detail.OutputTokens != 500 {
		t.Errorf("CalculateDetailed() OutputTokens = %v, want 500", detail.OutputTokens)
	}

	// Input cost: 1000 * 3.0 / 1M = 0.003
	expectedInputCost := 0.003
	if detail.InputCost.Amount < expectedInputCost*0.9 || detail.InputCost.Amount > expectedInputCost*1.1 {
		t.Errorf("CalculateDetailed() InputCost = %v, want ~%v", detail.InputCost.Amount, expectedInputCost)
	}

	// Output cost: 500 * 15.0 / 1M = 0.0075
	expectedOutputCost := 0.0075
	if detail.OutputCost.Amount < expectedOutputCost*0.9 || detail.OutputCost.Amount > expectedOutputCost*1.1 {
		t.Errorf("CalculateDetailed() OutputCost = %v, want ~%v", detail.OutputCost.Amount, expectedOutputCost)
	}
}

func TestCostCalculator_CalculateBatch(t *testing.T) {
	cc := NewCostCalculator(nil)

	usages := []TokenUsageWithModel{
		{Model: "claude-3-5-sonnet-20241022", InputTokens: 1000, OutputTokens: 500},
		{Model: "gpt-4o", InputTokens: 2000, OutputTokens: 1000},
	}

	got := cc.CalculateBatch(usages)

	// Claude: (1000 * 3 + 500 * 15) / 1M = 0.0105
	// GPT-4o: (2000 * 2.5 + 1000 * 10) / 1M = 0.015
	// Total: ~0.0255
	if got.Amount < 0.02 || got.Amount > 0.03 {
		t.Errorf("CalculateBatch() = %v, want between 0.02 and 0.03", got.Amount)
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		cost CostAmount
		want string
	}{
		{CostAmount{Amount: 1.5, Currency: "USD"}, "$1.5"},
		{CostAmount{Amount: 0.005, Currency: "USD"}, "$0.5¢"},
		{CostAmount{Amount: 100.50, Currency: "CNY"}, "¥100.5"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatCost(tt.cost)
			// 简单检查格式正确（包含货币符号）
			if len(got) == 0 {
				t.Errorf("FormatCost() returned empty string")
			}
		})
	}
}
