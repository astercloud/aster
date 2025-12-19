package culture

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/security"
)

// Culture 文化系统
type Culture struct {
	// 基本信息
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	// 文化维度
	Dimensions []CultureDimension `json:"dimensions"`

	// 规范和价值观
	Norms   []Norm   `json:"norms"`
	Values  []Value  `json:"values"`
	Beliefs []Belief `json:"beliefs"`

	// 行为模式
	Behaviors []BehaviorPattern `json:"behaviors"`
	Rituals   []Ritual          `json:"rituals"`

	// 沟通风格
	CommunicationStyles []CommunicationStyle `json:"communication_styles"`

	// 决策模式
	DecisionStyles []DecisionStyle `json:"decision_styles"`

	// 冲突解决
	ConflictResolution []ConflictResolutionStrategy `json:"conflict_resolution"`

	// 学习和适应
	LearningStyles       []LearningStyle      `json:"learning_styles"`
	AdaptationStrategies []AdaptationStrategy `json:"adaptation_strategies"`

	// 元数据
	Metadata map[string]any `json:"metadata"`
	Tags     []string       `json:"tags"`

	// 状态信息
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
	Active    bool      `json:"active"`

	// 上下文
	Context *CultureContext `json:"context,omitempty"`

	mu sync.RWMutex
}

// CultureDimension 文化维度
type CultureDimension struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        DimensionType  `json:"type"`
	Description string         `json:"description"`
	Value       float64        `json:"value"` // 0.0 - 1.0
	MinValue    float64        `json:"min_value"`
	MaxValue    float64        `json:"max_value"`
	Category    string         `json:"category"`
	Attributes  map[string]any `json:"attributes"`
	Indicators  []Indicator    `json:"indicators"` // 指标
}

// DimensionType 维度类型
type DimensionType string

const (
	DimensionTypePowerDistance        DimensionType = "power_distance"        // 权力距离
	DimensionTypeIndividualism        DimensionType = "individualism"         // 个人主义
	DimensionTypeUncertaintyAvoidance DimensionType = "uncertainty_avoidance" // 不确定性规避
	DimensionTypeLongTermOrientation  DimensionType = "long_term_orientation" // 长期导向
	DimensionTypeMasculinity          DimensionType = "masculinity"           // 男性化
	DimensionTypeContext              DimensionType = "context"               // 上下文
	DimensionTypeTimeOrientation      DimensionType = "time_orientation"      // 时间导向
	DimensionTypeRiskTolerance        DimensionType = "risk_tolerance"        // 风险容忍度
	DimensionTypeFormality            DimensionType = "formality"             // 正式性
	DimensionTypeHierarchy            DimensionType = "hierarchy"             // 层级结构
	DimensionTypeCollaboration        DimensionType = "collaboration"         // 协作倾向
	DimensionTypeInnovation           DimensionType = "innovation"            // 创新倾向
)

// Indicator 指标
type Indicator struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
}

// Norm 规范
type Norm struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        NormType        `json:"type"`
	Strength    float64         `json:"strength"` // 0.0 - 1.0
	Scope       NormScope       `json:"scope"`
	Conditions  []NormCondition `json:"conditions"`
	Exceptions  []NormException `json:"exceptions"`
	Attributes  map[string]any  `json:"attributes"`
}

// NormType 规范类型
type NormType string

const (
	NormTypeExplicit   NormType = "explicit"   // 显性规范
	NormTypeImplicit   NormType = "implicit"   // 隐性规范
	NormTypeFormal     NormType = "formal"     // 正式规范
	NormTypeInformal   NormType = "informal"   // 非正式规范
	NormTypeProcedural NormType = "procedural" // 程序性规范
)

// NormScope 规范范围
type NormScope string

const (
	NormScopeGlobal       NormScope = "global"       // 全局
	NormScopeTeam         NormScope = "team"         // 团队
	NormScopeOrganization NormScope = "organization" // 组织
	NormScopeCulture      NormScope = "culture"      // 文化
)

// NormCondition 规范条件
type NormCondition struct {
	Type        string `json:"type"`
	Field       string `json:"field"`
	Operator    string `json:"operator"`
	Value       any    `json:"value"`
	Description string `json:"description"`
}

// NormException 规范例外
type NormException struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	Conditions  []NormCondition `json:"conditions"`
	Attributes  map[string]any  `json:"attributes"`
}

// Value 价值观
type Value struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	Category       ValueCategory        `json:"category"`
	Priority       int                  `json:"priority"`   // 1-10
	Importance     float64              `json:"importance"` // 0.0 - 1.0
	Manifestations []ValueManifestation `json:"manifestations"`
	Attributes     map[string]any       `json:"attributes"`
}

// ValueCategory 价值观类别
type ValueCategory string

const (
	ValueCategoryMoral        ValueCategory = "moral"        // 道德价值观
	ValueCategoryAesthetic    ValueCategory = "aesthetic"    // 审美价值观
	ValueCategorySocial       ValueCategory = "social"       // 社会价值观
	ValueCategoryEconomic     ValueCategory = "economic"     // 经济价值观
	ValueCategoryPolitical    ValueCategory = "political"    // 政治价值观
	ValueCategoryReligious    ValueCategory = "religious"    // 宗教价值观
	ValueCategoryPersonal     ValueCategory = "personal"     // 个人价值观
	ValueCategoryProfessional ValueCategory = "professional" // 职业价值观
)

// ValueManifestation 价值观体现
type ValueManifestation struct {
	Context    string `json:"context"`
	Behavior   string `json:"behavior"`
	Expression string `json:"expression"`
	Example    string `json:"example"`
}

// Belief 信念
type Belief struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Type        BeliefType       `json:"type"`
	Category    BeliefCategory   `json:"category"`
	Strength    float64          `json:"strength"`
	Evidence    []BeliefEvidence `json:"evidence"`
	Source      string           `json:"source"`
	Attributes  map[string]any   `json:"attributes"`
}

// BeliefType 信念类型
type BeliefType string

const (
	BeliefTypeCore         BeliefType = "core"         // 核心信念
	BeliefTypePeripheral   BeliefType = "peripheral"   // 边缘信念
	BeliefTypeInstrumental BeliefType = "instrumental" // 工具性信念
	BeliefTypeTerminal     BeliefType = "terminal"     // 终端信念
)

// BeliefCategory 信念类别
type BeliefCategory string

