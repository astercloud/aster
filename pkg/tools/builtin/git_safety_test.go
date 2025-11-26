package builtin

import (
	"strings"
	"testing"
)

// TestGitSafetyValidator_SafeCommands 测试安全的 Git 命令
func TestGitSafetyValidator_SafeCommands(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	safeCommands := []string{
		"git status",
		"git log",
		"git log --oneline -10",
		"git diff",
		"git diff HEAD~1",
		"git show HEAD",
		"git branch",
		"git remote",
		"git fetch origin",
		"git stash list",
	}

	for _, cmd := range safeCommands {
		result := validator.Check(cmd)
		if !result.IsGitCommand {
			t.Errorf("Expected %q to be recognized as a git command", cmd)
		}
		if result.Risk != GitRiskSafe {
			t.Errorf("Expected %q to be safe, got risk level %s", cmd, result.RiskName)
		}
		if result.RequiresApproval {
			t.Errorf("Safe command %q should not require approval", cmd)
		}
		if result.Blocked {
			t.Errorf("Safe command %q should not be blocked", cmd)
		}
	}
}

// TestGitSafetyValidator_LowRiskCommands 测试低风险命令
func TestGitSafetyValidator_LowRiskCommands(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	lowRiskCommands := []string{
		"git push origin feature-branch",
		"git pull --rebase origin main",
		"git merge feature-branch",
	}

	for _, cmd := range lowRiskCommands {
		result := validator.Check(cmd)
		if !result.IsGitCommand {
			t.Errorf("Expected %q to be recognized as a git command", cmd)
		}
		if result.Risk != GitRiskLow {
			t.Errorf("Expected %q to be low risk, got risk level %s", cmd, result.RiskName)
		}
		// 严格模式下低风险不需要批准
		if result.RequiresApproval {
			t.Errorf("Low risk command %q should not require approval in strict mode", cmd)
		}
		if result.Blocked {
			t.Errorf("Low risk command %q should not be blocked", cmd)
		}
	}
}

// TestGitSafetyValidator_MediumRiskCommands 测试中等风险命令
func TestGitSafetyValidator_MediumRiskCommands(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	mediumRiskCommands := []struct {
		cmd            string
		expectedReason string
	}{
		{"git commit --amend -m 'fix typo'", "amend"},
		{"git rebase -i HEAD~3", "Interactive rebase"},
		{"git push --no-verify", "bypass"},
		{"git commit --no-verify", "bypass"},
		{"git branch -d old-feature", "Deleting branches"},
		{"git push origin --delete old-branch", "irreversible"},
		{"git config --global user.name 'Test'", "global"},
		{"git submodule deinit lib", "submodule"},
		{"git cherry-pick abc123", "Cherry-pick"},
	}

	for _, tc := range mediumRiskCommands {
		result := validator.Check(tc.cmd)
		if !result.IsGitCommand {
			t.Errorf("Expected %q to be recognized as a git command", tc.cmd)
		}
		if result.Risk < GitRiskMedium {
			t.Errorf("Expected %q to be at least medium risk, got risk level %s", tc.cmd, result.RiskName)
		}
		// 严格模式下中等风险需要批准
		if !result.RequiresApproval {
			t.Errorf("Medium risk command %q should require approval in strict mode", tc.cmd)
		}
		if result.Blocked {
			t.Errorf("Medium risk command %q should not be blocked", tc.cmd)
		}
	}
}

// TestGitSafetyValidator_HighRiskCommands 测试高风险命令
func TestGitSafetyValidator_HighRiskCommands(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	highRiskCommands := []struct {
		cmd            string
		expectedReason string
	}{
		{"git push --force origin feature", "Force push"},
		{"git push --force-with-lease origin feature", "Force push"},
		{"git reset --hard HEAD~1", "Hard reset"},
		{"git reset --hard origin/main", "Hard reset"},
		{"git clean -fd", "clean"},
		{"git clean -fdx", "clean"},
		{"git config --system core.editor vim", "system"},
	}

	for _, tc := range highRiskCommands {
		result := validator.Check(tc.cmd)
		if !result.IsGitCommand {
			t.Errorf("Expected %q to be recognized as a git command", tc.cmd)
		}
		if result.Risk < GitRiskHigh {
			t.Errorf("Expected %q to be at least high risk, got risk level %s", tc.cmd, result.RiskName)
		}
		// 高风险命令需要批准
		if !result.RequiresApproval {
			t.Errorf("High risk command %q should require approval", tc.cmd)
		}
		if result.Blocked {
			t.Errorf("High risk command %q should not be blocked (only critical commands are blocked)", tc.cmd)
		}
	}
}

