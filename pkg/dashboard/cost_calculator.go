package dashboard

import (
	"maps"
	"strings"
	"sync"
)

// CostCalculator 成本计算器
type CostCalculator struct {
	mu       sync.RWMutex
	pricing  map[string]ModelPricing
	currency string
}

// NewCostCalculator 创建成本计算器
func NewCostCalculator(customPricing map[string]ModelPricing) *CostCalculator {
	pricing := make(map[string]ModelPricing)

	// 复制默认定价
	maps.Copy(pricing, DefaultModelPricing)

	// 合并自定义定价
	maps.Copy(pricing, customPricing)

	return &CostCalculator{
		pricing:  pricing,
		currency: "USD",
	}
}

// Calculate 计算成本
func (cc *CostCalculator) Calculate(inputTokens, outputTokens int64, model string) CostAmount {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	pricing := cc.getPricing(model)

	// 计算成本 (价格是每百万 token)
	inputCost := float64(inputTokens) * pricing.InputPricePerM / 1_000_000
	outputCost := float64(outputTokens) * pricing.OutputPricePerM / 1_000_000

	return CostAmount{
		Amount:   inputCost + outputCost,
		Currency: cc.currency,
	}
}

// CalculateDetailed 计算详细成本
func (cc *CostCalculator) CalculateDetailed(inputTokens, outputTokens int64, model string) *DetailedCost {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	pricing := cc.getPricing(model)

	inputCost := float64(inputTokens) * pricing.InputPricePerM / 1_000_000
	outputCost := float64(outputTokens) * pricing.OutputPricePerM / 1_000_000

	return &DetailedCost{
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		InputCost: CostAmount{
			Amount:   inputCost,
			Currency: cc.currency,
		},
		OutputCost: CostAmount{
			Amount:   outputCost,
			Currency: cc.currency,
		},
		TotalCost: CostAmount{
			Amount:   inputCost + outputCost,
			Currency: cc.currency,
		},
		Pricing: pricing,
	}
}

// SetPricing 设置模型定价
func (cc *CostCalculator) SetPricing(model string, pricing ModelPricing) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.pricing[model] = pricing
}

// GetPricing 获取模型定价
func (cc *CostCalculator) GetPricing(model string) ModelPricing {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.getPricing(model)
}

// getPricing 内部获取定价方法（不加锁）
func (cc *CostCalculator) getPricing(model string) ModelPricing {
	// 精确匹配
	if pricing, ok := cc.pricing[model]; ok {
		return pricing
	}

	// 模糊匹配（处理模型名称变体）
	modelLower := strings.ToLower(model)

	// Claude 系列
	if strings.Contains(modelLower, "claude") {
		if strings.Contains(modelLower, "opus") {
			return ModelPricing{
				Model:           model,
				InputPricePerM:  15.0,
				OutputPricePerM: 75.0,
				Currency:        "USD",
			}
		}
		if strings.Contains(modelLower, "sonnet") {
			return ModelPricing{
				Model:           model,
				InputPricePerM:  3.0,
				OutputPricePerM: 15.0,
				Currency:        "USD",
			}
		}
		if strings.Contains(modelLower, "haiku") {
			return ModelPricing{
				Model:           model,
				InputPricePerM:  0.8,
				OutputPricePerM: 4.0,
				Currency:        "USD",
			}
		}
	}

	// GPT 系列
	if strings.Contains(modelLower, "gpt-4") {
		if strings.Contains(modelLower, "mini") {
			return ModelPricing{
				Model:           model,
				InputPricePerM:  0.15,
				OutputPricePerM: 0.6,
				Currency:        "USD",
			}
		}
		if strings.Contains(modelLower, "turbo") {
			return ModelPricing{
				Model:           model,
				InputPricePerM:  10.0,
				OutputPricePerM: 30.0,
				Currency:        "USD",
			}
		}
		// GPT-4o
		return ModelPricing{
			Model:           model,
			InputPricePerM:  2.5,
			OutputPricePerM: 10.0,
			Currency:        "USD",
		}
	}

	if strings.Contains(modelLower, "gpt-3.5") {
		return ModelPricing{
			Model:           model,
			InputPricePerM:  0.5,
			OutputPricePerM: 1.5,
			Currency:        "USD",
		}
	}

	// DeepSeek 系列
	if strings.Contains(modelLower, "deepseek") {
		if strings.Contains(modelLower, "coder") {
			return ModelPricing{
				Model:           model,
				InputPricePerM:  0.14,
				OutputPricePerM: 0.28,
				Currency:        "USD",
			}
		}
		return ModelPricing{
			Model:           model,
			InputPricePerM:  0.14,
			OutputPricePerM: 0.28,
			Currency:        "USD",
		}
	}

	// Gemini 系列
	if strings.Contains(modelLower, "gemini") {
		if strings.Contains(modelLower, "pro") {
			return ModelPricing{
				Model:           model,
				InputPricePerM:  1.25,
				OutputPricePerM: 5.0,
				Currency:        "USD",
			}
		}
		if strings.Contains(modelLower, "flash") {
			return ModelPricing{
				Model:           model,
				InputPricePerM:  0.075,
				OutputPricePerM: 0.3,
				Currency:        "USD",
			}
		}
	}

	// 默认定价（使用中等价格）
	return ModelPricing{
		Model:           model,
		InputPricePerM:  1.0,
		OutputPricePerM: 3.0,
		Currency:        "USD",
	}
}

