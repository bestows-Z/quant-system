package repository

import (
	"context"
	"database/sql"
)

type RiskRuleRepository struct {
	db *sql.DB
}

func NewRiskRuleRepository(db *sql.DB) *RiskRuleRepository {
	return &RiskRuleRepository{db: db}
}

func (r *RiskRuleRepository) EnsureSystemRules(ctx context.Context) {
	items := []struct {
		code, name, riskType, description, threshold string
	}{
		{"stock_high_volatility", "个股高波动", "STOCK", "当日涨跌幅绝对值过高", `{"change_pct":5}`},
		{"stock_indicator_overheat", "指标过热/过冷", "STOCK", "RSI、BOLL、MACD 等指标异常", `{"rsi_high":70,"rsi_low":30}`},
		{"account_concentration", "账户集中度", "ACCOUNT", "单票或前三大持仓占比过高", `{"single_high":0.5,"top3_high":0.8}`},
		{"account_drawdown", "账户回撤", "ACCOUNT", "模拟账户权益曲线回撤过大", `{"medium":0.1,"high":0.2,"extreme":0.3}`},
		{"position_single_weight", "单票仓位", "POSITION", "单个持仓权重过高", `{"high":0.5,"extreme":0.7}`},
		{"backtest_large_drawdown", "回测大回撤", "BACKTEST", "策略回测最大回撤偏大", `{"medium":0.1,"high":0.2,"extreme":0.3}`},
		{"strategy_high_risk_signal", "策略高风险信号", "STRATEGY", "策略信号风险等级较高", `{"levels":["HIGH","EXTREME"]}`},
	}
	for _, item := range items {
		_, _ = r.db.ExecContext(ctx, `INSERT IGNORE INTO risk_rules (rule_code,rule_name,code,name,risk_type,description,threshold_json,params_json,status,is_system,enabled) VALUES (?,?,?,?,?,?,?,?, 'ENABLED',1,1)`,
			item.code, item.name, item.code, item.name, item.riskType, item.description, item.threshold, item.threshold)
	}
}