// TestGitSafetyValidator_CriticalCommands 测试极高风险命令（应被阻止）
func TestGitSafetyValidator_CriticalCommands(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	criticalCommands := []string{
		"git push --force origin main",
		"git push --force origin master",
		"git push origin main --force",
		"git push origin master --force",
		"git push -f origin main",
	}

	for _, cmd := range criticalCommands {
		result := validator.Check(cmd)
		if !result.IsGitCommand {
			t.Errorf("Expected %q to be recognized as a git command", cmd)
		}
		if result.Risk != GitRiskCritical {
			t.Errorf("Expected %q to be critical risk, got risk level %s", cmd, result.RiskName)
		}
		if !result.Blocked {
			t.Errorf("Critical command %q should be blocked", cmd)
		}
		if result.Reason == "" {
			t.Errorf("Blocked command %q should have a reason", cmd)
		}
	}
}

// TestGitSafetyValidator_NonGitCommands 测试非 Git 命令
func TestGitSafetyValidator_NonGitCommands(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	nonGitCommands := []string{
		"ls -la",
		"cd /tmp",
		"echo 'hello'",
		"npm install",
		"go build",
		"docker run",
		"github cli",
		"gitk",
	}

	for _, cmd := range nonGitCommands {
		result := validator.Check(cmd)
		if result.IsGitCommand {
			t.Errorf("Expected %q to NOT be recognized as a git command", cmd)
		}
		if result.RequiresApproval {
			t.Errorf("Non-git command %q should not require approval", cmd)
		}
		if result.Blocked {
			t.Errorf("Non-git command %q should not be blocked", cmd)
		}
	}
}

// TestGitSafetyValidator_SafetyLevels 测试不同安全级别
func TestGitSafetyValidator_SafetyLevels(t *testing.T) {
	mediumRiskCmd := "git commit --amend -m 'fix'"
	highRiskCmd := "git reset --hard HEAD~1"
	criticalCmd := "git push --force origin main"

	testCases := []struct {
		level         GitSafetyLevel
		levelName     string
		cmd           string
		shouldApprove bool
		shouldBlock   bool
	}{
		// 严格模式
		{GitSafetyLevelStrict, "strict", mediumRiskCmd, true, false},
		{GitSafetyLevelStrict, "strict", highRiskCmd, true, false},
		{GitSafetyLevelStrict, "strict", criticalCmd, true, true},

		// 正常模式
		{GitSafetyLevelNormal, "normal", mediumRiskCmd, false, false},
		{GitSafetyLevelNormal, "normal", highRiskCmd, true, false},
		{GitSafetyLevelNormal, "normal", criticalCmd, true, true},

		// 宽松模式
		{GitSafetyLevelPermissive, "permissive", mediumRiskCmd, false, false},
		{GitSafetyLevelPermissive, "permissive", highRiskCmd, false, false},
		{GitSafetyLevelPermissive, "permissive", criticalCmd, true, true},
	}

	for _, tc := range testCases {
		validator := NewGitSafetyValidator(tc.level)
		result := validator.Check(tc.cmd)

		if result.RequiresApproval != tc.shouldApprove {
			t.Errorf("[%s] %q: expected RequiresApproval=%v, got %v",
				tc.levelName, tc.cmd, tc.shouldApprove, result.RequiresApproval)
		}
		if result.Blocked != tc.shouldBlock {
			t.Errorf("[%s] %q: expected Blocked=%v, got %v",
				tc.levelName, tc.cmd, tc.shouldBlock, result.Blocked)
		}
	}
}

// TestGitSafetyValidator_Recommendations 测试建议生成
func TestGitSafetyValidator_Recommendations(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	testCases := []struct {
		cmd                string
		expectedSubstrings []string
	}{
		{
			cmd: "git push --force origin feature",
			expectedSubstrings: []string{
				"force-with-lease",
				"backup branch",
			},
		},
		{
			cmd: "git reset --hard HEAD~1",
			expectedSubstrings: []string{
				"stash",
				"reflog",
				"backup branch",
			},
		},
		{
			cmd: "git commit --amend -m 'fix'",
			expectedSubstrings: []string{
				"authorship",
				"pushed to remote",
				"backup branch",
			},
		},
		{
			cmd:                "git clean -fd",
			expectedSubstrings: []string{"clean -n"},
		},
		{
			cmd: "git rebase -i HEAD~3",
			expectedSubstrings: []string{
				"Interactive rebase",
				"non-interactive",
			},
		},
	}

	for _, tc := range testCases {
		result := validator.Check(tc.cmd)
		allRecs := strings.Join(result.Recommendations, " ")

		for _, expected := range tc.expectedSubstrings {
			if !strings.Contains(strings.ToLower(allRecs), strings.ToLower(expected)) {
				t.Errorf("Command %q: expected recommendation containing %q, got: %v",
					tc.cmd, expected, result.Recommendations)
			}
		}
	}
}