const (
	BeliefCategorySelf         BeliefCategory = "self"         // 自我信念
	BeliefCategoryOthers       BeliefCategory = "others"       // 他人信念
	BeliefCategoryWorld        BeliefCategory = "world"        // 世界信念
	BeliefCategoryFuture       BeliefCategory = "future"       // 未来信念
	BeliefCategoryWork         BeliefCategory = "work"         // 工作信念
	BeliefCategoryRelationship BeliefCategory = "relationship" // 关系信念
)

// BeliefEvidence 信念证据
type BeliefEvidence struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Source      string  `json:"source"`
	Strength    float64 `json:"strength"`
	Attributes  any     `json:"attributes"`
}

// BehaviorPattern 行为模式
type BehaviorPattern struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        BehaviorType      `json:"type"`
	Frequency   FrequencyLevel    `json:"frequency"`
	Triggers    []BehaviorTrigger `json:"triggers"`
	Outcomes    []BehaviorOutcome `json:"outcomes"`
	Attributes  map[string]any    `json:"attributes"`
}

// BehaviorType 行为类型
type BehaviorType string

const (
	BehaviorTypeCommunication  BehaviorType = "communication"   // 沟通行为
	BehaviorTypeDecision       BehaviorType = "decision"        // 决策行为
	BehaviorTypeProblemSolving BehaviorType = "problem_solving" // 问题解决行为
	BehaviorTypeConflict       BehaviorType = "conflict"        // 冲突行为
	BehaviorTypeLearning       BehaviorType = "learning"        // 学习行为
	BehaviorTypeLeadership     BehaviorType = "leadership"      // 领导行为
	BehaviorTypeTeamwork       BehaviorType = "teamwork"        // 团队行为
	BehaviorTypeInnovation     BehaviorType = "innovation"      // 创新行为
)

// FrequencyLevel 频率级别
type FrequencyLevel string

const (
	FrequencyLevelAlways    FrequencyLevel = "always"    // 总是
	FrequencyLevelUsually   FrequencyLevel = "usually"   // 通常
	FrequencyLevelSometimes FrequencyLevel = "sometimes" // 有时
	FrequencyLevelRarely    FrequencyLevel = "rarely"    // 很少
	FrequencyLevelNever     FrequencyLevel = "never"     // 从不
)

// BehaviorTrigger 行为触发器
type BehaviorTrigger struct {
	Type        string `json:"type"`
	Condition   string `json:"condition"`
	Description string `json:"description"`
	Attributes  any    `json:"attributes"`
}

// BehaviorOutcome 行为结果
type BehaviorOutcome struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Probability float64 `json:"probability"`
	Impact      string  `json:"impact"`
}

// Ritual 仪式
type Ritual struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        RitualType     `json:"type"`
	Purpose     RitualPurpose  `json:"purpose"`
	Frequency   FrequencyLevel `json:"frequency"`
	Steps       []RitualStep   `json:"steps"`
	Symbols     []RitualSymbol `json:"symbols"`
	Attributes  map[string]any `json:"attributes"`
}

// RitualType 仪式类型
type RitualType string

const (
	RitualTypeWelcome     RitualType = "welcome"     // 欢迎仪式
	RitualTypeDeparture   RitualType = "departure"   // 告别仪式
	RitualTypeCelebration RitualType = "celebration" // 庆祝仪式
	RitualTypeTransition  RitualType = "transition"  // 过渡仪式
	RitualTypeRecognition RitualType = "recognition" // 认可仪式
	RitualTypeResolution  RitualType = "resolution"  // 解决仪式
	RitualTypeLearning    RitualType = "learning"    // 学习仪式
	RitualTypeTeam        RitualType = "team"        // 团队仪式
)

// RitualPurpose 仪式目的
type RitualPurpose string

const (
	RitualPurposeSocial        RitualPurpose = "social"        // 社交
	RitualPurposeEmotional     RitualPurpose = "emotional"     // 情感
	RitualPurposePsychological RitualPurpose = "psychological" // 心理
	RitualPurposeSpiritual     RitualPurpose = "spiritual"     // 精神
	RitualPurposeProfessional  RitualPurpose = "professional"  // 职业
	RitualPurposeCultural      RitualPurpose = "cultural"      // 文化
)