// ListPricing 列出所有定价
func (cc *CostCalculator) ListPricing() map[string]ModelPricing {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	result := make(map[string]ModelPricing)
	maps.Copy(result, cc.pricing)

	return result
}

// SetCurrency 设置货币单位
func (cc *CostCalculator) SetCurrency(currency string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.currency = currency
}

// DetailedCost 详细成本
type DetailedCost struct {
	Model        string       `json:"model"`
	InputTokens  int64        `json:"input_tokens"`
	OutputTokens int64        `json:"output_tokens"`
	InputCost    CostAmount   `json:"input_cost"`
	OutputCost   CostAmount   `json:"output_cost"`
	TotalCost    CostAmount   `json:"total_cost"`
	Pricing      ModelPricing `json:"pricing"`
}

// EstimateCost 预估成本（用于预算规划）
func (cc *CostCalculator) EstimateCost(model string, estimatedInputTokens, estimatedOutputTokens int64) CostAmount {
	return cc.Calculate(estimatedInputTokens, estimatedOutputTokens, model)
}

// CalculateBatch 批量计算成本
func (cc *CostCalculator) CalculateBatch(usages []TokenUsageWithModel) CostAmount {
	var totalAmount float64

	for _, usage := range usages {
		cost := cc.Calculate(usage.InputTokens, usage.OutputTokens, usage.Model)
		totalAmount += cost.Amount
	}

	return CostAmount{
		Amount:   totalAmount,
		Currency: cc.currency,
	}
}

// TokenUsageWithModel Token 使用量（带模型信息）
type TokenUsageWithModel struct {
	Model        string `json:"model"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
}

// FormatCost 格式化成本显示
func FormatCost(cost CostAmount) string {
	switch cost.Currency {
	case "USD":
		if cost.Amount < 0.01 {
			return "$" + formatFloat(cost.Amount*100, 2) + "¢"
		}
		return "$" + formatFloat(cost.Amount, 4)
	case "CNY":
		return "¥" + formatFloat(cost.Amount, 2)
	default:
		return formatFloat(cost.Amount, 4) + " " + cost.Currency
	}
}

// formatFloat 格式化浮点数
func formatFloat(f float64, precision int) string {
	format := "%." + string(rune('0'+precision)) + "f"
	return strings.TrimRight(strings.TrimRight(
		sprintf(format, f), "0"), ".")
}

// sprintf 简化的格式化函数
func sprintf(format string, a float64) string {
	// 简单实现，实际使用 fmt.Sprintf
	switch format {
	case "%.2f":
		return floatToString(a, 2)
	case "%.4f":
		return floatToString(a, 4)
	default:
		return floatToString(a, 2)
	}
}

// floatToString 将浮点数转换为字符串
func floatToString(f float64, precision int) string {
	// 使用整数运算避免浮点精度问题
	multiplier := int64(1)
	for range precision {
		multiplier *= 10
	}

	intPart := int64(f)
	decPart := int64((f - float64(intPart)) * float64(multiplier))
	if decPart < 0 {
		decPart = -decPart
	}

	result := intToString(intPart) + "."

	// 补零
	decStr := intToString(decPart)
	for len(decStr) < precision {
		decStr = "0" + decStr
	}

	return result + decStr
}

// intToString 将整数转换为字符串
func intToString(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}