// TestGitSafetyValidator_FormatCheckResult 测试结果格式化
func TestGitSafetyValidator_FormatCheckResult(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	// 测试被阻止的命令
	blockedResult := validator.Check("git push --force origin main")
	formatted := blockedResult.FormatCheckResult()
	if !strings.Contains(formatted, "BLOCKED") {
		t.Error("Blocked command format should contain 'BLOCKED'")
	}
	if !strings.Contains(formatted, blockedResult.Reason) {
		t.Error("Blocked command format should contain the reason")
	}

	// 测试需要批准的命令
	approvalResult := validator.Check("git reset --hard HEAD")
	formatted = approvalResult.FormatCheckResult()
	if !strings.Contains(formatted, "WARNING") {
		t.Error("Approval required command format should contain 'WARNING'")
	}
	if !strings.Contains(formatted, "approval") {
		t.Error("Approval required command format should mention approval")
	}

	// 测试安全命令
	safeResult := validator.Check("git status")
	formatted = safeResult.FormatCheckResult()
	if formatted != "" {
		t.Errorf("Safe command should return empty format, got: %s", formatted)
	}

	// 测试非 Git 命令
	nonGitResult := validator.Check("ls -la")
	formatted = nonGitResult.FormatCheckResult()
	if formatted != "" {
		t.Errorf("Non-git command should return empty format, got: %s", formatted)
	}
}

// TestGitSafetyValidator_CaseInsensitive 测试大小写不敏感
func TestGitSafetyValidator_CaseInsensitive(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	commands := []string{
		"GIT STATUS",
		"Git Status",
		"git STATUS",
		"GIT push --FORCE origin MAIN",
	}

	for _, cmd := range commands {
		result := validator.Check(cmd)
		if !result.IsGitCommand {
			t.Errorf("Expected %q to be recognized as a git command (case insensitive)", cmd)
		}
	}

	// 验证危险命令在大小写变化时仍被识别
	forceMainResult := validator.Check("GIT PUSH --FORCE ORIGIN MAIN")
	if forceMainResult.Risk != GitRiskCritical {
		t.Errorf("Expected uppercase force push to main to be critical risk, got %s", forceMainResult.RiskName)
	}
}

// TestGitSafetyValidator_GlobalInstance 测试全局实例
func TestGitSafetyValidator_GlobalInstance(t *testing.T) {
	// 获取默认全局实例
	validator := GetGlobalGitSafetyValidator()
	if validator == nil {
		t.Fatal("Global git safety validator should not be nil")
	}

	// 验证默认是严格模式
	result := validator.Check("git commit --amend -m 'fix'")
	if !result.RequiresApproval {
		t.Error("Default global validator should be in strict mode (medium risk should require approval)")
	}

	// 测试设置不同级别
	SetGlobalGitSafetyLevel(GitSafetyLevelPermissive)
	validator = GetGlobalGitSafetyValidator()
	result = validator.Check("git commit --amend -m 'fix'")
	if result.RequiresApproval {
		t.Error("Permissive mode should not require approval for medium risk commands")
	}

	// 恢复为严格模式
	SetGlobalGitSafetyLevel(GitSafetyLevelStrict)
}

// TestGitSafetyValidator_Warnings 测试警告生成
func TestGitSafetyValidator_Warnings(t *testing.T) {
	validator := NewGitSafetyValidator(GitSafetyLevelStrict)

	testCases := []struct {
		cmd             string
		expectWarnings  bool
		warningContains string
	}{
		{"git push --force origin feature", true, "overwrite"},
		{"git reset --hard HEAD", true, "uncommitted"},
		{"git clean -fd", true, "untracked"},
		{"git status", false, ""},
		{"git log", false, ""},
	}

	for _, tc := range testCases {
		result := validator.Check(tc.cmd)
		hasWarnings := len(result.Warnings) > 0

		if hasWarnings != tc.expectWarnings {
			t.Errorf("Command %q: expected warnings=%v, got %v (warnings: %v)",
				tc.cmd, tc.expectWarnings, hasWarnings, result.Warnings)
		}

		if tc.expectWarnings && tc.warningContains != "" {
			allWarnings := strings.Join(result.Warnings, " ")
			if !strings.Contains(strings.ToLower(allWarnings), strings.ToLower(tc.warningContains)) {
				t.Errorf("Command %q: expected warning containing %q, got: %v",
					tc.cmd, tc.warningContains, result.Warnings)
			}
		}
	}
}

// TestGitSafetyValidator_RiskToName 测试风险名称转换
func TestGitSafetyValidator_RiskToName(t *testing.T) {
	testCases := []struct {
		risk     GitCommandRisk
		expected string
	}{
		{GitRiskSafe, "safe"},
		{GitRiskLow, "low"},
		{GitRiskMedium, "medium"},
		{GitRiskHigh, "high"},
		{GitRiskCritical, "critical"},
		{GitCommandRisk(100), "unknown"},
	}

	for _, tc := range testCases {
		result := riskToName(tc.risk)
		if result != tc.expected {
			t.Errorf("riskToName(%d): expected %q, got %q", tc.risk, tc.expected, result)
		}
	}
}