// RitualStep 仪式步骤
type RitualStep struct {
	Order       int    `json:"order"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Symbol      string `json:"symbol,omitempty"`
}

// RitualSymbol 仪式符号
type RitualSymbol struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Meaning     string `json:"meaning"`
	Type        string `json:"type"`
}

// CommunicationStyle 沟通风格
type CommunicationStyle struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        CommStyleType   `json:"type"`
	Directness  DirectnessLevel `json:"directness"`
	Context     ContextLevel    `json:"context"`
	Formality   FormalityLevel  `json:"formality"`
	Emotional   EmotionalLevel  `json:"emotional"`
	Attributes  map[string]any  `json:"attributes"`
}

// CommStyleType 沟通风格类型
type CommStyleType string

const (
	CommStyleTypeAssertive  CommStyleType = "assertive"  // 断言型
	CommStyleTypePassive    CommStyleType = "passive"    // 被动型
	CommStyleTypeAggressive CommStyleType = "aggressive" // 主动型
	CommStyleTypeAnalytical CommStyleType = "analytical" // 分析型
	CommStyleTypeExpressive CommStyleType = "expressive" // 表达型
	CommStyleTypeDriver     CommStyleType = "driver"     // 驱动型
	CommStyleTypeAmiable    CommStyleType = "amiable"    // 友好型
)

// DirectnessLevel 直接性级别
type DirectnessLevel string

const (
	DirectnessLevelVeryDirect   DirectnessLevel = "very_direct"   // 非常直接
	DirectnessLevelDirect       DirectnessLevel = "direct"        // 直接
	DirectnessLevelModerate     DirectnessLevel = "moderate"      // 中等
	DirectnessLevelIndirect     DirectnessLevel = "indirect"      // 间接
	DirectnessLevelVeryIndirect DirectnessLevel = "very_indirect" // 非常间接
)

// ContextLevel 上下文级别
type ContextLevel string

const (
	ContextLevelHigh   ContextLevel = "high"   // 高上下文
	ContextLevelMedium ContextLevel = "medium" // 中等上下文
	ContextLevelLow    ContextLevel = "low"    // 低上下文
)

// FormalityLevel 正式性级别
type FormalityLevel string

const (
	FormalityLevelVeryFormal FormalityLevel = "very_formal" // 非常正式
	FormalityLevelFormal     FormalityLevel = "formal"      // 正式
	FormalityLevelSemiFormal FormalityLevel = "semi_formal" // 半正式
	FormalityLevelInformal   FormalityLevel = "informal"    // 非正式
	FormalityLevelCasual     FormalityLevel = "casual"      // 随意
)

// EmotionalLevel 情感级别
type EmotionalLevel string

const (
	EmotionalLevelVeryHigh EmotionalLevel = "very_high" // 非常高情感
	EmotionalLevelHigh     EmotionalLevel = "high"      // 高情感
	EmotionalLevelModerate EmotionalLevel = "moderate"  // 中等情感
	EmotionalLevelLow      EmotionalLevel = "low"       // 低情感
	EmotionalLevelVeryLow  EmotionalLevel = "very_low"  // 非常低情感
)

// DecisionStyle 决策风格
type DecisionStyle struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Type           DecisionType       `json:"type"`
	Approach       DecisionApproach   `json:"approach"`
	Speed          DecisionSpeed      `json:"speed"`
	ConsensusLevel ConsensusLevel     `json:"consensus_level"`
	RiskTolerance  RiskToleranceLevel `json:"risk_tolerance"`
	DataPreference DataPreference     `json:"data_preference"`
	Attributes     map[string]any     `json:"attributes"`
}

// DecisionType 决策类型
type DecisionType string

const (
	DecisionTypeRational      DecisionType = "rational"      // 理性决策
	DecisionTypeIntuitive     DecisionType = "intuitive"     // 直觉决策
	DecisionTypeAnalytical    DecisionType = "analytical"    // 分析决策
	DecisionTypeCreative      DecisionType = "creative"      // 创造性决策
	DecisionTypeCollaborative DecisionType = "collaborative" // 协作决策
	DecisionTypeAuthoritative DecisionType = "authoritative" // 权威决策
	DecisionTypeDemocratic    DecisionType = "democratic"    // 民主决策
	DecisionTypeConsensus     DecisionType = "consensus"     // 共识决策
)

// DecisionApproach 决策方法
type DecisionApproach string

const (
	DecisionApproachSystematic DecisionApproach = "systematic"  // 系统性
	DecisionApproachHeuristic  DecisionApproach = "heuristic"   // 启发式
	DecisionApproachDataDriven DecisionApproach = "data_driven" // 数据驱动
	DecisionApproachExperience DecisionApproach = "experience"  // 经验驱动
	DecisionApproachBalanced   DecisionApproach = "balanced"    // 平衡型
)

// DecisionSpeed 决策速度
type DecisionSpeed string

const (
	DecisionSpeedImmediate  DecisionSpeed = "immediate"  // 立即
	DecisionSpeedFast       DecisionSpeed = "fast"       // 快速
	DecisionSpeedModerate   DecisionSpeed = "moderate"   // 中等
	DecisionSpeedDeliberate DecisionSpeed = "deliberate" // 深思熟虑
	DecisionSpeedExtended   DecisionSpeed = "extended"   // 延长
)

// ConsensusLevel 共识级别
type ConsensusLevel string

const (
	ConsensusLevelFull      ConsensusLevel = "full"      // 完全共识
	ConsensusLevelMajority  ConsensusLevel = "majority"  // 多数共识
	ConsensusLevelPlurality ConsensusLevel = "plurality" // 相对多数
	ConsensusLevelExpert    ConsensusLevel = "expert"    // 专家决定
	ConsensusLevelLeader    ConsensusLevel = "leader"    // 领导决定
)

// RiskToleranceLevel 风险容忍度级别
type RiskToleranceLevel string

const (
	RiskToleranceLevelVeryLow  RiskToleranceLevel = "very_low"  // 非常低
	RiskToleranceLevelLow      RiskToleranceLevel = "low"       // 低
	RiskToleranceLevelModerate RiskToleranceLevel = "moderate"  // 中等
	RiskToleranceLevelHigh     RiskToleranceLevel = "high"      // 高
	RiskToleranceLevelVeryHigh RiskToleranceLevel = "very_high" // 非常高
)

// DataPreference 数据偏好
type DataPreference string

const (
	DataPreferenceQuantitative  DataPreference = "quantitative"  // 定量数据
	DataPreferenceQualitative   DataPreference = "qualitative"   // 定性数据
	DataPreferenceMixed         DataPreference = "mixed"         // 混合数据
	DataPreferenceMinimal       DataPreference = "minimal"       // 最小数据
	DataPreferenceComprehensive DataPreference = "comprehensive" // 全面数据
)

// ConflictResolutionStrategy 冲突解决策略
type ConflictResolutionStrategy struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Type        ConflictType        `json:"type"`
	Style       ConflictStyle       `json:"style"`
	Approach    ConflictApproach    `json:"approach"`
	Timing      ConflictTiming      `json:"timing"`
	Involvement InvolvementLevel    `json:"involvement"`
	Outcome     ConflictOutcome     `json:"outcome"`
	Techniques  []ConflictTechnique `json:"techniques"`
	Attributes  map[string]any      `json:"attributes"`
}

// ConflictType 冲突类型
type ConflictType string

const (
	ConflictTypeTask         ConflictType = "task"         // 任务冲突
	ConflictTypeProcess      ConflictType = "process"      // 过程冲突
	ConflictTypeRelationship ConflictType = "relationship" // 关系冲突
	ConflictTypeStatus       ConflictType = "status"       // 状态冲突
	ConflictTypeValue        ConflictType = "value"        // 价值观冲突
	ConflictTypeInterest     ConflictType = "interest"     // 利益冲突
)

// ConflictStyle 冲突风格
type ConflictStyle string

const (
	ConflictStyleCompeting     ConflictStyle = "competing"     // 竞争型
	ConflictStyleCollaborating ConflictStyle = "collaborating" // 协作型
	ConflictStyleCompromising  ConflictStyle = "compromising"  // 妥协型
	ConflictStyleAvoiding      ConflictStyle = "avoiding"      // 回避型
	ConflictStyleAccommodating ConflictStyle = "accommodating" // 迁就型
)

// ConflictApproach 冲突方法
type ConflictApproach string

const (
	ConflictApproachIntegrative  ConflictApproach = "integrative"  // 整合型
	ConflictApproachDistributive ConflictApproach = "distributive" // 分配型
	ConflictApproachProcedural   ConflictApproach = "procedural"   // 程序型
	ConflictApproachEmotional    ConflictApproach = "emotional"    // 情感型
)

// ConflictTiming 冲突时机
type ConflictTiming string

const (
	ConflictTimingImmediate  ConflictTiming = "immediate"  // 立即
	ConflictTimingEarly      ConflictTiming = "early"      // 早期
	ConflictTimingDeliberate ConflictTiming = "deliberate" // 深思熟虑
	ConflictTimingDelayed    ConflictTiming = "delayed"    // 延迟
)

// InvolvementLevel 参与级别
type InvolvementLevel string

const (
	InvolvementLevelHigh     InvolvementLevel = "high"     // 高参与
	InvolvementLevelMedium   InvolvementLevel = "medium"   // 中等参与
	InvolvementLevelLow      InvolvementLevel = "low"      // 低参与
	InvolvementLevelDelegate InvolvementLevel = "delegate" // 委派
)

// ConflictOutcome 冲突结果
type ConflictOutcome string

const (
	ConflictOutcomeWinWin     ConflictOutcome = "win_win"    // 双赢
	ConflictOutcomeWinLose    ConflictOutcome = "win_lose"   // 一方赢
	ConflictOutcomeLoseLose   ConflictOutcome = "lose_lose"  // 双输
	ConflictOutcomeCompromise ConflictOutcome = "compromise" // 妥协
	ConflictOutcomeAvoidance  ConflictOutcome = "avoidance"  // 回避
)

// ConflictTechnique 冲突技巧
type ConflictTechnique struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Steps         []string `json:"steps"`
	Effectiveness string   `json:"effectiveness"`
}

// LearningStyle 学习风格
type LearningStyle struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Type        LearningType     `json:"type"`
	Modality    LearningModality `json:"modality"`
	Pace        LearningPace     `json:"pace"`
	Depth       LearningDepth    `json:"depth"`
	Social      LearningSocial   `json:"social"`
	Attributes  map[string]any   `json:"attributes"`
}

// LearningType 学习类型
type LearningType string

const (
	LearningTypeVisual      LearningType = "visual"      // 视觉型
	LearningTypeAuditory    LearningType = "auditory"    // 听觉型
	LearningTypeKinesthetic LearningType = "kinesthetic" // 动觉型
	LearningTypeReading     LearningType = "reading"     // 阅读型
	LearningTypeMixed       LearningType = "mixed"       // 混合型
)

// LearningModality 学习模态
type LearningModality string

const (
	LearningModalityTheoretical  LearningModality = "theoretical"  // 理论型
	LearningModalityPractical    LearningModality = "practical"    // 实践型
	LearningModalityConceptual   LearningModality = "conceptual"   // 概念型
	LearningModalityExperiential LearningModality = "experiential" // 经验型
)

// LearningPace 学习节奏
type LearningPace string

const (
	LearningPaceFast     LearningPace = "fast"     // 快节奏
	LearningPaceModerate LearningPace = "moderate" // 中等节奏
	LearningPaceSteady   LearningPace = "steady"   // 稳定节奏
	LearningPaceFlexible LearningPace = "flexible" // 灵活节奏
)

// LearningDepth 学习深度
type LearningDepth string

const (
	LearningDepthSurface   LearningDepth = "surface"   // 表面学习
	LearningDepthStrategic LearningDepth = "strategic" // 策略学习
	LearningDepthDeep      LearningDepth = "deep"      // 深度学习
)

// LearningSocial 学习社交性
type LearningSocial string

const (
	LearningSocialIndividual LearningSocial = "individual" // 个人学习
	LearningSocialPair       LearningSocial = "pair"       // 结对学习
	LearningSocialGroup      LearningSocial = "group"      // 小组学习
	LearningSocialTeam       LearningSocial = "team"       // 团队学习
)

// AdaptationStrategy 适应策略
type AdaptationStrategy struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Type        AdaptationType     `json:"type"`
	Scope       AdaptationScope    `json:"scope"`
	Trigger     AdaptationTrigger  `json:"trigger"`
	Response    AdaptationResponse `json:"response"`
	Flexibility FlexibilityLevel   `json:"flexibility"`
	Attributes  map[string]any     `json:"attributes"`
}

// AdaptationType 适应类型
type AdaptationType string

const (
	AdaptationTypeCognitive  AdaptationType = "cognitive"  // 认知适应
	AdaptationTypeBehavioral AdaptationType = "behavioral" // 行为适应
	AdaptationTypeEmotional  AdaptationType = "emotional"  // 情感适应
	AdaptationTypeStructural AdaptationType = "structural" // 结构适应
	AdaptationTypeProcedural AdaptationType = "procedural" // 程序适应
)

// AdaptationScope 适应范围
type AdaptationScope string

const (
	AdaptationScopeIndividual     AdaptationScope = "individual"     // 个人适应
	AdaptationScopeTeam           AdaptationScope = "team"           // 团队适应
	AdaptationScopeOrganizational AdaptationScope = "organizational" // 组织适应
	AdaptationScopeCultural       AdaptationScope = "cultural"       // 文化适应
)

// AdaptationTrigger 适应触发器
type AdaptationTrigger struct {
	Type        string `json:"type"`
	Condition   string `json:"condition"`
	Description string `json:"description"`
	Attributes  any    `json:"attributes"`
}

// AdaptationResponse 适应响应
type AdaptationResponse struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Strategy    string `json:"strategy"`
	Attributes  any    `json:"attributes"`
}

// FlexibilityLevel 灵活性级别
type FlexibilityLevel string

const (
	FlexibilityLevelRigid      FlexibilityLevel = "rigid"      // 僵化
	FlexibilityLevelStructured FlexibilityLevel = "structured" // 结构化
	FlexibilityLevelFlexible   FlexibilityLevel = "flexible"   // 灵活
	FlexibilityLevelAdaptive   FlexibilityLevel = "adaptive"   // 自适应
	FlexibilityLevelDynamic    FlexibilityLevel = "dynamic"    // 动态
)

// CultureContext 文化上下文
type CultureContext struct {
	Environment  string               `json:"environment"`
	Domain       string               `json:"domain"`
	Purpose      string               `json:"purpose"`
	Constraints  []ContextConstraint  `json:"constraints"`
	Resources    []ContextResource    `json:"resources"`
	Stakeholders []ContextStakeholder `json:"stakeholders"`
	Attributes   map[string]any       `json:"attributes"`
}

// Clone 克隆 CultureContext
func (cc *CultureContext) Clone() *CultureContext {
	if cc == nil {
		return nil
	}

	clone := &CultureContext{
		Environment:  cc.Environment,
		Domain:       cc.Domain,
		Purpose:      cc.Purpose,
		Constraints:  make([]ContextConstraint, len(cc.Constraints)),
		Resources:    make([]ContextResource, len(cc.Resources)),
		Stakeholders: make([]ContextStakeholder, len(cc.Stakeholders)),
		Attributes:   make(map[string]any),
	}

	copy(clone.Constraints, cc.Constraints)
	copy(clone.Resources, cc.Resources)
	copy(clone.Stakeholders, cc.Stakeholders)

	maps.Copy(clone.Attributes, cc.Attributes)

	return clone
}

// ContextConstraint 上下文约束
type ContextConstraint struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Attributes  any    `json:"attributes"`
}

// ContextResource 上下文资源
type ContextResource struct {
	Type         string `json:"type"`
	Description  string `json:"description"`
	Availability string `json:"availability"`
	Attributes   any    `json:"attributes"`
}

// ContextStakeholder 上下文利益相关者
type ContextStakeholder struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Influence  string `json:"influence"`
	Interest   string `json:"interest"`
	Attributes any    `json:"attributes"`
}

// CultureEngine 文化引擎接口
type CultureEngine interface {
	// 文化管理
	CreateCulture(culture *Culture) error
	UpdateCulture(culture *Culture) error
	DeleteCulture(cultureID string) error
	GetCulture(cultureID string) (*Culture, error)
	ListCultures(filters map[string]any) ([]*Culture, error)

	// 文化分析
	AnalyzeCulture(cultureID string) (*CultureAnalysis, error)
	CompareCultures(cultureID1, cultureID2 string) (*CultureComparison, error)
	MatchCultures(request *CultureMatchRequest) (*CultureMatch, error)

	// 文化适应
	AdaptCulture(cultureID string, context *CultureContext) (*AdaptationResult, error)
	GenerateAdaptationPlan(cultureID string, requirements []AdaptationRequirement) (*AdaptationPlan, error)

	// 文化评估
	EvaluateCultureFit(cultureID string, context *CultureContext) (*FitAssessment, error)
	PredictCulturalChallenges(cultureID string, scenario string) (*ChallengePrediction, error)

	// 文化学习
	LearnFromInteractions(cultureID string, interactions []CulturalInteraction) (*LearningResult, error)
	UpdateCultureProfile(cultureID string, feedback *CultureFeedback) error

	// 文化应用
	ApplyCultureGuidance(cultureID string, situation *Situation) (*Guidance, error)
	GenerateCulturalRecommendations(cultureID string, context *CultureContext) (*Recommendations, error)
}

// CultureAnalysis 文化分析结果
type CultureAnalysis struct {
	CultureID       string               `json:"culture_id"`
	AnalysisDate    time.Time            `json:"analysis_date"`
	OverallScore    float64              `json:"overall_score"`
	DimensionScores map[string]float64   `json:"dimension_scores"`
	Strengths       []string             `json:"strengths"`
	Weaknesses      []string             `json:"weaknesses"`
	Characteristics []string             `json:"characteristics"`
	Compatibility   []CompatibilityScore `json:"compatibility"`
	Recommendations []string             `json:"recommendations"`
	Metadata        map[string]any       `json:"metadata"`
}

// CompatibilityScore 兼容性评分
type CompatibilityScore struct {
	TargetCulture string   `json:"target_culture"`
	Score         float64  `json:"score"`
	Level         string   `json:"level"`
	Issues        []string `json:"issues"`
}

// CultureComparison 文化比较
type CultureComparison struct {
	Culture1ID           string               `json:"culture1_id"`
	Culture2ID           string               `json:"culture2_id"`
	ComparisonDate       time.Time            `json:"comparison_date"`
	Similarity           float64              `json:"similarity"`
	Differences          []CultureDifference  `json:"differences"`
	Commonalities        []string             `json:"commonalities"`
	PotentialConflicts   []ConflictArea       `json:"potential_conflicts"`
	SynergyOpportunities []SynergyOpportunity `json:"synergy_opportunities"`
	Metadata             map[string]any       `json:"metadata"`
}

// CultureDifference 文化差异
type CultureDifference struct {
	Dimension     string  `json:"dimension"`
	Culture1Value float64 `json:"culture1_value"`
	Culture2Value float64 `json:"culture2_value"`
	Magnitude     float64 `json:"magnitude"`
	Impact        string  `json:"impact"`
	Description   string  `json:"description"`
}

// ConflictArea 冲突领域
type ConflictArea struct {
	Area        string   `json:"area"`
	Probability float64  `json:"probability"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	Mitigation  []string `json:"mitigation"`
}

