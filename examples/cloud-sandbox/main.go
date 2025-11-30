// CloudSandbox 演示云端沙箱环境的使用，支持阿里云 AgentBay 和火山引擎
// 两种云平台的代码执行隔离环境。
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/astercloud/aster/pkg/sandbox/cloud"
)

func main() {
	// 选择要测试的云平台
	platform := os.Getenv("CLOUD_PLATFORM") // "aliyun" or "volcengine"
	if platform == "" {
		platform = "aliyun"
	}

	ctx := context.Background()

	switch platform {
	case "aliyun":
		testAliyun(ctx)
	case "volcengine":
		testVolcengine(ctx)
	default:
		log.Fatalf("Unknown platform: %s", platform)
	}
}

func testAliyun(ctx context.Context) {
	fmt.Println("=== Testing Aliyun AgentBay Sandbox ===")

	// 创建阿里云沙箱
	config := &cloud.AliyunConfig{
		MCPEndpoint:     os.Getenv("ALIYUN_MCP_ENDPOINT"),
		AccessKeyID:     os.Getenv("ALIYUN_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
		SecurityToken:   os.Getenv("ALIYUN_SECURITY_TOKEN"),
		Region:          "cn-hangzhou",
		WorkDir:         "/workspace",
	}

	sb, err := cloud.NewAliyunSandbox(config)
	if err != nil {
		log.Fatalf("Failed to create Aliyun sandbox: %v", err)
	}
	defer func() { _ = sb.Dispose() }()

	fmt.Printf("Sandbox created: %s\n\n", sb.Kind())

	// 测试 1: 执行命令
	fmt.Println("--- Test 1: Execute command ---")
	result, err := sb.Exec(ctx, "echo 'Hello from Aliyun Sandbox'", nil)
	if err != nil {
		log.Fatalf("Exec failed: %v", err)
	}
	fmt.Printf("Stdout: %s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("Stderr: %s\n", result.Stderr)
	}
	fmt.Printf("Exit Code: %d\n\n", result.Code)

	// 测试 2: 写入文件
	fmt.Println("--- Test 2: Write file ---")
	fs := sb.FS()
	err = fs.Write(ctx, "test.txt", "Hello World from Aliyun")
	if err != nil {
		log.Fatalf("Write failed: %v", err)
	}
	fmt.Println("File written successfully")

	// 测试 3: 读取文件
	fmt.Println("--- Test 3: Read file ---")
	content, err := fs.Read(ctx, "test.txt")
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}
	fmt.Printf("Content: %s\n\n", content)

	// 测试 4: 获取文件信息
	fmt.Println("--- Test 4: File info ---")
	info, err := fs.Stat(ctx, "test.txt")
	if err != nil {
		log.Fatalf("Stat failed: %v", err)
	}
	fmt.Printf("Path: %s\n", info.Path)
	fmt.Printf("Size: %d bytes\n", info.Size)
	fmt.Printf("IsDir: %v\n", info.IsDir)
	fmt.Printf("ModTime: %v\n\n", info.ModTime)

	// 测试 5: Glob 匹配
	fmt.Println("--- Test 5: Glob files ---")
	matches, err := fs.Glob(ctx, "*.txt", nil)
	if err != nil {
		log.Fatalf("Glob failed: %v", err)
	}
	fmt.Printf("Matched files: %v\n\n", matches)

	fmt.Println("=== All tests passed! ===")
}

func testVolcengine(ctx context.Context) {
	fmt.Println("=== Testing Volcengine Sandbox ===")

	// 创建火山引擎沙箱
	config := &cloud.VolcengineConfig{
		Endpoint:  os.Getenv("VOLCENGINE_ENDPOINT"),
		AccessKey: os.Getenv("VOLCENGINE_ACCESS_KEY"),
		SecretKey: os.Getenv("VOLCENGINE_SECRET_KEY"),
		Region:    "cn-beijing",
		WorkDir:   "/workspace",
		CPU:       2,
		Memory:    4096,
	}

	sb, err := cloud.NewVolcengineSandbox(config)
	if err != nil {
		log.Fatalf("Failed to create Volcengine sandbox: %v", err)
	}
	defer func() { _ = sb.Dispose() }()

	fmt.Printf("Sandbox created: %s\n", sb.Kind())
	fmt.Printf("Session ID: %s\n\n", sb.SessionID())

	// 测试 1: 执行命令
	fmt.Println("--- Test 1: Execute command ---")
	result, err := sb.Exec(ctx, "echo 'Hello from Volcengine Sandbox'", nil)
	if err != nil {
		log.Fatalf("Exec failed: %v", err)
	}
	fmt.Printf("Stdout: %s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("Stderr: %s\n", result.Stderr)
	}
	fmt.Printf("Exit Code: %d\n\n", result.Code)

	// 测试 2: 写入文件
	fmt.Println("--- Test 2: Write file ---")
	fs := sb.FS()
	err = fs.Write(ctx, "test.txt", "Hello World from Volcengine")
	if err != nil {
		log.Fatalf("Write failed: %v", err)
	}
	fmt.Println("File written successfully")

	// 测试 3: 读取文件
	fmt.Println("--- Test 3: Read file ---")
	content, err := fs.Read(ctx, "test.txt")
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}
	fmt.Printf("Content: %s\n\n", content)

	// 测试 4: 获取文件信息
	fmt.Println("--- Test 4: File info ---")
	info, err := fs.Stat(ctx, "test.txt")
	if err != nil {
		log.Fatalf("Stat failed: %v", err)
	}
	fmt.Printf("Path: %s\n", info.Path)
	fmt.Printf("Size: %d bytes\n", info.Size)
	fmt.Printf("IsDir: %v\n", info.IsDir)
	fmt.Printf("ModTime: %v\n\n", info.ModTime)

	// 测试 5: Glob 匹配
	fmt.Println("--- Test 5: Glob files ---")
	matches, err := fs.Glob(ctx, "*.txt", nil)
	if err != nil {
		log.Fatalf("Glob failed: %v", err)
	}
	fmt.Printf("Matched files: %v\n\n", matches)

	fmt.Println("=== All tests passed! ===")
}
