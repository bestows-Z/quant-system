package errorx

const (
	CodeOK                       = 0
	CodeInvalidArgument          = 10001
	CodeUnauthorized             = 10002
	CodeForbidden                = 10003
	CodeNotFound                 = 10004
	CodeTooManyRequests          = 10005
	CodeInternal                 = 10099
	AgentTaskNotFound            = 30001
	AgentTaskFailed              = 30002
	AgentInvalidQuestion         = 30003
	AgentInvalidSymbol           = 30004
	AgentToolCallFailed          = 30005
	AgentLLMFailed               = 30006
	AgentReportNotFound          = 30007
	AgentPermissionDenied        = 30008
	StrategyNotFound             = 31001
	StrategyRunFailed            = 31002
	StrategySignalNotFound       = 31003
	StrategyInvalidParam         = 31004
	StrategyQuantPythonFailed    = 31005
	BacktestTaskNotFound         = 32001
	BacktestRunFailed            = 32002
	BacktestResultNotFound       = 32003
	BacktestInvalidParam         = 32004
	BacktestQuantPythonFailed    = 32005
	RiskRecordNotFound           = 33001
	RiskAnalyzeFailed            = 33002
	RiskInvalidParam             = 33003
	RiskSourceUnavailable        = 33004
	RiskPermissionDenied         = 33005
	RiskRuleNotFound             = 33006
	TaskNotFound                 = 34001
	TaskCreateFailed             = 34002
	TaskUpdateFailed             = 34003
	TaskCancelFailed             = 34004
	TaskPermissionDenied         = 34005
	NotificationNotFound         = 35001
	NotificationCreateFailed     = 35002
	NotificationPermissionDenied = 35003
	NotificationEventDuplicated  = 35004
	WsUnauthorized               = 36001
	WsInvalidMessage             = 36002
	WsSubscribeFailed            = 36003
	WsPushFailed                 = 36004
)

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func New(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}