// SynergyOpportunity 协同机会
type SynergyOpportunity struct {
	Area         string   `json:"area"`
	Potential    float64  `json:"potential"`
	Description  string   `json:"description"`
	Requirements []string `json:"requirements"`
	Benefits     []string `json:"benefits"`
}

// CultureMatchRequest 文化匹配请求
type CultureMatchRequest struct {
	TargetCulture string             `json:"target_culture"`
	Preferences   map[string]any     `json:"preferences"`
	Constraints   []string           `json:"constraints"`
	Context       *CultureContext    `json:"context"`
	Priority      []string           `json:"priority"`
	Requirements  []MatchRequirement `json:"requirements"`
}

// MatchRequirement 匹配需求
type MatchRequirement struct {
	Type        string  `json:"type"`
	Criterion   string  `json:"criterion"`
	Weight      float64 `json:"weight"`
	Minimum     float64 `json:"minimum"`
	Preferred   float64 `json:"preferred"`
	Description string  `json:"description"`
}

// CultureMatch 文化匹配结果
type CultureMatch struct {
	RequestID       string               `json:"request_id"`
	MatchedCultures []CultureMatchResult `json:"matched_cultures"`
	BestMatch       *CultureMatchResult  `json:"best_match,omitempty"`
	MatchDate       time.Time            `json:"match_date"`
	Score           float64              `json:"score"`
	Confidence      float64              `json:"confidence"`
	Recommendations []string             `json:"recommendations"`
}

