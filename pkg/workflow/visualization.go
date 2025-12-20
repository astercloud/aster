package workflow

import (
	"fmt"
	"strings"
)

// WorkflowVisualizer 工作流可视化器
type WorkflowVisualizer struct {
	workflow *WorkflowDefinition
}

// NewWorkflowVisualizer 创建工作流可视化器
func NewWorkflowVisualizer(workflow *WorkflowDefinition) *WorkflowVisualizer {
	return &WorkflowVisualizer{
		workflow: workflow,
	}
}

// GenerateDOT 生成DOT格式的可视化图形
func (v *WorkflowVisualizer) GenerateDOT() string {
	var builder strings.Builder

	builder.WriteString("digraph Workflow {\n")
	builder.WriteString("  rankdir=TB;\n")
	builder.WriteString("  node [shape=box, style=filled];\n")
	builder.WriteString("  edge [fontsize=10];\n\n")

	// 定义节点样式
	builder.WriteString("  // 节点样式定义\n")
	for _, node := range v.workflow.Nodes {
		nodeStyle := v.getNodeStyle(node.Type)
		builder.WriteString(fmt.Sprintf("  %s [label=\"%s\", %s];\n",
			node.ID, node.Name, nodeStyle))
	}

	// 定义边
	builder.WriteString("\n  // 边定义\n")
	for _, edge := range v.workflow.Edges {
		label := ""
		if edge.Label != "" {
			label = fmt.Sprintf(" [label=\"%s\"]", edge.Label)
		}
		builder.WriteString(fmt.Sprintf("  %s -> %s%s;\n",
			edge.From, edge.To, label))
	}

	builder.WriteString("}\n")
	return builder.String()
}

// GenerateMermaid 生成Mermaid格式的流程图
func (v *WorkflowVisualizer) GenerateMermaid() string {
	var builder strings.Builder

	builder.WriteString("graph TD\n")

	// 添加节点
	for _, node := range v.workflow.Nodes {
		nodeShape := v.getMermaidShape(node.Type)
		builder.WriteString(fmt.Sprintf("  %s[%s]%s\n",
			node.ID, node.Name, nodeShape))
	}

	// 添加边
	for _, edge := range v.workflow.Edges {
		label := ""
		if edge.Label != "" {
			label = " | " + edge.Label
		}
		builder.WriteString(fmt.Sprintf("  %s --> %s%s\n",
			edge.From, edge.To, label))
	}

	return builder.String()
}

