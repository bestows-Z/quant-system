package autotraderservice

import (
	"context"

	"quant-system/backend-go/apps/auto-trader-rpc/autotrader"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	UserReq              = autotrader.UserReq
	AccountReq           = autotrader.AccountReq
	StrategyReq          = autotrader.StrategyReq
	CreateAccountReq     = autotrader.CreateAccountReq
	GenerateStrategyReq  = autotrader.GenerateStrategyReq
	UpdateAccountReq     = autotrader.UpdateAccountReq
	DeliveryStatementReq = autotrader.DeliveryStatementReq
	RawReply             = autotrader.RawReply
	AutoTrader           interface {
		CreateAccount(context.Context, *CreateAccountReq, ...grpc.CallOption) (*RawReply, error)
		ListAccounts(context.Context, *UserReq, ...grpc.CallOption) (*RawReply, error)
		ListArchivedAccounts(context.Context, *UserReq, ...grpc.CallOption) (*RawReply, error)
		UpdateAccount(context.Context, *UpdateAccountReq, ...grpc.CallOption) (*RawReply, error)
		ArchiveAccount(context.Context, *AccountReq, ...grpc.CallOption) (*RawReply, error)
		GenerateStrategy(context.Context, *GenerateStrategyReq, ...grpc.CallOption) (*RawReply, error)
		ListStrategies(context.Context, *AccountReq, ...grpc.CallOption) (*RawReply, error)
		GetStrategy(context.Context, *StrategyReq, ...grpc.CallOption) (*RawReply, error)
		EvaluateStrategy(context.Context, *StrategyReq, ...grpc.CallOption) (*RawReply, error)
		PromoteStrategy(context.Context, *StrategyReq, ...grpc.CallOption) (*RawReply, error)
		Start(context.Context, *AccountReq, ...grpc.CallOption) (*RawReply, error)
		Pause(context.Context, *AccountReq, ...grpc.CallOption) (*RawReply, error)
		ListRuns(context.Context, *AccountReq, ...grpc.CallOption) (*RawReply, error)
		ListDecisions(context.Context, *AccountReq, ...grpc.CallOption) (*RawReply, error)
		ListDeliveryStatements(context.Context, *DeliveryStatementReq, ...grpc.CallOption) (*RawReply, error)
	}
	defaultAutoTrader struct{ cli zrpc.Client }
)

func NewAutoTrader(cli zrpc.Client) AutoTrader { return &defaultAutoTrader{cli: cli} }
func (m *defaultAutoTrader) client() autotrader.AutoTraderClient {
	return autotrader.NewAutoTraderClient(m.cli.Conn())
}
func (m *defaultAutoTrader) CreateAccount(ctx context.Context, in *CreateAccountReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().CreateAccount(ctx, in, opts...)
}
func (m *defaultAutoTrader) ListAccounts(ctx context.Context, in *UserReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().ListAccounts(ctx, in, opts...)
}
func (m *defaultAutoTrader) ListArchivedAccounts(ctx context.Context, in *UserReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().ListArchivedAccounts(ctx, in, opts...)
}
func (m *defaultAutoTrader) UpdateAccount(ctx context.Context, in *UpdateAccountReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().UpdateAccount(ctx, in, opts...)
}
func (m *defaultAutoTrader) ArchiveAccount(ctx context.Context, in *AccountReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().ArchiveAccount(ctx, in, opts...)
}
func (m *defaultAutoTrader) GenerateStrategy(ctx context.Context, in *GenerateStrategyReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().GenerateStrategy(ctx, in, opts...)
}
func (m *defaultAutoTrader) ListStrategies(ctx context.Context, in *AccountReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().ListStrategies(ctx, in, opts...)
}
func (m *defaultAutoTrader) GetStrategy(ctx context.Context, in *StrategyReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().GetStrategy(ctx, in, opts...)
}
func (m *defaultAutoTrader) EvaluateStrategy(ctx context.Context, in *StrategyReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().EvaluateStrategy(ctx, in, opts...)
}
func (m *defaultAutoTrader) PromoteStrategy(ctx context.Context, in *StrategyReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().PromoteStrategy(ctx, in, opts...)
}
func (m *defaultAutoTrader) Start(ctx context.Context, in *AccountReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().Start(ctx, in, opts...)
}
func (m *defaultAutoTrader) Pause(ctx context.Context, in *AccountReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().Pause(ctx, in, opts...)
}
func (m *defaultAutoTrader) ListRuns(ctx context.Context, in *AccountReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().ListRuns(ctx, in, opts...)
}
func (m *defaultAutoTrader) ListDecisions(ctx context.Context, in *AccountReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().ListDecisions(ctx, in, opts...)
}
func (m *defaultAutoTrader) ListDeliveryStatements(ctx context.Context, in *DeliveryStatementReq, opts ...grpc.CallOption) (*RawReply, error) {
	return m.client().ListDeliveryStatements(ctx, in, opts...)
}