// CultureMatchResult 文化匹配结果
type CultureMatchResult struct {
	CultureID   string         `json:"culture_id"`
	CultureName string         `json:"culture_name"`
	Score       float64        `json:"score"`
	Fit         string         `json:"fit"`
	Strengths   []string       `json:"strengths"`
	Weaknesses  []string       `json:"weaknesses"`
	Analysis    map[string]any `json:"analysis"`
}

// AdaptationResult 适应结果
type AdaptationResult struct {
	CultureID       string              `json:"culture_id"`
	AdaptationDate  time.Time           `json:"adaptation_date"`
	Success         bool                `json:"success"`
	Adaptations     []AppliedAdaptation `json:"adaptations"`
	Changes         []CultureChange     `json:"changes"`
	Impact          AdaptationImpact    `json:"impact"`
	Recommendations []string            `json:"recommendations"`
	Metadata        map[string]any      `json:"metadata"`
}

// AppliedAdaptation 应用的适应
type AppliedAdaptation struct {
	StrategyID string    `json:"strategy_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Applied    bool      `json:"applied"`
	Effect     string    `json:"effect"`
	Timestamp  time.Time `json:"timestamp"`
}

// CultureChange 文化变更
type CultureChange struct {
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Impact      string         `json:"impact"`
	Magnitude   float64        `json:"magnitude"`
	Status      string         `json:"status"`
	Attributes  map[string]any `json:"attributes"`
}

// AdaptationImpact 适应影响
type AdaptationImpact struct {
	Positive   float64            `json:"positive"`
	Negative   float64            `json:"negative"`
	Neutral    float64            `json:"neutral"`
	Overall    float64            `json:"overall"`
	Categories map[string]float64 `json:"categories"`
}

// AdaptationPlan 适应计划
type AdaptationPlan struct {
	PlanID       string                  `json:"plan_id"`
	CultureID    string                  `json:"culture_id"`
	CreatedDate  time.Time               `json:"created_date"`
	TargetDate   time.Time               `json:"target_date"`
	Status       PlanStatus              `json:"status"`
	Priority     int                     `json:"priority"`
	Requirements []AdaptationRequirement `json:"requirements"`
	Strategies   []AdaptationStrategy    `json:"strategies"`
	Timeline     []PlanTimeline          `json:"timeline"`
	Resources    []PlanResource          `json:"resources"`
	Risks        []PlanRisk              `json:"risks"`
	Metrics      []PlanMetric            `json:"metrics"`
	Metadata     map[string]any          `json:"metadata"`
}

// AdaptationRequirement 适应需求
type AdaptationRequirement struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Priority    int            `json:"priority"`
	Source      string         `json:"source"`
	Attributes  map[string]any `json:"attributes"`
}

// PlanStatus 计划状态
type PlanStatus string

const (
	PlanStatusDraft      PlanStatus = "draft"       // 草稿
	PlanStatusApproved   PlanStatus = "approved"    // 已批准
	PlanStatusInProgress PlanStatus = "in_progress" // 进行中
	PlanStatusCompleted  PlanStatus = "completed"   // 已完成
	PlanStatusCancelled  PlanStatus = "canceled"    // 已取消
	PlanStatusSuspended  PlanStatus = "suspended"   // 已暂停
)

// PlanTimeline 计划时间线
type PlanTimeline struct {
	Phase        string    `json:"phase"`
	Description  string    `json:"description"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	Status       string    `json:"status"`
	Dependencies []string  `json:"dependencies"`
}

