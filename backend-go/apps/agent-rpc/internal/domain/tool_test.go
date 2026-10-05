package domain

import (
	"context"
	"testing"
	"time"
)

type delayedTool struct {
	name  string
	delay time.Duration
}

func (t delayedTool) Name() string        { return t.name }
func (t delayedTool) Description() string { return t.name }
func (t delayedTool) Execute(ctx context.Context, _ ToolInput) (ToolOutput, error) {
	select {
	case <-time.After(t.delay):
		return ToolOutput{ToolName: t.name, Data: t.name, Success: true}, nil
	case <-ctx.Done():
		return ToolOutput{}, ctx.Err()
	}
}

func TestExecuteToolsRunsConcurrentlyAndPreservesOrder(t *testing.T) {
	tools := []AgentTool{
		delayedTool{name: "first", delay: 120 * time.Millisecond},
		delayedTool{name: "second", delay: 120 * time.Millisecond},
		delayedTool{name: "third", delay: 120 * time.Millisecond},
	}
	started := time.Now()
	outputs := ExecuteTools(context.Background(), tools, ToolInput{})
	if elapsed := time.Since(started); elapsed >= 300*time.Millisecond {
		t.Fatalf("tools were not concurrent: elapsed=%s", elapsed)
	}
	for i, want := range []string{"first", "second", "third"} {
		if outputs[i].ToolName != want || !outputs[i].Success {
			t.Fatalf("output %d = %+v, want successful %q", i, outputs[i], want)
		}
	}
}

func TestExecuteToolsHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outputs := ExecuteTools(ctx, []AgentTool{delayedTool{name: "cancelled", delay: time.Second}}, ToolInput{})
	if len(outputs) != 1 || outputs[0].Success || outputs[0].Error == "" {
		t.Fatalf("expected cancelled tool output, got %+v", outputs)
	}
}
