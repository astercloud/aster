package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Constraint 约束接口
type Constraint interface {
	Name() string
	Validate(value interface{}) (bool, string) // 返回 (是否满足, 错误信息)
}

// ConstraintSet 约束集合
type ConstraintSet struct {
	constraints map[string]Constraint
}

func NewConstraintSet() *ConstraintSet {
	return &ConstraintSet{
		constraints: make(map[string]Constraint),
	}
}

func (cs *ConstraintSet) Add(constraint Constraint) error {
	if constraint == nil {
		return fmt.Errorf("constraint cannot be nil")
	}
	cs.constraints[constraint.Name()] = constraint
	return nil
}

func (cs *ConstraintSet) Validate(values map[string]interface{}) (bool, []string) {
	violations := make([]string, 0)

	for name, constraint := range cs.constraints {
		value, exists := values[name]
		if !exists {
			violations = append(violations, fmt.Sprintf("required value missing: %s", name))
			continue
		}

		passed, msg := constraint.Validate(value)
		if !passed {
			violations = append(violations, msg)
		}
	}

	return len(violations) == 0, violations
}

func (cs *ConstraintSet) List() []Constraint {
	constraints := make([]Constraint, 0, len(cs.constraints))
	for _, c := range cs.constraints {
		constraints = append(constraints, c)
	}
	return constraints
}

// ===== RangeConstraint =====

type RangeConstraint struct {
	name string
	min  int64
	max  int64
}

func NewRangeConstraint(name string, min, max int64) *RangeConstraint {
	return &RangeConstraint{
		name: name,
		min:  min,
		max:  max,
	}
}

func (c *RangeConstraint) Name() string { return c.name }

func (c *RangeConstraint) Validate(value interface{}) (bool, string) {
	if value == nil {
		return false, fmt.Sprintf("%s: value is nil", c.name)
	}

	var num int64

	switch v := value.(type) {
	case int:
		num = int64(v)
	case int32:
		num = int64(v)
	case int64:
		num = v
	case float64:
		num = int64(v)
	case string:
		var err error
		num, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return false, fmt.Sprintf("%s: invalid number format", c.name)
		}
	default:
		return false, fmt.Sprintf("%s: unsupported type %T", c.name, value)
	}

	if num < c.min || num > c.max {
		return false, fmt.Sprintf("%s: value %d not in range [%d, %d]", c.name, num, c.min, c.max)
	}

	return true, ""
}

// ===== StringLengthConstraint =====

type StringLengthConstraint struct {
	name   string
	minLen int
	maxLen int
}

func NewStringLengthConstraint(name string, minLen, maxLen int) *StringLengthConstraint {
	return &StringLengthConstraint{
		name:   name,
		minLen: minLen,
		maxLen: maxLen,
	}
}

func (c *StringLengthConstraint) Name() string { return c.name }

func (c *StringLengthConstraint) Validate(value interface{}) (bool, string) {
	if value == nil {
		return false, fmt.Sprintf("%s: value is nil", c.name)
	}

	str, ok := value.(string)
	if !ok {
		return false, fmt.Sprintf("%s: value is not string", c.name)
	}

	length := len(str)
	if length < c.minLen || length > c.maxLen {
		return false, fmt.Sprintf("%s: length %d not in range [%d, %d]", c.name, length, c.minLen, c.maxLen)
	}

	return true, ""
}

// ===== PatternConstraint =====

type PatternConstraint struct {
	name    string
	pattern string
	regex   *regexp.Regexp
}

func NewPatternConstraint(name string, pattern string) (*PatternConstraint, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &PatternConstraint{
		name:    name,
		pattern: pattern,
		regex:   regex,
	}, nil
}

func (c *PatternConstraint) Name() string { return c.name }

func (c *PatternConstraint) Validate(value interface{}) (bool, string) {
	if value == nil {
		return false, fmt.Sprintf("%s: value is nil", c.name)
	}

	str, ok := value.(string)
	if !ok {
		return false, fmt.Sprintf("%s: value is not string", c.name)
	}

	if !c.regex.MatchString(str) {
		return false, fmt.Sprintf("%s: value does not match pattern %s", c.name, c.pattern)
	}

	return true, ""
}

// ===== ChoiceConstraint =====

type ChoiceConstraint struct {
	name            string
	choices         []string
	caseInsensitive bool
}

func NewChoiceConstraint(name string, choices []string) *ChoiceConstraint {
	return &ChoiceConstraint{
		name:    name,
		choices: choices,
	}
}

func (c *ChoiceConstraint) WithCaseInsensitive() *ChoiceConstraint {
	c.caseInsensitive = true
	return c
}

func (c *ChoiceConstraint) Name() string { return c.name }

func (c *ChoiceConstraint) Validate(value interface{}) (bool, string) {
	if value == nil {
		return false, fmt.Sprintf("%s: value is nil", c.name)
	}

	str, ok := value.(string)
	if !ok {
		return false, fmt.Sprintf("%s: value is not string", c.name)
	}

	compareStr := str
	if c.caseInsensitive {
		compareStr = strings.ToLower(str)
	}

	for _, choice := range c.choices {
		compareChoice := choice
		if c.caseInsensitive {
			compareChoice = strings.ToLower(choice)
		}

		if compareChoice == compareStr {
			return true, ""
		}
	}

	return false, fmt.Sprintf("%s: value %s not in choices [%s]", c.name, str, strings.Join(c.choices, ", "))
}

// ===== NotEmptyConstraint =====

type NotEmptyConstraint struct {
	name string
}

func NewNotEmptyConstraint(name string) *NotEmptyConstraint {
	return &NotEmptyConstraint{name: name}
}

func (c *NotEmptyConstraint) Name() string { return c.name }

func (c *NotEmptyConstraint) Validate(value interface{}) (bool, string) {
	if value == nil {
		return false, fmt.Sprintf("%s: value is nil", c.name)
	}

	switch v := value.(type) {
	case string:
		if len(strings.TrimSpace(v)) == 0 {
			return false, fmt.Sprintf("%s: value is empty", c.name)
		}
	case []interface{}:
		if len(v) == 0 {
			return false, fmt.Sprintf("%s: list is empty", c.name)
		}
	}

	return true, ""
}

// ===== CustomConstraint =====

type CustomConstraint struct {
	name       string
	validateFn func(value interface{}) (bool, string)
}

func NewCustomConstraint(name string, validateFn func(value interface{}) (bool, string)) *CustomConstraint {
	return &CustomConstraint{
		name:       name,
		validateFn: validateFn,
	}
}

func (c *CustomConstraint) Name() string { return c.name }

func (c *CustomConstraint) Validate(value interface{}) (bool, string) {
	if c.validateFn == nil {
		return false, fmt.Sprintf("%s: validation function not defined", c.name)
	}
	return c.validateFn(value)
}