// PlanResource 计划资源
type PlanResource struct {
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Quantity     int            `json:"quantity"`
	Availability string         `json:"availability"`
	Attributes   map[string]any `json:"attributes"`
}

// PlanRisk 计划风险
type PlanRisk struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Probability float64        `json:"probability"`
	Impact      string         `json:"impact"`
	Mitigation  []string       `json:"mitigation"`
	Attributes  map[string]any `json:"attributes"`
}

// PlanMetric 计划指标
type PlanMetric struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Target      float64 `json:"target"`
	Current     float64 `json:"current"`
	Unit        string  `json:"unit"`
}

// FitAssessment 适配评估
type FitAssessment struct {
	CultureID       string             `json:"culture_id"`
	AssessmentDate  time.Time          `json:"assessment_date"`
	OverallFit      float64            `json:"overall_fit"`
	DimensionFits   map[string]float64 `json:"dimension_fits"`
	Strengths       []string           `json:"strengths"`
	Concerns        []string           `json:"concerns"`
	Gaps            []FitGap           `json:"gaps"`
	Recommendations []string           `json:"recommendations"`
	Confidence      float64            `json:"confidence"`
	Metadata        map[string]any     `json:"metadata"`
}

// FitGap 适配差距
type FitGap struct {
	Dimension   string  `json:"dimension"`
	Current     float64 `json:"current"`
	Target      float64 `json:"target"`
	Gap         float64 `json:"gap"`
	Priority    string  `json:"priority"`
	Description string  `json:"description"`
}

// ChallengePrediction 挑战预测
type ChallengePrediction struct {
	CultureID       string               `json:"culture_id"`
	Scenario        string               `json:"scenario"`
	PredictionDate  time.Time            `json:"prediction_date"`
	Challenges      []PredictedChallenge `json:"challenges"`
	OverallRisk     security.RiskLevel   `json:"overall_risk"`
	Recommendations []string             `json:"recommendations"`
	Confidence      float64              `json:"confidence"`
	Metadata        map[string]any       `json:"metadata"`
}

// PredictedChallenge 预测的挑战
type PredictedChallenge struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Probability float64  `json:"probability"`
	Impact      string   `json:"impact"`
	Severity    string   `json:"severity"`
	Timeline    string   `json:"timeline"`
	Mitigation  []string `json:"mitigation"`
}

// CulturalInteraction 文化互动
type CulturalInteraction struct {
	ID              string                   `json:"id"`
	CultureID       string                   `json:"culture_id"`
	InteractionType string                   `json:"interaction_type"`
	Participants    []InteractionParticipant `json:"participants"`
	Context         string                   `json:"context"`
	Content         string                   `json:"content"`
	Outcome         string                   `json:"outcome"`
	Timestamp       time.Time                `json:"timestamp"`
	Metadata        map[string]any           `json:"metadata"`
}

