package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/client"
	"quant-system/backend-go/common/strategyparams"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var aShareSymbolPattern = regexp.MustCompile(`^[0-9]{6}$`)

type validatedBacktestRequest struct {
	Symbol, StrategyCode, StartDate, EndDate, ParamsJSON string
	InitialCash, FeeRate, SlippageRate                   float64
	PythonRequest                                        client.BacktestRequest
}

func validateRunRequest(in *backtest.RunBacktestReq) (*validatedBacktestRequest, error) {
	if in == nil || in.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "用户参数错误")
	}
	symbol := strings.TrimSpace(in.GetSymbol())
	if !aShareSymbolPattern.MatchString(symbol) {
		return nil, status.Error(codes.InvalidArgument, "股票代码必须是 6 位数字")
	}
	code := defaultString(strings.TrimSpace(in.GetStrategyCode()), "moving_average_cross")
	allowed := map[string]bool{
		"moving_average_cross": true, "rsi_overbought_oversold": true,
		"ma_cross": true, "macd_cross": true, "rsi_reversal": true,
		"boll_breakout": true, "volume_breakout": true, "trend_score": true,
	}
	if !allowed[code] {
		return nil, status.Errorf(codes.InvalidArgument, "不支持的回测策略: %s", code)
	}
	startDate := defaultDate(strings.TrimSpace(in.GetStartDate()), "2024-01-01")
	endDate := defaultDate(strings.TrimSpace(in.GetEndDate()), "2024-12-31")
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "开始日期必须是 YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "结束日期必须是 YYYY-MM-DD")
	}
	if start.After(end) {
		return nil, status.Error(codes.InvalidArgument, "开始日期不能晚于结束日期")
	}

	initialCash, err := strictNumber(in.GetInitialCash(), 100000, 0, 1e12, false, "初始资金")
	if err != nil {
		return nil, err
	}
	feeRate, err := strictNumber(in.GetFeeRate(), 0.0003, 0, 0.05, true, "佣金率")
	if err != nil {
		return nil, err
	}
	slippageRate, err := strictNumber(in.GetSlippageRate(), 0.0001, 0, 0.10, true, "滑点率")
	if err != nil {
		return nil, err
	}
	adjust := defaultString(strings.ToLower(strings.TrimSpace(in.GetAdjust())), "qfq")
	if adjust != "qfq" {
		return nil, status.Error(codes.InvalidArgument, "当前回测仅支持前复权 qfq")
	}
	minCommission, err := strictNumber(in.GetMinCommission(), 5, 0, 1000, true, "最低佣金")
	if err != nil {
		return nil, err
	}
	stampDutyRate, err := strictNumber(in.GetStampDutyRate(), 0.0005, 0, 0.05, true, "印花税率")
	if err != nil {
		return nil, err
	}
	transferFeeRate, err := strictNumber(in.GetTransferFeeRate(), 0.00001, 0, 0.01, true, "过户费率")
	if err != nil {
		return nil, err
	}
	maxVolumeRatio, err := strictNumber(in.GetMaxVolumeRatio(), 0.10, 0.000001, 1, false, "成交量参与率")
	if err != nil {
		return nil, err
	}
	targetPositionRatio, err := strictNumber(in.GetTargetPositionRatio(), 0.95, 0.000001, 1, false, "目标资金使用率")
	if err != nil {
		return nil, err
	}
	tickSize, err := strictNumber(in.GetTickSize(), 0.01, 0.000001, 10, false, "价格最小变动单位")
	if err != nil {
		return nil, err
	}
	lotSize := in.GetLotSize()
	if lotSize == 0 {
		lotSize = 100
	}
	if lotSize < 1 || lotSize > 10000 {
		return nil, status.Error(codes.InvalidArgument, "买入单位必须在 1 到 10000 股之间")
	}
	paramsJSON := strings.TrimSpace(in.GetParamsJson())
	if paramsJSON == "" {
		paramsJSON = "{}"
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil || params == nil {
		return nil, status.Error(codes.InvalidArgument, "策略参数必须是有效的 JSON 对象")
	}
	if _, err := strategyparams.Validate(pythonStrategyCode(code), params); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	pythonRequest := client.BacktestRequest{
		Symbol: symbol, Strategy: pythonStrategyCode(code), StartDate: startDate, EndDate: endDate,
		InitialCash: initialCash, CommissionRate: feeRate, MinCommission: minCommission,
		StampDutyRate: stampDutyRate, TransferFeeRate: transferFeeRate, SlippageRate: slippageRate,
		MaxVolumeRatio: maxVolumeRatio, TargetPositionRatio: targetPositionRatio, LotSize: int(lotSize), TickSize: tickSize,
		Adjust: adjust, ParamsJSON: paramsJSON,
	}
	return &validatedBacktestRequest{
		Symbol: symbol, StrategyCode: code, StartDate: startDate, EndDate: endDate, ParamsJSON: paramsJSON,
		InitialCash: initialCash, FeeRate: feeRate, SlippageRate: slippageRate, PythonRequest: pythonRequest,
	}, nil
}

func strictNumber(raw string, fallback, minimum, maximum float64, allowZero bool, name string) (float64, error) {
	value := fallback
	var err error
	if strings.TrimSpace(raw) != "" {
		value, err = strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return 0, status.Errorf(codes.InvalidArgument, "%s必须是有效数字", name)
		}
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < minimum || value > maximum || (!allowZero && value == 0) {
		return 0, status.Errorf(codes.InvalidArgument, "%s超出允许范围", name)
	}
	return value, nil
}

