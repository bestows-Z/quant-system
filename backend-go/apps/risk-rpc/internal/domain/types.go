package domain

const Disclaimer = "风控结果仅供研究，不构成投资建议"

const (
	RiskTypeStock    = "STOCK"
	RiskTypeAccount  = "ACCOUNT"
	RiskTypePosition = "POSITION"
	RiskTypeStrategy = "STRATEGY"
	RiskTypeBacktest = "BACKTEST"
	RiskTypeSystem   = "SYSTEM"

	RiskLevelLow     = "LOW"
	RiskLevelMedium  = "MEDIUM"
	RiskLevelHigh    = "HIGH"
	RiskLevelExtreme = "EXTREME"
)
