package workflow

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/stars"
	"github.com/google/uuid"
)

// Step 步骤接口
type Step interface {
	ID() string
	Name() string
	Type() StepType
	Description() string
	Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error]
	Config() *StepConfig
}

// ===== AgentStep =====

type AgentStep struct {
	id          string
	name        string
	description string
	agent       *agent.Agent
	config      *StepConfig
}

func NewAgentStep(name string, agent *agent.Agent) *AgentStep {
	return &AgentStep{
		id:    uuid.New().String(),
		name:  name,
		agent: agent,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeAgent,
			MaxRetries:  3,
			Timeout:     5 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (s *AgentStep) ID() string          { return s.id }
func (s *AgentStep) Name() string        { return s.name }
func (s *AgentStep) Type() StepType      { return StepTypeAgent }
func (s *AgentStep) Description() string { return s.description }
func (s *AgentStep) Config() *StepConfig { return s.config }

func (s *AgentStep) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		inputMessage := input.GetInputAsString()
		if inputMessage == "" && input.PreviousStepContent != nil {
			if str, ok := input.PreviousStepContent.(string); ok {
				inputMessage = str
			}
		}

		output := &StepOutput{
			StepID:    s.id,
			StepName:  s.name,
			StepType:  StepTypeAgent,
			Content:   fmt.Sprintf("Agent %s processed: %s", s.name, inputMessage),
			StartTime: startTime,
			EndTime:   time.Now(),
			Metadata:  make(map[string]interface{}),
			Metrics:   &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}

func (s *AgentStep) WithDescription(desc string) *AgentStep {
	s.description = desc
	return s
}

func (s *AgentStep) WithTimeout(timeout time.Duration) *AgentStep {
	s.config.Timeout = timeout
	return s
}

// ===== StarsStep =====

type StarsStep struct {
	id          string
	name        string
	description string
	stars       *stars.Stars
	config      *StepConfig
}

func NewStarsStep(name string, stars *stars.Stars) *StarsStep {
	return &StarsStep{
		id:    uuid.New().String(),
		name:  name,
		stars: stars,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeStars,
			MaxRetries:  3,
			Timeout:     10 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (s *StarsStep) ID() string          { return s.id }
func (s *StarsStep) Name() string        { return s.name }
func (s *StarsStep) Type() StepType      { return StepTypeStars }
func (s *StarsStep) Description() string { return s.description }
func (s *StarsStep) Config() *StepConfig { return s.config }

func (s *StarsStep) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		inputMessage := input.GetInputAsString()
		if inputMessage == "" && input.PreviousStepContent != nil {
			if str, ok := input.PreviousStepContent.(string); ok {
				inputMessage = str
			}
		}

		output := &StepOutput{
			StepID:    s.id,
			StepName:  s.name,
			StepType:  StepTypeStars,
			Content:   fmt.Sprintf("Stars team %s processed: %s", s.name, inputMessage),
			StartTime: startTime,
			EndTime:   time.Now(),
			Metadata:  make(map[string]interface{}),
			Metrics:   &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}

func (s *StarsStep) WithDescription(desc string) *StarsStep {
	s.description = desc
	return s
}

// ===== FunctionStep =====

type FunctionStep struct {
	id          string
	name        string
	description string
	executor    func(ctx context.Context, input *StepInput) (*StepOutput, error)
	config      *StepConfig
}

func NewFunctionStep(name string, executor func(ctx context.Context, input *StepInput) (*StepOutput, error)) *FunctionStep {
	return &FunctionStep{
		id:       uuid.New().String(),
		name:     name,
		executor: executor,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeFunction,
			MaxRetries:  1,
			Timeout:     1 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (s *FunctionStep) ID() string          { return s.id }
func (s *FunctionStep) Name() string        { return s.name }
func (s *FunctionStep) Type() StepType      { return StepTypeFunction }
func (s *FunctionStep) Description() string { return s.description }
func (s *FunctionStep) Config() *StepConfig { return s.config }

func (s *FunctionStep) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		output, err := s.executor(ctx, input)
		if err != nil {
			errorOutput := &StepOutput{
				StepID:    s.id,
				StepName:  s.name,
				StepType:  StepTypeFunction,
				Error:     err,
				StartTime: startTime,
				EndTime:   time.Now(),
			}
			errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
			yield(errorOutput, err)
			return
		}

		if output != nil {
			output.StepID = s.id
			output.StepName = s.name
			output.StepType = StepTypeFunction
			output.StartTime = startTime
			output.EndTime = time.Now()
			output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
			if output.Metrics == nil {
				output.Metrics = &StepMetrics{}
			}
			output.Metrics.ExecutionTime = output.Duration
		}
		yield(output, nil)
	}
}

func (s *FunctionStep) WithDescription(desc string) *FunctionStep {
	s.description = desc
	return s
}

func (s *FunctionStep) WithTimeout(timeout time.Duration) *FunctionStep {
	s.config.Timeout = timeout
	return s
}

// ===== Helper Functions =====

func SimpleFunction(name string, fn func(input interface{}) (interface{}, error)) *FunctionStep {
	return NewFunctionStep(name, func(ctx context.Context, stepInput *StepInput) (*StepOutput, error) {
		input := stepInput.Input
		if input == nil && stepInput.PreviousStepContent != nil {
			input = stepInput.PreviousStepContent
		}

		output, err := fn(input)
		if err != nil {
			return nil, err
		}

		return &StepOutput{
			Content:  output,
			Metadata: make(map[string]interface{}),
		}, nil
	})
}

func TransformFunction(name string, transform func(input interface{}) interface{}) *FunctionStep {
	return SimpleFunction(name, func(input interface{}) (interface{}, error) {
		return transform(input), nil
	})
}

// ===== ConditionStep =====

type ConditionStep struct {
	id          string
	name        string
	description string
	condition   func(*StepInput) bool
	ifTrue      Step
	ifFalse     Step
	config      *StepConfig
}

func NewConditionStep(name string, condition func(*StepInput) bool, ifTrue, ifFalse Step) *ConditionStep {
	return &ConditionStep{
		id:        uuid.New().String(),
		name:      name,
		condition: condition,
		ifTrue:    ifTrue,
		ifFalse:   ifFalse,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeCondition,
			MaxRetries:  1,
			Timeout:     5 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (s *ConditionStep) ID() string          { return s.id }
func (s *ConditionStep) Name() string        { return s.name }
func (s *ConditionStep) Type() StepType      { return StepTypeCondition }
func (s *ConditionStep) Description() string { return s.description }
func (s *ConditionStep) Config() *StepConfig { return s.config }

func (s *ConditionStep) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		conditionResult := s.condition(input)

		var branch Step
		var branchName string
		if conditionResult {
			branch = s.ifTrue
			branchName = "true"
		} else {
			branch = s.ifFalse
			branchName = "false"
		}

		var branchOutput *StepOutput
		for output, err := range branch.Execute(ctx, input) {
			if err != nil {
				errorOutput := &StepOutput{
					StepID:    s.id,
					StepName:  s.name,
					StepType:  StepTypeCondition,
					Error:     err,
					StartTime: startTime,
					EndTime:   time.Now(),
					Metadata:  map[string]interface{}{"condition": conditionResult, "branch": branchName},
				}
				errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
				yield(errorOutput, err)
				return
			}
			branchOutput = output
		}

		output := &StepOutput{
			StepID:      s.id,
			StepName:    s.name,
			StepType:    StepTypeCondition,
			Content:     branchOutput.Content,
			StartTime:   startTime,
			EndTime:     time.Now(),
			NestedSteps: []*StepOutput{branchOutput},
			Metadata:    map[string]interface{}{"condition": conditionResult, "branch": branchName},
			Metrics:     &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}

// ===== LoopStep =====

type LoopStep struct {
	id            string
	name          string
	description   string
	body          Step
	maxIterations int
	stopCondition func(*StepOutput) bool
	config        *StepConfig
}

func NewLoopStep(name string, body Step, maxIterations int) *LoopStep {
	return &LoopStep{
		id:            uuid.New().String(),
		name:          name,
		body:          body,
		maxIterations: maxIterations,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeLoop,
			MaxRetries:  1,
			Timeout:     30 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (s *LoopStep) ID() string          { return s.id }
func (s *LoopStep) Name() string        { return s.name }
func (s *LoopStep) Type() StepType      { return StepTypeLoop }
func (s *LoopStep) Description() string { return s.description }
func (s *LoopStep) Config() *StepConfig { return s.config }

func (s *LoopStep) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		var iterations []*StepOutput
		var lastOutput *StepOutput

		for i := 0; i < s.maxIterations; i++ {
			loopInput := &StepInput{
				Input:               input.Input,
				PreviousStepOutputs: input.PreviousStepOutputs,
				AdditionalData:      input.AdditionalData,
				SessionState:        input.SessionState,
			}

			if lastOutput != nil {
				loopInput.PreviousStepContent = lastOutput.Content
			}

			var iterOutput *StepOutput
			for output, err := range s.body.Execute(ctx, loopInput) {
				if err != nil {
					errorOutput := &StepOutput{
						StepID:      s.id,
						StepName:    s.name,
						StepType:    StepTypeLoop,
						Error:       err,
						StartTime:   startTime,
						EndTime:     time.Now(),
						NestedSteps: iterations,
						Metadata:    map[string]interface{}{"iterations": i, "max": s.maxIterations},
					}
					errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
					yield(errorOutput, err)
					return
				}
				iterOutput = output
			}

			iterations = append(iterations, iterOutput)
			lastOutput = iterOutput

			if s.stopCondition != nil && s.stopCondition(iterOutput) {
				break
			}

			if ctx.Err() != nil {
				yield(nil, ctx.Err())
				return
			}
		}

		output := &StepOutput{
			StepID:      s.id,
			StepName:    s.name,
			StepType:    StepTypeLoop,
			Content:     lastOutput.Content,
			StartTime:   startTime,
			EndTime:     time.Now(),
			NestedSteps: iterations,
			Metadata:    map[string]interface{}{"iterations": len(iterations), "max": s.maxIterations},
			Metrics:     &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}

func (s *LoopStep) WithStopCondition(condition func(*StepOutput) bool) *LoopStep {
	s.stopCondition = condition
	return s
}

// ===== ParallelStep =====

type ParallelStep struct {
	id          string
	name        string
	description string
	steps       []Step
	config      *StepConfig
}

func NewParallelStep(name string, steps ...Step) *ParallelStep {
	return &ParallelStep{
		id:    uuid.New().String(),
		name:  name,
		steps: steps,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeParallel,
			MaxRetries:  1,
			Timeout:     10 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (s *ParallelStep) ID() string          { return s.id }
func (s *ParallelStep) Name() string        { return s.name }
func (s *ParallelStep) Type() StepType      { return StepTypeParallel }
func (s *ParallelStep) Description() string { return s.description }
func (s *ParallelStep) Config() *StepConfig { return s.config }

func (s *ParallelStep) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		var wg sync.WaitGroup
		results := make([]*StepOutput, len(s.steps))
		errors := make([]error, len(s.steps))

		for i, step := range s.steps {
			wg.Add(1)
			go func(index int, st Step) {
				defer wg.Done()
				for output, err := range st.Execute(ctx, input) {
					if err != nil {
						errors[index] = err
						return
					}
					results[index] = output
				}
			}(i, step)
		}

		wg.Wait()

		var firstError error
		for _, err := range errors {
			if err != nil {
				firstError = err
				break
			}
		}

		if firstError != nil {
			errorOutput := &StepOutput{
				StepID:      s.id,
				StepName:    s.name,
				StepType:    StepTypeParallel,
				Error:       firstError,
				StartTime:   startTime,
				EndTime:     time.Now(),
				NestedSteps: results,
				Metadata:    map[string]interface{}{"parallel_steps": len(s.steps)},
			}
			errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
			yield(errorOutput, firstError)
			return
		}

		combinedContent := make(map[string]interface{})
		for i, result := range results {
			if result != nil {
				combinedContent[fmt.Sprintf("step_%d", i)] = result.Content
			}
		}

		output := &StepOutput{
			StepID:      s.id,
			StepName:    s.name,
			StepType:    StepTypeParallel,
			Content:     combinedContent,
			StartTime:   startTime,
			EndTime:     time.Now(),
			NestedSteps: results,
			Metadata:    map[string]interface{}{"parallel_steps": len(s.steps)},
			Metrics:     &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}

// ===== RouterStep =====

type RouterStep struct {
	id          string
	name        string
	description string
	router      func(*StepInput) string
	routes      map[string]Step
	defaultStep Step
	config      *StepConfig
}

func NewRouterStep(name string, router func(*StepInput) string, routes map[string]Step) *RouterStep {
	return &RouterStep{
		id:     uuid.New().String(),
		name:   name,
		router: router,
		routes: routes,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeRouter,
			MaxRetries:  1,
			Timeout:     5 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (s *RouterStep) ID() string          { return s.id }
func (s *RouterStep) Name() string        { return s.name }
func (s *RouterStep) Type() StepType      { return StepTypeRouter }
func (s *RouterStep) Description() string { return s.description }
func (s *RouterStep) Config() *StepConfig { return s.config }

func (s *RouterStep) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		routeName := s.router(input)
		step, exists := s.routes[routeName]
		if !exists {
			if s.defaultStep == nil {
				err := fmt.Errorf("route '%s' not found", routeName)
				errorOutput := &StepOutput{
					StepID:    s.id,
					StepName:  s.name,
					StepType:  StepTypeRouter,
					Error:     err,
					StartTime: startTime,
					EndTime:   time.Now(),
					Metadata:  map[string]interface{}{"route": routeName},
				}
				errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
				yield(errorOutput, err)
				return
			}
			step = s.defaultStep
			routeName = "default"
		}

		var routeOutput *StepOutput
		for output, err := range step.Execute(ctx, input) {
			if err != nil {
				errorOutput := &StepOutput{
					StepID:    s.id,
					StepName:  s.name,
					StepType:  StepTypeRouter,
					Error:     err,
					StartTime: startTime,
					EndTime:   time.Now(),
					Metadata:  map[string]interface{}{"route": routeName},
				}
				errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
				yield(errorOutput, err)
				return
			}
			routeOutput = output
		}

		output := &StepOutput{
			StepID:      s.id,
			StepName:    s.name,
			StepType:    StepTypeRouter,
			Content:     routeOutput.Content,
			StartTime:   startTime,
			EndTime:     time.Now(),
			NestedSteps: []*StepOutput{routeOutput},
			Metadata:    map[string]interface{}{"route": routeName},
			Metrics:     &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}

func (s *RouterStep) WithDefault(step Step) *RouterStep {
	s.defaultStep = step
	return s
}

// ===== StepsGroup =====

type StepsGroup struct {
	id          string
	name        string
	description string
	steps       []Step
	config      *StepConfig
}

func NewStepsGroup(name string, steps ...Step) *StepsGroup {
	return &StepsGroup{
		id:    uuid.New().String(),
		name:  name,
		steps: steps,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeSteps,
			MaxRetries:  1,
			Timeout:     30 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (s *StepsGroup) ID() string          { return s.id }
func (s *StepsGroup) Name() string        { return s.name }
func (s *StepsGroup) Type() StepType      { return StepTypeSteps }
func (s *StepsGroup) Description() string { return s.description }
func (s *StepsGroup) Config() *StepConfig { return s.config }

func (s *StepsGroup) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		var outputs []*StepOutput
		var lastOutput *StepOutput

		for _, step := range s.steps {
			stepInput := &StepInput{
				Input:               input.Input,
				PreviousStepOutputs: input.PreviousStepOutputs,
				AdditionalData:      input.AdditionalData,
				SessionState:        input.SessionState,
			}

			if lastOutput != nil {
				stepInput.PreviousStepContent = lastOutput.Content
			}

			var stepOutput *StepOutput
			for output, err := range step.Execute(ctx, stepInput) {
				if err != nil {
					errorOutput := &StepOutput{
						StepID:      s.id,
						StepName:    s.name,
						StepType:    StepTypeSteps,
						Error:       err,
						StartTime:   startTime,
						EndTime:     time.Now(),
						NestedSteps: outputs,
						Metadata:    map[string]interface{}{"completed": len(outputs), "total": len(s.steps)},
					}
					errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
					yield(errorOutput, err)
					return
				}
				stepOutput = output
			}

			outputs = append(outputs, stepOutput)
			lastOutput = stepOutput

			if ctx.Err() != nil {
				yield(nil, ctx.Err())
				return
			}
		}

		output := &StepOutput{
			StepID:      s.id,
			StepName:    s.name,
			StepType:    StepTypeSteps,
			Content:     lastOutput.Content,
			StartTime:   startTime,
			EndTime:     time.Now(),
			NestedSteps: outputs,
			Metadata:    map[string]interface{}{"completed": len(outputs), "total": len(s.steps)},
			Metrics:     &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}