// InteractionParticipant 互动参与者
type InteractionParticipant struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Culture  string `json:"culture"`
	Behavior string `json:"behavior"`
}

// LearningResult 学习结果
type LearningResult struct {
	CultureID    string               `json:"culture_id"`
	LearningDate time.Time            `json:"learning_date"`
	Interactions int                  `json:"interactions"`
	Patterns     []LearnedPattern     `json:"patterns"`
	Adjustments  []CultureAdjustment  `json:"adjustments"`
	Improvements []CultureImprovement `json:"improvements"`
	Confidence   float64              `json:"confidence"`
	Metadata     map[string]any       `json:"metadata"`
}

// LearnedPattern 学习的模式
type LearnedPattern struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Frequency   float64 `json:"frequency"`
	Confidence  float64 `json:"confidence"`
	Context     string  `json:"context"`
}

// CultureAdjustment 文化调整
type CultureAdjustment struct {
	Dimension  string  `json:"dimension"`
	Adjustment float64 `json:"adjustment"`
	Reason     string  `json:"reason"`
	Impact     string  `json:"impact"`
}

// CultureImprovement 文化改进
type CultureImprovement struct {
	Area        string `json:"area"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Effort      string `json:"effort"`
	Benefit     string `json:"benefit"`
}

// CultureFeedback 文化反馈
type CultureFeedback struct {
	ID          string         `json:"id"`
	CultureID   string         `json:"culture_id"`
	Source      string         `json:"source"`
	Type        FeedbackType   `json:"type"`
	Content     string         `json:"content"`
	Rating      int            `json:"rating"`
	Categories  []string       `json:"categories"`
	Suggestions []string       `json:"suggestions"`
	Timestamp   time.Time      `json:"timestamp"`
	Metadata    map[string]any `json:"metadata"`
}

// FeedbackType 反馈类型
type FeedbackType string

const (
	FeedbackTypePerformance FeedbackType = "performance" // 绩效反馈
	FeedbackTypeBehavior    FeedbackType = "behavior"    // 行为反馈
	FeedbackTypeOutcome     FeedbackType = "outcome"     // 结果反馈
	FeedbackTypeProcess     FeedbackType = "process"     // 过程反馈
	FeedbackTypeCultural    FeedbackType = "cultural"    // 文化反馈
)

// Situation 情境
type Situation struct {
	ID           string         `json:"id"`
	Type         SituationType  `json:"type"`
	Description  string         `json:"description"`
	Context      map[string]any `json:"context"`
	Participants []string       `json:"participants"`
	Objectives   []string       `json:"objectives"`
	Constraints  []string       `json:"constraints"`
	Timestamp    time.Time      `json:"timestamp"`
	Metadata     map[string]any `json:"metadata"`
}

// SituationType 情境类型
type SituationType string

const (
	SituationTypeCommunication SituationType = "communication" // 沟通情境
	SituationTypeDecision      SituationType = "decision"      // 决策情境
	SituationTypeConflict      SituationType = "conflict"      // 冲突情境
	SituationTypeCollaboration SituationType = "collaboration" // 协作情境
	SituationTypeNegotiation   SituationType = "negotiation"   // 谈判情境
	SituationTypeLeadership    SituationType = "leadership"    // 领导情境
	SituationTypeTeamwork      SituationType = "teamwork"      // 团队情境
	SituationTypeLearning      SituationType = "learning"      // 学习情境
)

// Guidance 指导
type Guidance struct {
	ID           string                `json:"id"`
	CultureID    string                `json:"culture_id"`
	SituationID  string                `json:"situation_id"`
	Type         GuidanceType          `json:"type"`
	Priority     int                   `json:"priority"`
	Advice       []GuidanceAdvice      `json:"advice"`
	Principles   []string              `json:"principles"`
	Alternatives []GuidanceAlternative `json:"alternatives"`
	Timing       GuidanceTiming        `json:"timing"`
	Impact       GuidanceImpact        `json:"impact"`
	Confidence   float64               `json:"confidence"`
	Metadata     map[string]any        `json:"metadata"`
}

// GuidanceType 指导类型
type GuidanceType string

const (
	GuidanceTypeRecommendation GuidanceType = "recommendation" // 推荐
	GuidanceTypeWarning        GuidanceType = "warning"        // 警告
	GuidanceTypeInstruction    GuidanceType = "instruction"    // 指导
	GuidanceTypeExplanation    GuidanceType = "explanation"    // 解释
	GuidanceTypeSuggestion     GuidanceType = "suggestion"     // 建议
	GuidanceTypeFeedback       GuidanceType = "feedback"       // 反馈
)

// GuidanceAdvice 指导建议
type GuidanceAdvice struct {
	Action     string   `json:"action"`
	Reason     string   `json:"reason"`
	Steps      []string `json:"steps"`
	Outcome    string   `json:"outcome"`
	Confidence float64  `json:"confidence"`
}

// GuidanceAlternative 指导替代方案
type GuidanceAlternative struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
}

// GuidanceTiming 指导时机
type GuidanceTiming string

const (
	GuidanceTimingImmediate  GuidanceTiming = "immediate"  // 立即
	GuidanceTimingEarly      GuidanceTiming = "early"      // 早期
	GuidanceTimingDeliberate GuidanceTiming = "deliberate" // 深思熟虑
	GuidanceTimingLater      GuidanceTiming = "later"      // 后期
)

// GuidanceImpact 指导影响
type GuidanceImpact struct {
	Type          string  `json:"type"`
	Magnitude     float64 `json:"magnitude"`
	Duration      string  `json:"duration"`
	Reversibility string  `json:"reversibility"`
}

// Recommendations 推荐
type Recommendations struct {
	ID              string           `json:"id"`
	CultureID       string           `json:"culture_id"`
	Context         *CultureContext  `json:"context"`
	Recommendations []Recommendation `json:"recommendations"`
	GeneratedAt     time.Time        `json:"generated_at"`
	Confidence      float64          `json:"confidence"`
	Metadata        map[string]any   `json:"metadata"`
}

// Recommendation 推荐
type Recommendation struct {
	Type        RecommendationType     `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Priority    int                    `json:"priority"`
	Actions     []RecommendationAction `json:"actions"`
	Benefits    []string               `json:"benefits"`
	Risks       []string               `json:"risks"`
	Effort      string                 `json:"effort"`
	Timeline    string                 `json:"timeline"`
}

