from fastapi.testclient import TestClient

from app.main import app


client = TestClient(app)


def test_health_response_contract():
    response = client.get("/health")
    payload = response.json()

    assert response.status_code == 200
    assert payload["code"] == 0
    assert payload["data"]["status"] == "ok"


def test_kline_response_contract():
    response = client.get("/api/kline/000001?start=2025-01-01&end=2025-03-31")
    payload = response.json()

    assert response.status_code == 200
    assert payload["code"] == 0
    assert payload["data"]["symbol"] == "000001"
    assert payload["data"]["data_mode"] == "mock"
    assert payload["data"]["items"]