// GenerateASCII 生成ASCII艺术格式的工作流图
func (v *WorkflowVisualizer) GenerateASCII() string {
	var builder strings.Builder

	// 简单的ASCII表示
	builder.WriteString(fmt.Sprintf("工作流: %s\n", v.workflow.Name))
	builder.WriteString(strings.Repeat("=", len(v.workflow.Name)+10) + "\n\n")

	// 按位置排序节点
	sortedNodes := make([]NodeDef, len(v.workflow.Nodes))
	copy(sortedNodes, v.workflow.Nodes)

	// 简单的Y坐标排序
	for i := range len(sortedNodes) - 1 {
		for j := i + 1; j < len(sortedNodes); j++ {
			if sortedNodes[i].Position.Y > sortedNodes[j].Position.Y {
				sortedNodes[i], sortedNodes[j] = sortedNodes[j], sortedNodes[i]
			}
		}
	}

	// 生成可视化
	for _, node := range sortedNodes {
		icon := v.getNodeIcon(node.Type)
		builder.WriteString(fmt.Sprintf("%s [%s] %s\n",
			icon, node.Type, node.Name))

		// 显示连接
		for _, edge := range v.workflow.Edges {
			if edge.From == node.ID {
				targetNode := v.findNodeByID(edge.To)
				if targetNode != nil {
					label := ""
					if edge.Label != "" {
						label = fmt.Sprintf(" (%s)", edge.Label)
					}
					builder.WriteString(fmt.Sprintf("    ↓%s\n", label))
					builder.WriteString(fmt.Sprintf("    %s [%s] %s\n",
						v.getNodeIcon(targetNode.Type),
						targetNode.Type, targetNode.Name))
				}
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// getNodeStyle 获取节点的DOT样式
func (v *WorkflowVisualizer) getNodeStyle(nodeType NodeType) string {
	switch nodeType {
	case NodeTypeStart:
		return "fillcolor=green, shape=ellipse"
	case NodeTypeEnd:
		return "fillcolor=red, shape=ellipse"
	case NodeTypeTask:
		return "fillcolor=lightblue, shape=box"
	case NodeTypeCondition:
		return "fillcolor=yellow, shape=diamond"
	case NodeTypeLoop:
		return "fillcolor=orange, shape=hexagon"
	case NodeTypeParallel:
		return "fillcolor=cyan, shape=parallelogram"
	case NodeTypeMerge:
		return "fillcolor=purple, shape=triangle"
	case NodeTypeError:
		return "fillcolor=gray, shape=octagon"
	case NodeTypeTimeout:
		return "fillcolor=brown, shape=octagon"
	default:
		return "fillcolor=lightgray, shape=box"
	}
}

// getMermaidShape 获取节点的Mermaid形状
func (v *WorkflowVisualizer) getMermaidShape(nodeType NodeType) string {
	// Mermaid使用特殊字符来表示形状
	switch nodeType {
	case NodeTypeStart, NodeTypeEnd:
		return "([椭圆形])"
	case NodeTypeCondition:
		return "{菱形}"
	case NodeTypeLoop:
		return "{六边形}"
	default:
		return "[矩形]"
	}
}

// getNodeIcon 获取节点的ASCII图标
func (v *WorkflowVisualizer) getNodeIcon(nodeType NodeType) string {
	switch nodeType {
	case NodeTypeStart:
		return "○"
	case NodeTypeEnd:
		return "●"
	case NodeTypeTask:
		return "□"
	case NodeTypeCondition:
		return "◇"
	case NodeTypeLoop:
		return "⟲"
	case NodeTypeParallel:
		return "‖"
	case NodeTypeMerge:
		return "∴"
	case NodeTypeError:
		return "⚠"
	case NodeTypeTimeout:
		return "⏰"
	default:
		return "◻"
	}
}

// findNodeByID 根据ID查找节点
func (v *WorkflowVisualizer) findNodeByID(id string) *NodeDef {
	for _, node := range v.workflow.Nodes {
		if node.ID == id {
			return &node
		}
	}
	return nil
}

// WorkflowExecutorVisualizer 工作流执行可视化器
type WorkflowExecutorVisualizer struct {
	execution *WorkflowContext
}

// NewWorkflowExecutorVisualizer 创建执行可视化器
func NewWorkflowExecutorVisualizer(execution *WorkflowContext) *WorkflowExecutorVisualizer {
	return &WorkflowExecutorVisualizer{
		execution: execution,
	}
}

// GenerateExecutionState 生成执行状态的可视化
func (v *WorkflowExecutorVisualizer) GenerateExecutionState() string {
	var builder strings.Builder

	builder.WriteString("执行状态可视化\n")
	builder.WriteString("================\n\n")

	builder.WriteString(fmt.Sprintf("工作流ID: %s\n", v.execution.WorkflowID))
	builder.WriteString(fmt.Sprintf("执行ID: %s\n", v.execution.ExecutionID))
	builder.WriteString(fmt.Sprintf("状态: %s\n", v.execution.Status))
	builder.WriteString(fmt.Sprintf("当前节点: %s\n", v.execution.CurrentNode))
	builder.WriteString(fmt.Sprintf("开始时间: %s\n", v.execution.StartTime))

	builder.WriteString("\n节点执行状态:\n")
	builder.WriteString("-------------\n")

	for nodeID, completed := range v.execution.Completed {
		status := "✓ 完成"
		if !completed {
			status = "○ 等待"
		}
		builder.WriteString(fmt.Sprintf("%s: %s\n", nodeID, status))
	}

	for nodeID, err := range v.execution.Failed {
		builder.WriteString(fmt.Sprintf("%s: ✗ 失败 - %s\n", nodeID, err.Error()))
	}

	return builder.String()
}

// GenerateProgressDiagram 生成进度图
func (v *WorkflowExecutorVisualizer) GenerateProgressDiagram() string {
	var builder strings.Builder

	totalNodes := len(v.execution.Completed) + len(v.execution.Failed)
	completedNodes := len(v.execution.Completed)
	progress := 0
	if totalNodes > 0 {
		progress = (completedNodes * 100) / totalNodes
	}

	builder.WriteString("执行进度:\n")
	builder.WriteString(fmt.Sprintf("[%s] %d%%\n", strings.Repeat("█", progress/10), progress))

	return builder.String()
}