// RecommendationType 推荐类型
type RecommendationType string

const (
	RecommendationTypeAction      RecommendationType = "action"      // 行动推荐
	RecommendationTypeStrategy    RecommendationType = "strategy"    // 策略推荐
	RecommendationTypeProcess     RecommendationType = "process"     // 过程推荐
	RecommendationTypeLearning    RecommendationType = "learning"    // 学习推荐
	RecommendationTypeAdjustment  RecommendationType = "adjustment"  // 调整推荐
	RecommendationTypeImprovement RecommendationType = "improvement" // 改进推荐
)

// RecommendationAction 推荐行动
type RecommendationAction struct {
	Step        string   `json:"step"`
	Description string   `json:"description"`
	Resources   []string `json:"resources"`
	Timeline    string   `json:"timeline"`
}

// NewCulture 创建新文化
func NewCulture(id, name, description string) *Culture {
	return &Culture{
		ID:                   id,
		Name:                 name,
		Description:          description,
		Version:              "1.0.0",
		Dimensions:           make([]CultureDimension, 0),
		Norms:                make([]Norm, 0),
		Values:               make([]Value, 0),
		Beliefs:              make([]Belief, 0),
		Behaviors:            make([]BehaviorPattern, 0),
		Rituals:              make([]Ritual, 0),
		CommunicationStyles:  make([]CommunicationStyle, 0),
		DecisionStyles:       make([]DecisionStyle, 0),
		ConflictResolution:   make([]ConflictResolutionStrategy, 0),
		LearningStyles:       make([]LearningStyle, 0),
		AdaptationStrategies: make([]AdaptationStrategy, 0),
		Metadata:             make(map[string]any),
		Tags:                 make([]string, 0),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		Active:               true,
	}
}

// AddDimension 添加文化维度
func (c *Culture) AddDimension(dimension CultureDimension) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Dimensions = append(c.Dimensions, dimension)
	c.UpdatedAt = time.Now()
}

// AddNorm 添加规范
func (c *Culture) AddNorm(norm Norm) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Norms = append(c.Norms, norm)
	c.UpdatedAt = time.Now()
}

// AddValue 添加价值观
func (c *Culture) AddValue(value Value) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Values = append(c.Values, value)
	c.UpdatedAt = time.Now()
}

// AddBelief 添加信念
func (c *Culture) AddBelief(belief Belief) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Beliefs = append(c.Beliefs, belief)
	c.UpdatedAt = time.Now()
}

// GetDimensionValue 获取维度值
func (c *Culture) GetDimensionValue(dimensionType DimensionType) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, dimension := range c.Dimensions {
		if dimension.Type == dimensionType {
			return dimension.Value, true
		}
	}

	return 0.0, false
}

// SetDimensionValue 设置维度值
func (c *Culture) SetDimensionValue(dimensionType DimensionType, value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, dimension := range c.Dimensions {
		if dimension.Type == dimensionType {
			c.Dimensions[i].Value = value
			c.UpdatedAt = time.Now()
			return
		}
	}

	// 如果维度不存在，创建新维度
	newDimension := CultureDimension{
		ID:       string(dimensionType),
		Type:     dimensionType,
		Name:     string(dimensionType),
		Value:    value,
		MinValue: 0.0,
		MaxValue: 1.0,
	}

	c.Dimensions = append(c.Dimensions, newDimension)
	c.UpdatedAt = time.Now()
}

// ToJSON 转换为JSON
func (c *Culture) ToJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return json.MarshalIndent(c, "", "  ")
}

// Validate 验证文化定义
func (c *Culture) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ID == "" {
		return errors.New("culture ID is required")
	}

	if c.Name == "" {
		return errors.New("culture name is required")
	}

	// 验证维度值范围
	for _, dimension := range c.Dimensions {
		if dimension.Value < dimension.MinValue || dimension.Value > dimension.MaxValue {
			return fmt.Errorf("dimension %s value %f is out of range [%f, %f]",
				dimension.Type, dimension.Value, dimension.MinValue, dimension.MaxValue)
		}
	}

	return nil
}

// Clone 克隆文化
func (c *Culture) Clone() *Culture {
	c.mu.RLock()
	defer c.mu.RUnlock()

	clone := &Culture{
		ID:                   c.ID,
		Name:                 c.Name,
		Description:          c.Description,
		Version:              c.Version,
		Dimensions:           make([]CultureDimension, len(c.Dimensions)),
		Norms:                make([]Norm, len(c.Norms)),
		Values:               make([]Value, len(c.Values)),
		Beliefs:              make([]Belief, len(c.Beliefs)),
		Behaviors:            make([]BehaviorPattern, len(c.Behaviors)),
		Rituals:              make([]Ritual, len(c.Rituals)),
		CommunicationStyles:  make([]CommunicationStyle, len(c.CommunicationStyles)),
		DecisionStyles:       make([]DecisionStyle, len(c.DecisionStyles)),
		ConflictResolution:   make([]ConflictResolutionStrategy, len(c.ConflictResolution)),
		LearningStyles:       make([]LearningStyle, len(c.LearningStyles)),
		AdaptationStrategies: make([]AdaptationStrategy, len(c.AdaptationStrategies)),
		Metadata:             make(map[string]any),
		Tags:                 make([]string, len(c.Tags)),
		CreatedAt:            c.CreatedAt,
		UpdatedAt:            c.UpdatedAt,
		CreatedBy:            c.CreatedBy,
		UpdatedBy:            c.UpdatedBy,
		Active:               c.Active,
	}

	// 复制切片
	copy(clone.Dimensions, c.Dimensions)
	copy(clone.Norms, c.Norms)
	copy(clone.Values, c.Values)
	copy(clone.Beliefs, c.Beliefs)
	copy(clone.Behaviors, c.Behaviors)
	copy(clone.Rituals, c.Rituals)
	copy(clone.CommunicationStyles, c.CommunicationStyles)
	copy(clone.DecisionStyles, c.DecisionStyles)
	copy(clone.ConflictResolution, c.ConflictResolution)
	copy(clone.LearningStyles, c.LearningStyles)
	copy(clone.AdaptationStrategies, c.AdaptationStrategies)
	copy(clone.Tags, c.Tags)

	// 复制map
	maps.Copy(clone.Metadata, c.Metadata)

	if c.Context != nil {
		clone.Context = c.Context.Clone()
	}

	return clone
}
