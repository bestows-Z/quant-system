from unittest.mock import Mock

from app.collectors.research import ResearchDataCollector, _calculate_market_breadth
from app.core.errors import DataSourceError
from app.services.market_service import MarketService


def test_market_indices_parses_six_tiers():
    collector = ResearchDataCollector()
    lines = []
    for code, name in [("sh000001", "上证指数"), ("sz399001", "深证成指"), ("sh000688", "科创50"),
                       ("sz399006", "创业板指"), ("sh000300", "沪深300"), ("bj899050", "北证50")]:
        fields = [""] * 38
        fields[1], fields[3], fields[4], fields[30], fields[32], fields[33], fields[34], fields[36], fields[37] = (
            name, "3200", "3180", "20260717150000", "0.63", "3220", "3170", "1000", "2000"
        )
        lines.append(f'v_{code}="{"~".join(fields)}";')
    response = Mock(content="\n".join(lines).encode("gbk"))
    response.raise_for_status = Mock()
    collector.session.get = Mock(return_value=response)

    result = collector.market_indices()

    assert result["count"] == 6
    assert result["items"][2]["board"] == "科创板"
    assert result["items"][0]["change_pct"] == 0.0063


def test_market_breadth_uses_full_snapshot_amount_and_distribution():
    rows = [
        {"f2": 10, "f3": 8, "f6": 1_000_000},
        {"f2": 11, "f3": -4, "f6": 2_000_000},
        {"f2": 12, "f3": 0, "f6": 3_000_000},
        {"f2": "-", "f3": "-", "f6": "-"},
    ]
    result = _calculate_market_breadth(rows, 4)
    overview = result["overview"]
    assert overview["total"] == 3
    assert overview["total_amount"] == 6_000_000
    assert overview["scope"] == "all_a_shares"
    assert sum(item["count"] for item in result["distribution"]) == 3
    assert next(item for item in result["distribution"] if item["label"] == "平盘")["count"] == 1


def test_market_breadth_falls_back_to_tencent():
    collector = ResearchDataCollector()
    collector._eastmoney_market_breadth = Mock(side_effect=DataSourceError("eastmoney unavailable", code=503))
    collector._tencent_market_breadth = Mock(return_value={"overview": {"total": 5200}, "source": "tencent-batch"})

    result = collector.market_breadth()

    assert result["source"] == "tencent-batch"


def test_company_profile_maps_research_fields():
    collector = ResearchDataCollector()
    collector._get_json = Mock(return_value={"result": {"data": [{
        "SECURITY_CODE": "600176", "SECURITY_NAME_ABBR": "中国巨石", "ORG_NAME": "中国巨石股份有限公司",
        "EM2016": "基础化工-玻纤", "MAIN_BUSINESS": "玻璃纤维及制品", "ORG_PROFILE": " 公司简介 ",
        "LISTING_DATE": "1999-04-22 00:00:00", "ORG_WEB": "www.jushi.com",
    }]}})

    result = collector.company_profile("600176")

    assert result["company_name"] == "中国巨石股份有限公司"
    assert result["listing_date"] == "1999-04-22"
    assert result["main_business"] == "玻璃纤维及制品"


def test_money_flow_keeps_real_source_categories():
    collector = ResearchDataCollector()
    collector._get_json = Mock(return_value={"data": {"name": "中国巨石", "klines": [
        "2026-07-17,-275052272,233376096,41676176,-36098192,-238954080,-3.34,2.84,0.51,-0.44,-2.91,43.80,-8.20"
    ]}})

    result = collector.money_flow("600176")

    row = result["items"][0]
    assert row["main_net_inflow"] == -275052272
    assert row["retail_net_inflow"] == 233376096
    assert row["main_net_ratio"] == -0.0334


def test_money_flow_failure_returns_truthful_optional_status():
    service = object.__new__(MarketService)
    service.research = Mock()
    service.research.money_flow.side_effect = DataSourceError("proxy failed", code=503)

    result = service.money_flow("600176", 20)

    assert result["status"] == "SOURCE_UNAVAILABLE"
    assert result["items"] == []
    assert result["data_mode"] == "real"


def test_money_flow_falls_back_to_tushare_and_converts_wan_to_yuan():
    collector = ResearchDataCollector(tushare_token="test-token")
    collector._eastmoney_money_flow = Mock(side_effect=DataSourceError("eastmoney unavailable", code=503))
    response = Mock()
    response.raise_for_status = Mock()
    response.json.return_value = {"code": 0, "data": {"fields": [
        "ts_code", "trade_date", "buy_sm_amount", "sell_sm_amount", "buy_md_amount", "sell_md_amount",
        "buy_lg_amount", "sell_lg_amount", "buy_elg_amount", "sell_elg_amount", "net_mf_amount",
    ], "items": [["600176.SH", "20260717", 100, 80, 50, 60, 200, 150, 300, 180, 180]]}}
    collector.session.post = Mock(return_value=response)

    result = collector.money_flow("600176")

    assert result["source"] == "tushare-pro"
    assert result["items"][0]["trade_date"] == "2026-07-17"
    assert result["items"][0]["retail_net_inflow"] == 200000
    assert result["items"][0]["main_net_inflow"] == 1700000


def test_dragon_tiger_maps_latest_records():
    collector = ResearchDataCollector()
    collector._get_json = Mock(return_value={"result": {"data": [{
        "TRADE_DATE": "2026-07-17 00:00:00", "SECURITY_CODE": "000566", "SECURITY_NAME_ABBR": "海南海药",
        "EXPLAIN": "机构卖出", "CLOSE_PRICE": 5.1, "CHANGE_RATE": -10.0529, "BILLBOARD_DEAL_AMT": 347928955.27,
    }]}})

    result = collector.dragon_tiger()

    assert result["count"] == 1
    assert result["items"][0]["trade_date"] == "2026-07-17"
    assert result["items"][0]["change_percent"] == -0.100529


def test_dragon_tiger_seats_merges_buy_and_sell_rankings():
    collector = ResearchDataCollector()
    collector._get_json = Mock(side_effect=[
        {"result": {"data": [{"OPERATEDEPT_CODE": "A", "OPERATEDEPT_NAME": "机构专用", "BUY": 120, "SELL": 20, "NET": 100}]}},
        {"result": {"data": [{"OPERATEDEPT_CODE": "A", "OPERATEDEPT_NAME": "机构专用", "BUY": 120, "SELL": 20, "NET": 100},
                              {"OPERATEDEPT_CODE": "B", "OPERATEDEPT_NAME": "某证券营业部", "BUY": 10, "SELL": 80, "NET": -70}]}},
    ])

    result = collector.dragon_tiger_seats("000566", "2026-07-17")

    assert result["count"] == 2
    assert result["items"][0]["seat_type"] == "INSTITUTION"
    assert result["items"][0]["buy_rank"] == 1
    assert result["items"][0]["sell_rank"] == 1
    assert result["items"][1]["sell_amount"] == 80