func validateBacktestPayload(payload *client.BacktestPayload) error {
	if payload == nil {
		return fmt.Errorf("回测服务未返回结果")
	}
	if err := validateRealBacktest(payload.Source, payload.DataMode); err != nil {
		return err
	}
	if payload.EngineVersion != "backtest-v2.1" {
		return fmt.Errorf("回测引擎版本无效: %s", payload.EngineVersion)
	}
	if payload.ExecutionModel != "NEXT_OPEN_V2" {
		return fmt.Errorf("回测执行模型无效: %s", payload.ExecutionModel)
	}
	if payload.Adjust != "qfq" || payload.DataQuality.Adjust != "qfq" {
		return fmt.Errorf("回测结果缺少可验证的前复权证据")
	}
	if payload.DataQuality.Status != "PASS" && payload.DataQuality.Status != "PARTIAL" {
		return fmt.Errorf("回测数据质量未通过: %s", payload.DataQuality.Status)
	}
	if payload.DataQuality.ValidRows <= 0 || len(payload.EquityCurve) == 0 {
		return fmt.Errorf("回测权益曲线为空")
	}
	if payload.DataQuality.Source != "" && payload.DataQuality.Source != payload.Source {
		return fmt.Errorf("回测数据来源与质量报告不一致")
	}
	for _, key := range []string{"final_equity", "total_return", "annual_return", "max_drawdown", "win_rate", "profit_loss_ratio"} {
		if _, ok := finiteNumber(payload.Summary[key]); !ok {
			return fmt.Errorf("回测指标 %s 不是有限数字", key)
		}
	}

	lastEquity := 0.0
	lastDate := ""
	for index, point := range payload.EquityCurve {
		tradeDate := strings.TrimSpace(fmt.Sprint(point["trade_date"]))
		if _, err := time.Parse("2006-01-02", tradeDate); err != nil || (lastDate != "" && tradeDate <= lastDate) {
			return fmt.Errorf("第 %d 个权益点日期无效或未递增", index+1)
		}
		lastDate = tradeDate
		equity, equityOK := finiteNumber(point["equity"])
		cash, cashOK := finiteNumber(point["cash"])
		positionValue, positionOK := finiteNumber(point["position_value"])
		if !equityOK || !cashOK || !positionOK {
			return fmt.Errorf("第 %d 个权益点包含无效数字", index+1)
		}
		if cash < -0.005 {
			return fmt.Errorf("第 %d 个权益点现金为负", index+1)
		}
		if math.Abs(equity-cash-positionValue) > 0.02 {
			return fmt.Errorf("第 %d 个权益点不满足资产守恒", index+1)
		}
		lastEquity = equity
	}
	finalEquity, _ := finiteNumber(payload.Summary["final_equity"])
	if math.Abs(finalEquity-lastEquity) > 0.02 {
		return fmt.Errorf("最终资产与权益曲线末点不一致")
	}
	for index, trade := range payload.Trades {
		signalDate := strings.TrimSpace(fmt.Sprint(trade["signal_date"]))
		tradeDate := strings.TrimSpace(fmt.Sprint(trade["trade_date"]))
		if signalDate == "" || tradeDate == "" || tradeDate <= signalDate {
			return fmt.Errorf("第 %d 笔成交未满足次日执行约束", index+1)
		}
		price, priceOK := finiteNumber(trade["price"])
		quantity, quantityOK := finiteNumber(trade["quantity"])
		if !priceOK || !quantityOK || price <= 0 || quantity <= 0 {
			return fmt.Errorf("第 %d 笔成交价格或数量无效", index+1)
		}
		fee, feeOK := finiteNumber(trade["fee"])
		amount, amountOK := finiteNumber(trade["amount"])
		if !feeOK || !amountOK || fee < 0 || amount <= 0 || math.Abs(amount-price*quantity) > 0.02 {
			return fmt.Errorf("第 %d 笔成交金额或费用无效", index+1)
		}
	}
	for index, order := range payload.Orders {
		signalDate := strings.TrimSpace(fmt.Sprint(order["signal_date"]))
		if _, err := time.Parse("2006-01-02", signalDate); err != nil {
			return fmt.Errorf("第 %d 笔委托信号日期无效", index+1)
		}
		tradeDate := strings.TrimSpace(fmt.Sprint(order["trade_date"]))
		if tradeDate != "" && tradeDate != "<nil>" && tradeDate <= signalDate {
			return fmt.Errorf("第 %d 笔委托未满足次日执行约束", index+1)
		}
		requested, requestedOK := finiteNumber(order["requested_quantity"])
		filled, filledOK := finiteNumber(order["filled_quantity"])
		cancelled, cancelledOK := finiteNumber(order["cancelled_quantity"])
		if !requestedOK || !filledOK || !cancelledOK || requested < 0 || filled < 0 || cancelled < 0 || filled > requested || cancelled > requested {
			return fmt.Errorf("第 %d 笔委托数量无效", index+1)
		}
	}
	return nil
}

func finiteNumber(value any) (float64, bool) {
	var number float64
	switch typed := value.(type) {
	case float64:
		number = typed
	case float32:
		number = float64(typed)
	case int:
		number = float64(typed)
	case int64:
		number = float64(typed)
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		number = parsed
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return 0, false
		}
		number = parsed
	default:
		return 0, false
	}
	return number, !math.IsNaN(number) && !math.IsInf(number, 0)
}
