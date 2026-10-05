package server

import (
	"context"

	"quant-system/backend-go/apps/auto-trader-rpc/autotrader"
	"quant-system/backend-go/apps/auto-trader-rpc/internal/svc"
)

type AutoTraderServer struct {
	svcCtx *svc.ServiceContext
	autotrader.UnimplementedAutoTraderServer
}

func NewAutoTraderServer(svcCtx *svc.ServiceContext) *AutoTraderServer {
	return &AutoTraderServer{svcCtx: svcCtx}
}
func reply(json string, err error) (*autotrader.RawReply, error) {
	if err != nil {
		return nil, err
	}
	return &autotrader.RawReply{Json: json}, nil
}
func (s *AutoTraderServer) CreateAccount(ctx context.Context, in *autotrader.CreateAccountReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.CreateAccount(ctx, in))
}
func (s *AutoTraderServer) ListAccounts(ctx context.Context, in *autotrader.UserReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.ListAccounts(ctx, in.GetUserId()))
}
func (s *AutoTraderServer) ListArchivedAccounts(ctx context.Context, in *autotrader.UserReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.ListArchivedAccounts(ctx, in.GetUserId()))
}
func (s *AutoTraderServer) UpdateAccount(ctx context.Context, in *autotrader.UpdateAccountReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.UpdateAccount(ctx, in))
}
func (s *AutoTraderServer) ArchiveAccount(ctx context.Context, in *autotrader.AccountReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.ArchiveAccount(ctx, in.GetUserId(), in.GetAccountId()))
}
func (s *AutoTraderServer) GenerateStrategy(ctx context.Context, in *autotrader.GenerateStrategyReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.GenerateStrategy(ctx, in))
}
func (s *AutoTraderServer) ListStrategies(ctx context.Context, in *autotrader.AccountReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.ListStrategies(ctx, in.GetUserId(), in.GetAccountId()))
}
func (s *AutoTraderServer) GetStrategy(ctx context.Context, in *autotrader.StrategyReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.GetStrategy(ctx, in.GetUserId(), in.GetStrategyId()))
}
func (s *AutoTraderServer) EvaluateStrategy(ctx context.Context, in *autotrader.StrategyReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.EvaluateStrategy(ctx, in.GetUserId(), in.GetStrategyId()))
}
func (s *AutoTraderServer) PromoteStrategy(ctx context.Context, in *autotrader.StrategyReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.PromoteStrategy(ctx, in.GetUserId(), in.GetStrategyId()))
}
func (s *AutoTraderServer) Start(ctx context.Context, in *autotrader.AccountReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.Start(ctx, in.GetUserId(), in.GetAccountId()))
}
func (s *AutoTraderServer) Pause(ctx context.Context, in *autotrader.AccountReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.Pause(ctx, in.GetUserId(), in.GetAccountId()))
}
func (s *AutoTraderServer) ListRuns(ctx context.Context, in *autotrader.AccountReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.ListRuns(ctx, in.GetUserId(), in.GetAccountId()))
}
func (s *AutoTraderServer) ListDecisions(ctx context.Context, in *autotrader.AccountReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.ListDecisions(ctx, in.GetUserId(), in.GetAccountId()))
}
func (s *AutoTraderServer) ListDeliveryStatements(ctx context.Context, in *autotrader.DeliveryStatementReq) (*autotrader.RawReply, error) {
	return reply(s.svcCtx.Service.ListDeliveryStatements(ctx, in))
}
