package domain

import (
	"context"
	"time"
)

func ExecuteTool(ctx context.Context, tool AgentTool, input ToolInput) ToolOutput {
	start := time.Now()
	output, err := tool.Execute(ctx, input)
	if err != nil {
		return ToolOutput{ToolName: tool.Name(), Error: err.Error(), Success: false, LatencyMs: time.Since(start).Milliseconds()}
	}
	if output.ToolName == "" {
		output.ToolName = tool.Name()
	}
	if output.LatencyMs == 0 {
		output.LatencyMs = time.Since(start).Milliseconds()
	}
	return output
}

// ExecuteTools runs independent research tools concurrently while preserving
// their configured order in the returned slice.
func ExecuteTools(ctx context.Context, tools []AgentTool, input ToolInput) []ToolOutput {
	outputs := make([]ToolOutput, len(tools))
	done := make(chan struct{}, len(tools))
	for i, tool := range tools {
		go func(index int, current AgentTool) {
			defer func() { done <- struct{}{} }()
			outputs[index] = ExecuteTool(ctx, current, input)
		}(i, tool)
	}
	for range tools {
		<-done
	}
	return outputs
}
