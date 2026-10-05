package domain

type ContextBuilder struct{}

func (ContextBuilder) Build(req AnalyzeRequest, outputs map[string]ToolOutput) AgentContext {
	return AgentContext{Request: req, Tools: outputs}
}
