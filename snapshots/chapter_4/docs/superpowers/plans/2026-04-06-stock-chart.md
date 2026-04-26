# Stock Chart Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an interactive stock price chart page where users pick a portfolio holding and view candlestick/line chart with date range presets.

**Architecture:** Three-layer change — Python data service gets a `/price/{ticker}/history` endpoint, Go backend proxies it through `GET /api/prices/{ticker}/history` with in-memory caching, and a new Next.js `/charts` page renders TradingView Lightweight Charts.

**Tech Stack:** Python/FastAPI, Go/Chi, Next.js 16 App Router, TradingView `lightweight-charts`

**Spec:** `docs/superpowers/specs/2026-04-06-stock-chart-design.md`

---

## File Map

### Python Data Service (`data-service/`)

| Action | File                              | Responsibility                                                           |
| ------ | --------------------------------- | ------------------------------------------------------------------------ |
| Modify | `src/models/price.py`             | Add `HistoricalPricePoint` and `HistoricalPriceResponse` Pydantic models |
| Modify | `src/services/market_data.py`     | Add `get_historical_prices()` method using yfinance `.history()`         |
| Modify | `src/routers/prices.py`           | Add `GET /price/{ticker}/history` endpoint                               |
| Modify | `pyproject.toml`                  | Add `tenacity` dependency                                                |
| Create | `tests/test_historical_prices.py` | Tests for the historical prices endpoint                                 |

### Go Backend (`backend/`)

| Action | File                                           | Responsibility                                              |
| ------ | ---------------------------------------------- | ----------------------------------------------------------- |
| Create | `internal/model/historical_price.go`           | `HistoricalPrice` and `HistoricalPriceResponse` structs     |
| Modify | `internal/client/data_service.go`              | Add `GetPriceHistory()` method                              |
| Create | `internal/service/history_cache.go`            | In-memory TTL cache for historical price responses          |
| Modify | `internal/service/portfolio.go`                | Add `GetPriceHistory()` method using cache + client         |
| Modify | `internal/handler/portfolio.go`                | Add `GetPriceHistory()` handler with query param validation |
| Modify | `cmd/server/main.go`                           | Wire new route `GET /api/prices/{ticker}/history`           |
| Create | `internal/client/data_service_history_test.go` | Client tests for `GetPriceHistory`                          |
| Create | `internal/handler/portfolio_history_test.go`   | Handler tests for `GetPriceHistory`                         |
| Create | `internal/service/history_cache_test.go`       | Cache TTL and concurrency tests                             |

### Frontend (`frontend/`)

| Action | File                         | Responsibility                                                          |
| ------ | ---------------------------- | ----------------------------------------------------------------------- |
| Modify | `lib/api.ts`                 | Add `HistoricalPriceResponse` type and `getHistoricalPrices()` function |
| Create | `lib/chart-utils.ts`         | `determineChartMode()` pure function                                    |
| Create | `lib/chart-utils.test.ts`    | Vitest tests for `determineChartMode()`                                 |
| Create | `components/stock-chart.tsx` | Client Component wrapping lightweight-charts                            |
| Create | `app/charts/page.tsx`        | Server Component page for the chart                                     |
| Modify | `app/layout.tsx`             | Add "Charts" link to navbar                                             |

---

## Task 1: Python — Pydantic Models for Historical Prices

**Files:**

- Modify: `data-service/src/models/price.py`

- [ ] **Step 1: Add models to `price.py`**

Append to the existing file after the `HealthResponse` class:

```python
class HistoricalPricePoint(BaseModel):
    date: str
    open: float | None = None
    high: float | None = None
    low: float | None = None
    close: float | None = None
    adj_close: float | None = None
    volume: float | None = None


class HistoricalPriceResponse(BaseModel):
    ticker: str
    currency: str = "USD"
    interval: str = "daily"
    prices: list[HistoricalPricePoint]
```

- [ ] **Step 2: Verify the data service still starts**

Run: `cd data-service && uv run python -c "from src.models.price import HistoricalPricePoint, HistoricalPriceResponse; print('OK')"`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add data-service/src/models/price.py
git commit -m "feat(data-service): add historical price Pydantic models"
```

---

## Task 2: Python — Add `tenacity` Dependency

**Files:**

- Modify: `data-service/pyproject.toml`

- [ ] **Step 1: Add `tenacity` to dependencies**

In `pyproject.toml`, add `"tenacity>=8.2.0"` to the `dependencies` list:

```toml
dependencies = [
    "fastapi>=0.115.0",
    "uvicorn[standard]>=0.34.0",
    "yfinance>=0.2.50",
    "pydantic>=2.10.0",
    "tenacity>=8.2.0",
]
```

- [ ] **Step 2: Install**

Run: `cd data-service && uv sync`
Expected: tenacity installs without errors.

- [ ] **Step 3: Commit**

```bash
git add data-service/pyproject.toml data-service/uv.lock
git commit -m "chore(data-service): add tenacity dependency for retry logic"
```

---

## Task 3: Python — `get_historical_prices()` Service Method

**Files:**

- Modify: `data-service/src/services/market_data.py`
- Create: `data-service/tests/test_historical_prices.py`

- [ ] **Step 1: Write the failing tests**

Create `data-service/tests/test_historical_prices.py`:

```python
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, patch

import pandas as pd
from httpx import ASGITransport, AsyncClient

from src.main import app


def _make_history_df(rows: list[dict]) -> pd.DataFrame:
    """Build a DataFrame that looks like yfinance .history() output."""
    df = pd.DataFrame(rows)
    df.index = pd.to_datetime(df.pop("Date"))
    df.index.name = "Date"
    return df


async def test_historical_prices_returns_ohlcv():
    df = _make_history_df([
        {"Date": "2025-04-07", "Open": 150.0, "High": 152.0, "Low": 149.0, "Close": 151.5, "Volume": 48000000},
        {"Date": "2025-04-08", "Open": 151.0, "High": 153.0, "Low": 150.0, "Close": 152.0, "Volume": 50000000},
    ])
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = df
    mock_ticker.info = {"currency": "USD"}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL/history?start=2025-04-07&end=2025-04-09")

    assert response.status_code == 200
    data = response.json()
    assert data["ticker"] == "AAPL"
    assert data["currency"] == "USD"
    assert data["interval"] == "daily"
    assert len(data["prices"]) == 2
    assert data["prices"][0]["open"] == 150.0
    assert data["prices"][0]["volume"] == 48000000


async def test_historical_prices_invalid_ticker_returns_404():
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = pd.DataFrame()
    mock_ticker.info = {}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/INVALID/history?start=2025-01-01&end=2025-12-31")

    assert response.status_code == 404
    assert "retryable" in response.json()
    assert response.json()["retryable"] is False


async def test_historical_prices_resamples_weekly_for_long_range():
    """Ranges > 5 years should be resampled to weekly."""
    dates = pd.bdate_range("2018-01-01", "2025-04-07")
    rows = [{"Date": d.strftime("%Y-%m-%d"), "Open": 100.0, "High": 101.0, "Low": 99.0, "Close": 100.5, "Volume": 1000000} for d in dates]
    df = _make_history_df(rows)
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = df
    mock_ticker.info = {"currency": "USD"}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL/history?start=2018-01-01&end=2025-04-07")

    assert response.status_code == 200
    data = response.json()
    assert data["interval"] == "weekly"
    # Weekly resampling should produce far fewer rows than daily
    assert len(data["prices"]) < len(rows)


async def test_historical_prices_null_ohlc_preserved():
    """Rows with NaN OHLC should become null in JSON."""
    df = _make_history_df([
        {"Date": "2025-04-07", "Open": float("nan"), "High": float("nan"), "Low": float("nan"), "Close": 45.2, "Volume": float("nan")},
    ])
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = df
    mock_ticker.info = {"currency": "USD"}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL/history?start=2025-04-07&end=2025-04-08")

    assert response.status_code == 200
    data = response.json()
    assert data["prices"][0]["open"] is None
    assert data["prices"][0]["close"] == 45.2
    assert data["prices"][0]["volume"] is None
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd data-service && uv run pytest tests/test_historical_prices.py -v`
Expected: FAIL — `get_historical_prices` method and `/price/{ticker}/history` route don't exist yet.

- [ ] **Step 3: Implement `get_historical_prices()` in `market_data.py`**

Add these imports at the top of `data-service/src/services/market_data.py`:

```python
import math

import pandas as pd
from tenacity import retry, stop_after_attempt, wait_exponential
```

Add this method to the `MarketDataService` class after `get_price()`:

```python
    async def get_historical_prices(
        self, ticker: str, start_date: str, end_date: str
    ) -> dict:
        """Get historical OHLCV data for a ticker.

        Args:
            ticker: Stock symbol (e.g. "AAPL").
            start_date: Start date as YYYY-MM-DD.
            end_date: End date as YYYY-MM-DD.

        Returns:
            Dict with ticker, currency, interval, and prices list.

        Raises:
            ValueError: If ticker is not found or has no data for the range.
        """
        try:
            stock = yf.Ticker(ticker)
            df = self._fetch_history(stock, start_date, end_date)
        except ValueError:
            raise
        except Exception as e:
            logger.exception("Failed to fetch history for %s", ticker)
            raise ValueError(f"No data available for {ticker}") from e

        if df.empty:
            raise ValueError(f"No data available for {ticker}")

        currency = "USD"
        try:
            currency = stock.info.get("currency", "USD")
        except Exception:
            pass

        # Determine resampling interval based on date range length
        start = pd.Timestamp(start_date)
        end = pd.Timestamp(end_date)
        span_years = (end - start).days / 365.25
        interval = "daily"

        if span_years > 15:
            df = self._resample(df, "ME")
            interval = "monthly"
        elif span_years > 5:
            df = self._resample(df, "W")
            interval = "weekly"

        prices = []
        for date, row in df.iterrows():
            point = {
                "date": date.strftime("%Y-%m-%d"),
                "open": self._nan_to_none(row.get("Open")),
                "high": self._nan_to_none(row.get("High")),
                "low": self._nan_to_none(row.get("Low")),
                "close": self._nan_to_none(row.get("Close")),
                "adj_close": self._nan_to_none(row.get("Close")),
                "volume": self._nan_to_none(row.get("Volume")),
            }
            prices.append(point)

        return {
            "ticker": ticker.upper(),
            "currency": currency,
            "interval": interval,
            "prices": prices,
        }

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=10),
        reraise=True,
    )
    def _fetch_history(self, stock: yf.Ticker, start: str, end: str) -> pd.DataFrame:
        """Fetch history with retry logic."""
        return stock.history(start=start, end=end)

    @staticmethod
    def _resample(df: pd.DataFrame, rule: str) -> pd.DataFrame:
        """Resample OHLCV data to a coarser interval."""
        return df.resample(rule).agg({
            "Open": "first",
            "High": "max",
            "Low": "min",
            "Close": "last",
            "Volume": "sum",
        }).dropna(how="all")

    @staticmethod
    def _nan_to_none(value) -> float | None:
        """Convert NaN/None to None for JSON serialization."""
        if value is None:
            return None
        try:
            if math.isnan(value):
                return None
        except (TypeError, ValueError):
            return None
        return float(value)
```

- [ ] **Step 4: Add the route to `prices.py`**

Add import at top of `data-service/src/routers/prices.py`:

```python
from src.models.price import HistoricalPriceResponse, PriceResponse
```

(Replace the existing `from src.models.price import PriceResponse` line.)

Add this endpoint after the existing `get_price` function:

```python
@router.get("/price/{ticker}/history", response_model=HistoricalPriceResponse)
async def get_historical_prices(
    ticker: str, start: str, end: str
) -> HistoricalPriceResponse:
    """Get historical OHLCV data for a stock ticker."""
    try:
        result = await market_data_service.get_historical_prices(
            ticker.upper(), start, end
        )
        return HistoricalPriceResponse(**result)
    except ValueError as e:
        error_msg = str(e)
        is_retryable = "unavailable" in error_msg.lower() or "busy" in error_msg.lower()
        raise HTTPException(
            status_code=503 if is_retryable else 404,
            detail={"error": error_msg, "retryable": is_retryable},
        )
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd data-service && uv run pytest tests/test_historical_prices.py -v`
Expected: All 4 tests PASS.

- [ ] **Step 6: Run all existing tests to ensure no regressions**

Run: `cd data-service && uv run pytest tests/ -v`
Expected: All tests PASS (existing + new).

- [ ] **Step 7: Commit**

```bash
git add data-service/src/services/market_data.py data-service/src/routers/prices.py data-service/src/models/price.py data-service/tests/test_historical_prices.py
git commit -m "feat(data-service): add historical prices endpoint with resampling and retry"
```

---

## Task 4: Go — Historical Price Model

**Files:**

- Create: `backend/internal/model/historical_price.go`

- [ ] **Step 1: Create the model file**

Create `backend/internal/model/historical_price.go`:

```go
package model

type HistoricalPrice struct {
	Date     string   `json:"date"`
	Open     *float64 `json:"open"`
	High     *float64 `json:"high"`
	Low      *float64 `json:"low"`
	Close    *float64 `json:"close"`
	AdjClose *float64 `json:"adj_close"`
	Volume   *float64 `json:"volume"`
}

type HistoricalPriceResponse struct {
	Ticker   string            `json:"ticker"`
	Currency string            `json:"currency"`
	Interval string            `json:"interval"`
	Prices   []HistoricalPrice `json:"prices"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd backend && go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/model/historical_price.go
git commit -m "feat(backend): add historical price model structs"
```

---

## Task 5: Go — Data Service Client `GetPriceHistory()`

**Files:**

- Modify: `backend/internal/client/data_service.go`
- Create: `backend/internal/client/data_service_history_test.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/client/data_service_history_test.go`:

```go
package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
)

func TestDataServiceClient_GetPriceHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/price/AAPL/history" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("start") != "2025-01-01" {
			t.Errorf("unexpected start: %s", r.URL.Query().Get("start"))
		}
		if r.URL.Query().Get("end") != "2025-12-31" {
			t.Errorf("unexpected end: %s", r.URL.Query().Get("end"))
		}
		w.Header().Set("Content-Type", "application/json")
		open := 150.0
		high := 152.0
		low := 149.0
		close := 151.5
		vol := 48000000.0
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ticker":   "AAPL",
			"currency": "USD",
			"interval": "daily",
			"prices": []map[string]interface{}{
				{"date": "2025-01-02", "open": open, "high": high, "low": low, "close": close, "adj_close": close, "volume": vol},
			},
		})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	resp, err := c.GetPriceHistory("AAPL", "2025-01-01", "2025-12-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", resp.Ticker)
	}
	if len(resp.Prices) != 1 {
		t.Fatalf("expected 1 price, got %d", len(resp.Prices))
	}
	if *resp.Prices[0].Open != 150.0 {
		t.Errorf("expected open 150.0, got %f", *resp.Prices[0].Open)
	}
}

func TestDataServiceClient_GetPriceHistory_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail": map[string]interface{}{"error": "No data available", "retryable": false},
		})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	_, err := c.GetPriceHistory("INVALID", "2025-01-01", "2025-12-31")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDataServiceClient_GetPriceHistory_503(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail": map[string]interface{}{"error": "Provider busy", "retryable": true},
		})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	_, err := c.GetPriceHistory("AAPL", "2025-01-01", "2025-12-31")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/client/ -run TestDataServiceClient_GetPriceHistory -v`
Expected: FAIL — `GetPriceHistory` method does not exist.

- [ ] **Step 3: Implement `GetPriceHistory` in `data_service.go`**

Add this method after `GetPrice()` in `backend/internal/client/data_service.go`:

```go
func (c *DataServiceClient) GetPriceHistory(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
	url := fmt.Sprintf("%s/price/%s/history?start=%s&end=%s", c.baseURL, ticker, start, end)

	// Use a longer timeout for potentially large historical data requests
	historyClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := historyClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch price history: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("data service returned %d for ticker %s history", resp.StatusCode, ticker)
	}

	var result model.HistoricalPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode history response: %w", err)
	}

	return &result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/client/ -v`
Expected: All client tests PASS (existing + new).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/client/data_service.go backend/internal/client/data_service_history_test.go
git commit -m "feat(backend): add GetPriceHistory client method"
```

---

## Task 6: Go — In-Memory History Cache

**Files:**

- Create: `backend/internal/service/history_cache.go`
- Create: `backend/internal/service/history_cache_test.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/service/history_cache_test.go`:

```go
package service

import (
	"testing"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

func TestHistoryCache_GetMiss(t *testing.T) {
	cache := NewHistoryCache(15 * time.Minute)
	_, ok := cache.Get("AAPL", "2025-01-01", "2025-12-31")
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestHistoryCache_SetAndGet(t *testing.T) {
	cache := NewHistoryCache(15 * time.Minute)
	resp := &model.HistoricalPriceResponse{
		Ticker:   "AAPL",
		Currency: "USD",
		Interval: "daily",
		Prices:   []model.HistoricalPrice{{Date: "2025-01-02"}},
	}

	cache.Set("AAPL", "2025-01-01", "2025-12-31", resp)

	got, ok := cache.Get("AAPL", "2025-01-01", "2025-12-31")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", got.Ticker)
	}
	if len(got.Prices) != 1 {
		t.Errorf("expected 1 price, got %d", len(got.Prices))
	}
}

func TestHistoryCache_Expiry(t *testing.T) {
	cache := NewHistoryCache(1 * time.Millisecond)
	resp := &model.HistoricalPriceResponse{Ticker: "AAPL"}
	cache.Set("AAPL", "2025-01-01", "2025-12-31", resp)

	time.Sleep(5 * time.Millisecond)

	_, ok := cache.Get("AAPL", "2025-01-01", "2025-12-31")
	if ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestHistoryCache_DifferentKeysIndependent(t *testing.T) {
	cache := NewHistoryCache(15 * time.Minute)
	resp1 := &model.HistoricalPriceResponse{Ticker: "AAPL"}
	resp2 := &model.HistoricalPriceResponse{Ticker: "GOOG"}

	cache.Set("AAPL", "2025-01-01", "2025-12-31", resp1)
	cache.Set("GOOG", "2025-01-01", "2025-12-31", resp2)

	got, ok := cache.Get("AAPL", "2025-01-01", "2025-12-31")
	if !ok || got.Ticker != "AAPL" {
		t.Fatal("expected AAPL cache hit")
	}
	got, ok = cache.Get("GOOG", "2025-01-01", "2025-12-31")
	if !ok || got.Ticker != "GOOG" {
		t.Fatal("expected GOOG cache hit")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/service/ -run TestHistoryCache -v`
Expected: FAIL — `NewHistoryCache` does not exist.

- [ ] **Step 3: Implement the cache**

Create `backend/internal/service/history_cache.go`:

```go
package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type historyCacheEntry struct {
	data      *model.HistoricalPriceResponse
	expiresAt time.Time
}

type HistoryCache struct {
	mu      sync.RWMutex
	entries map[string]historyCacheEntry
	ttl     time.Duration
}

func NewHistoryCache(ttl time.Duration) *HistoryCache {
	return &HistoryCache{
		entries: make(map[string]historyCacheEntry),
		ttl:     ttl,
	}
}

func (c *HistoryCache) cacheKey(ticker, start, end string) string {
	return fmt.Sprintf("%s:%s:%s", ticker, start, end)
}

func (c *HistoryCache) Get(ticker, start, end string) (*model.HistoricalPriceResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[c.cacheKey(ticker, start, end)]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *HistoryCache) Set(ticker, start, end string, data *model.HistoricalPriceResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[c.cacheKey(ticker, start, end)] = historyCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/service/ -run TestHistoryCache -v`
Expected: All 4 cache tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/history_cache.go backend/internal/service/history_cache_test.go
git commit -m "feat(backend): add in-memory TTL cache for historical prices"
```

---

## Task 7: Go — Service and Handler for `GetPriceHistory`

**Files:**

- Modify: `backend/internal/service/portfolio.go`
- Modify: `backend/internal/handler/portfolio.go`
- Modify: `backend/cmd/server/main.go`
- Create: `backend/internal/handler/portfolio_history_test.go`

- [ ] **Step 1: Write the failing handler tests**

Create `backend/internal/handler/portfolio_history_test.go`:

```go
package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type mockPortfolioService struct {
	getPortfolioFn     func() (*model.Portfolio, error)
	getPriceFn         func(ticker string) (*model.PriceCache, error)
	getPriceHistoryFn  func(ticker, start, end string) (*model.HistoricalPriceResponse, error)
}

func (m *mockPortfolioService) GetPortfolio() (*model.Portfolio, error) {
	if m.getPortfolioFn != nil {
		return m.getPortfolioFn()
	}
	return nil, nil
}

func (m *mockPortfolioService) GetPrice(ticker string) (*model.PriceCache, error) {
	if m.getPriceFn != nil {
		return m.getPriceFn(ticker)
	}
	return nil, nil
}

func (m *mockPortfolioService) GetPriceHistory(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
	if m.getPriceHistoryFn != nil {
		return m.getPriceHistoryFn(ticker, start, end)
	}
	return nil, nil
}

func newTestRouter(h *handler.PortfolioHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/prices/{ticker}/history", h.GetPriceHistory)
	return r
}

func TestGetPriceHistory_Success(t *testing.T) {
	open := 150.0
	mock := &mockPortfolioService{
		getPriceHistoryFn: func(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
			return &model.HistoricalPriceResponse{
				Ticker:   ticker,
				Currency: "USD",
				Interval: "daily",
				Prices: []model.HistoricalPrice{
					{Date: "2025-01-02", Open: &open},
				},
			}, nil
		},
	}

	h := &handler.PortfolioHandler{Svc: mock}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/AAPL/history?start=2025-01-01&end=2025-12-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.HistoricalPriceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", resp.Ticker)
	}
}

func TestGetPriceHistory_DefaultDates(t *testing.T) {
	mock := &mockPortfolioService{
		getPriceHistoryFn: func(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
			if start == "" || end == "" {
				t.Error("expected default dates to be set")
			}
			return &model.HistoricalPriceResponse{Ticker: ticker, Prices: []model.HistoricalPrice{}}, nil
		},
	}

	h := &handler.PortfolioHandler{Svc: mock}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/AAPL/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPriceHistory_InvalidDateFormat(t *testing.T) {
	h := &handler.PortfolioHandler{Svc: &mockPortfolioService{}}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/AAPL/history?start=not-a-date&end=2025-12-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetPriceHistory_StartAfterEnd(t *testing.T) {
	h := &handler.PortfolioHandler{Svc: &mockPortfolioService{}}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/AAPL/history?start=2025-12-31&end=2025-01-01", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetPriceHistory_ServiceError(t *testing.T) {
	mock := &mockPortfolioService{
		getPriceHistoryFn: func(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
			return nil, fmt.Errorf("data service returned 404 for ticker INVALID history")
		},
	}

	h := &handler.PortfolioHandler{Svc: mock}
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/prices/INVALID/history?start=2025-01-01&end=2025-12-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/handler/ -run TestGetPriceHistory -v`
Expected: FAIL — `GetPriceHistory` method and interface don't exist.

- [ ] **Step 3: Update the `PortfolioServiceInterface` in `portfolio.go` handler**

In `backend/internal/handler/portfolio.go`, add the import for `time` and add the new method to the interface and handler. Replace the entire file content:

```go
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type PortfolioServiceInterface interface {
	GetPortfolio() (*model.Portfolio, error)
	GetPrice(ticker string) (*model.PriceCache, error)
	GetPriceHistory(ticker, start, end string) (*model.HistoricalPriceResponse, error)
}

type PortfolioHandler struct {
	Svc PortfolioServiceInterface
}

func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	portfolio, err := h.Svc.GetPortfolio()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portfolio)
}

func (h *PortfolioHandler) GetPrice(w http.ResponseWriter, r *http.Request) {
	ticker := chi.URLParam(r, "ticker")
	price, err := h.Svc.GetPrice(ticker)
	if err != nil {
		http.Error(w, `{"error":"price not available for `+ticker+`"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(price)
}

func (h *PortfolioHandler) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	ticker := chi.URLParam(r, "ticker")

	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	// Default: 1 year range ending today
	now := time.Now()
	if end == "" {
		end = now.Format("2006-01-02")
	}
	if start == "" {
		start = now.AddDate(-1, 0, 0).Format("2006-01-02")
	}

	// Validate date formats
	startDate, err := time.Parse("2006-01-02", start)
	if err != nil {
		http.Error(w, `{"error":"invalid start date format, use YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}
	endDate, err := time.Parse("2006-01-02", end)
	if err != nil {
		http.Error(w, `{"error":"invalid end date format, use YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}

	// Validate date logic
	if !startDate.Before(endDate) {
		http.Error(w, `{"error":"start date must be before end date"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.Svc.GetPriceHistory(ticker, start, end)
	if err != nil {
		http.Error(w, `{"error":"price history not available for `+ticker+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
```

- [ ] **Step 4: Add `GetPriceHistory` to `PortfolioService` in `service/portfolio.go`**

Add this method at the end of `backend/internal/service/portfolio.go`, after `fetchPrice()`:

```go
func (s *PortfolioService) GetPriceHistory(ticker, start, end string) (*model.HistoricalPriceResponse, error) {
	// Check in-memory cache
	if s.HistoryCache != nil {
		if cached, ok := s.HistoryCache.Get(ticker, start, end); ok {
			return cached, nil
		}
	}

	// Fetch from Python data service
	resp, err := s.DataClient.GetPriceHistory(ticker, start, end)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.HistoryCache != nil {
		s.HistoryCache.Set(ticker, start, end, resp)
	}

	return resp, nil
}
```

Also add the `HistoryCache` field to the `PortfolioService` struct:

```go
type PortfolioService struct {
	PortfolioRepo  *repository.PortfolioRepo
	PriceCacheRepo *repository.PriceCacheRepo
	DataClient     *client.DataServiceClient
	HistoryCache   *HistoryCache
}
```

- [ ] **Step 5: Wire the route and cache in `main.go`**

In `backend/cmd/server/main.go`, add cache creation after the `dataClient` line:

```go
historyCache := service.NewHistoryCache(15 * time.Minute)
```

Add `time` to the imports:

```go
"time"
```

Update the `portfolioSvc` initialization to include the cache:

```go
portfolioSvc := &service.PortfolioService{
	PortfolioRepo:  portfolioRepo,
	PriceCacheRepo: priceCacheRepo,
	DataClient:     dataClient,
	HistoryCache:   historyCache,
}
```

Add the route inside the `/api` group after the existing `r.Get("/prices/{ticker}", ...)` line:

```go
r.Get("/prices/{ticker}/history", portfolioHandler.GetPriceHistory)
```

- [ ] **Step 6: Run handler tests**

Run: `cd backend && go test ./internal/handler/ -run TestGetPriceHistory -v`
Expected: All 5 handler tests PASS.

- [ ] **Step 7: Run all backend tests**

Run: `cd backend && go test ./... -v`
Expected: All tests PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/handler/portfolio.go backend/internal/handler/portfolio_history_test.go backend/internal/service/portfolio.go backend/cmd/server/main.go
git commit -m "feat(backend): add price history endpoint with caching"
```

---

## Task 8: Frontend — API Client and Chart Utils

**Files:**

- Modify: `frontend/lib/api.ts`
- Create: `frontend/lib/chart-utils.ts`
- Create: `frontend/lib/chart-utils.test.ts`

- [ ] **Step 1: Add types and function to `api.ts`**

Append to the end of `frontend/lib/api.ts`:

```typescript
export interface HistoricalPricePoint {
  date: string;
  open: number | null;
  high: number | null;
  low: number | null;
  close: number | null;
  adj_close: number | null;
  volume: number | null;
}

export interface HistoricalPriceResponse {
  ticker: string;
  currency: string;
  interval: string;
  prices: HistoricalPricePoint[];
}

export async function getHistoricalPrices(
  ticker: string,
  start: string,
  end: string,
): Promise<HistoricalPriceResponse> {
  // Client-side fetch — uses BACKEND_URL from env or falls back to relative path
  const baseUrl = typeof window === "undefined" ? BACKEND_URL : "";
  const res = await fetch(
    `${baseUrl}/api/prices/${ticker}/history?start=${start}&end=${end}`,
    { cache: "no-store" },
  );
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
  return res.json();
}
```

- [ ] **Step 2: Create `chart-utils.ts`**

Create `frontend/lib/chart-utils.ts`:

```typescript
import { HistoricalPricePoint } from "./api";

export type ChartMode = "candlestick" | "line";

export function determineChartMode(prices: HistoricalPricePoint[]): ChartMode {
  if (prices.length === 0) return "line";

  const nullOhlcCount = prices.filter(
    (p) => p.open === null && p.high === null && p.low === null,
  ).length;

  const nullRatio = nullOhlcCount / prices.length;
  return nullRatio > 0.2 ? "line" : "candlestick";
}

export function hasVolumeData(prices: HistoricalPricePoint[]): boolean {
  return prices.some((p) => p.volume !== null && p.volume !== 0);
}
```

- [ ] **Step 3: Write tests for chart-utils**

Create `frontend/lib/chart-utils.test.ts`:

```typescript
import { describe, expect, it } from "vitest";
import { determineChartMode, hasVolumeData } from "./chart-utils";
import { HistoricalPricePoint } from "./api";

function makePrice(
  overrides: Partial<HistoricalPricePoint> = {},
): HistoricalPricePoint {
  return {
    date: "2025-01-01",
    open: 100,
    high: 105,
    low: 99,
    close: 103,
    adj_close: 103,
    volume: 1000000,
    ...overrides,
  };
}

describe("determineChartMode", () => {
  it("returns candlestick when all OHLC data present", () => {
    const prices = [makePrice(), makePrice(), makePrice()];
    expect(determineChartMode(prices)).toBe("candlestick");
  });

  it("returns line when >20% of rows have null OHLC", () => {
    const prices = [
      makePrice({ open: null, high: null, low: null }),
      makePrice({ open: null, high: null, low: null }),
      makePrice(),
    ];
    // 2/3 = 66% null → line
    expect(determineChartMode(prices)).toBe("line");
  });

  it("returns candlestick when <=20% of rows have null OHLC", () => {
    const prices = [
      makePrice({ open: null, high: null, low: null }),
      makePrice(),
      makePrice(),
      makePrice(),
      makePrice(),
      makePrice(),
    ];
    // 1/6 = 16% null → candlestick
    expect(determineChartMode(prices)).toBe("candlestick");
  });

  it("returns line for empty array", () => {
    expect(determineChartMode([])).toBe("line");
  });

  it("returns candlestick when only some fields are null (not all three)", () => {
    const prices = [
      makePrice({ open: null }), // only open is null, high and low present
      makePrice(),
    ];
    expect(determineChartMode(prices)).toBe("candlestick");
  });
});

describe("hasVolumeData", () => {
  it("returns true when volume data exists", () => {
    expect(hasVolumeData([makePrice()])).toBe(true);
  });

  it("returns false when all volumes are null", () => {
    expect(hasVolumeData([makePrice({ volume: null })])).toBe(false);
  });

  it("returns false when all volumes are zero", () => {
    expect(hasVolumeData([makePrice({ volume: 0 })])).toBe(false);
  });

  it("returns false for empty array", () => {
    expect(hasVolumeData([])).toBe(false);
  });
});
```

- [ ] **Step 4: Install vitest (if not already installed)**

Run: `cd frontend && npm install -D vitest`

- [ ] **Step 5: Run the chart-utils tests**

Run: `cd frontend && npx vitest run lib/chart-utils.test.ts`
Expected: All 9 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/lib/api.ts frontend/lib/chart-utils.ts frontend/lib/chart-utils.test.ts frontend/package.json frontend/package-lock.json
git commit -m "feat(frontend): add historical prices API client and chart mode utils"
```

---

## Task 9: Frontend — Install `lightweight-charts` and Build Chart Component

**Files:**

- Create: `frontend/components/stock-chart.tsx`

- [ ] **Step 1: Install lightweight-charts**

Run: `cd frontend && npm install lightweight-charts`

- [ ] **Step 2: Create `stock-chart.tsx`**

Create `frontend/components/stock-chart.tsx`:

```tsx
"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  createChart,
  IChartApi,
  ISeriesApi,
  ColorType,
  CandlestickData,
  LineData,
  HistogramData,
  Time,
} from "lightweight-charts";
import {
  Holding,
  HistoricalPricePoint,
  HistoricalPriceResponse,
  getHistoricalPrices,
} from "@/lib/api";
import {
  determineChartMode,
  hasVolumeData,
  ChartMode,
} from "@/lib/chart-utils";

interface StockChartProps {
  holdings: Holding[];
}

type RangePreset = "1M" | "3M" | "6M" | "YTD" | "1Y" | "5Y" | "Max";

function getStartDate(preset: RangePreset): string {
  const now = new Date();
  let start: Date;

  switch (preset) {
    case "1M":
      start = new Date(now.getFullYear(), now.getMonth() - 1, now.getDate());
      break;
    case "3M":
      start = new Date(now.getFullYear(), now.getMonth() - 3, now.getDate());
      break;
    case "6M":
      start = new Date(now.getFullYear(), now.getMonth() - 6, now.getDate());
      break;
    case "YTD":
      start = new Date(now.getFullYear(), 0, 1);
      break;
    case "1Y":
      start = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate());
      break;
    case "5Y":
      start = new Date(now.getFullYear() - 5, now.getMonth(), now.getDate());
      break;
    case "Max":
      start = new Date(1970, 0, 1);
      break;
  }

  return start.toISOString().split("T")[0];
}

function formatDate(d: Date): string {
  return d.toISOString().split("T")[0];
}

export function StockChart({ holdings }: StockChartProps) {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);

  const [selectedTicker, setSelectedTicker] = useState(
    holdings[0]?.ticker ?? "",
  );
  const [activePreset, setActivePreset] = useState<RangePreset>("1Y");
  const [data, setData] = useState<HistoricalPriceResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [chartMode, setChartMode] = useState<ChartMode>("candlestick");

  const fetchData = useCallback(async (ticker: string, preset: RangePreset) => {
    setLoading(true);
    setError(null);
    try {
      const start = getStartDate(preset);
      const end = formatDate(new Date());
      const resp = await getHistoricalPrices(ticker, start, end);
      setData(resp);
      setChartMode(determineChartMode(resp.prices));
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : "Unable to load price data. The asset may be delisted or the data provider may be temporarily unavailable.",
      );
      setData(null);
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch data when ticker or preset changes
  useEffect(() => {
    if (selectedTicker) {
      fetchData(selectedTicker, activePreset);
    }
  }, [selectedTicker, activePreset, fetchData]);

  // Render chart
  useEffect(() => {
    if (!chartContainerRef.current || !data || data.prices.length === 0) return;

    // Clean up previous chart
    if (chartRef.current) {
      chartRef.current.remove();
      chartRef.current = null;
    }

    const container = chartContainerRef.current;
    const chart = createChart(container, {
      layout: {
        background: { type: ColorType.Solid, color: "white" },
        textColor: "#374151",
      },
      width: container.clientWidth,
      height: 400,
      grid: {
        vertLines: { color: "#f3f4f6" },
        horzLines: { color: "#f3f4f6" },
      },
      crosshair: {
        mode: 0,
      },
      timeScale: {
        borderColor: "#e5e7eb",
      },
      rightPriceScale: {
        borderColor: "#e5e7eb",
      },
    });

    chartRef.current = chart;

    const showVolume = hasVolumeData(data.prices);

    if (chartMode === "candlestick") {
      const candleSeries = chart.addCandlestickSeries({
        upColor: "#16a34a",
        downColor: "#dc2626",
        borderDownColor: "#dc2626",
        borderUpColor: "#16a34a",
        wickDownColor: "#dc2626",
        wickUpColor: "#16a34a",
      });

      const candleData: CandlestickData[] = data.prices
        .filter((p) => p.close !== null)
        .map((p) => ({
          time: p.date as Time,
          open: p.open ?? p.close!,
          high: p.high ?? p.close!,
          low: p.low ?? p.close!,
          close: p.close!,
        }));

      candleSeries.setData(candleData);
    } else {
      const lineSeries = chart.addLineSeries({
        color: "#2563eb",
        lineWidth: 2,
      });

      const lineData: LineData[] = data.prices
        .filter((p) => (p.adj_close ?? p.close) !== null)
        .map((p) => ({
          time: p.date as Time,
          value: p.adj_close ?? p.close!,
        }));

      lineSeries.setData(lineData);
    }

    if (showVolume) {
      const volumeSeries = chart.addHistogramSeries({
        priceFormat: { type: "volume" },
        priceScaleId: "volume",
      });

      chart.priceScale("volume").applyOptions({
        scaleMargins: { top: 0.8, bottom: 0 },
      });

      const volumeData: HistogramData[] = data.prices
        .filter((p) => p.volume !== null && p.volume !== 0)
        .map((p) => {
          const isUp =
            p.close !== null && p.open !== null ? p.close >= p.open : true;
          return {
            time: p.date as Time,
            value: p.volume!,
            color: isUp ? "rgba(22, 163, 74, 0.3)" : "rgba(220, 38, 38, 0.3)",
          };
        });

      volumeSeries.setData(volumeData);
    }

    chart.timeScale().fitContent();

    // Resize handler
    const handleResize = () => {
      if (chartRef.current && container) {
        chartRef.current.applyOptions({ width: container.clientWidth });
      }
    };
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      if (chartRef.current) {
        chartRef.current.remove();
        chartRef.current = null;
      }
    };
  }, [data, chartMode]);

  const presets: RangePreset[] = ["1M", "3M", "6M", "YTD", "1Y", "5Y", "Max"];

  return (
    <div>
      {/* Controls */}
      <div className="flex flex-wrap items-center gap-4 mb-4">
        {/* Asset selector */}
        <select
          value={selectedTicker}
          onChange={(e) => setSelectedTicker(e.target.value)}
          className="px-3 py-2 border rounded-md text-sm bg-white"
        >
          {holdings.map((h) => (
            <option key={h.ticker} value={h.ticker}>
              {h.ticker} — {h.name}
            </option>
          ))}
        </select>

        {/* Range presets */}
        <div className="flex gap-1">
          {presets.map((p) => (
            <button
              key={p}
              onClick={() => setActivePreset(p)}
              className={`px-3 py-1 text-sm rounded-md ${
                activePreset === p
                  ? "bg-gray-900 text-white"
                  : "bg-gray-100 text-gray-600 hover:bg-gray-200"
              }`}
            >
              {p}
            </button>
          ))}
        </div>

        {/* Reset zoom */}
        <button
          onClick={() => chartRef.current?.timeScale().fitContent()}
          className="px-3 py-1 text-sm bg-gray-100 text-gray-600 hover:bg-gray-200 rounded-md"
        >
          Reset Zoom
        </button>
      </div>

      {/* Chart area */}
      <div className="bg-white rounded-lg shadow-sm border p-4">
        {loading && (
          <div className="flex items-center justify-center h-[400px]">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900" />
          </div>
        )}

        {error && (
          <div className="flex flex-col items-center justify-center h-[400px] text-center">
            <p className="text-red-500 mb-4">{error}</p>
            <button
              onClick={() => fetchData(selectedTicker, activePreset)}
              className="px-4 py-2 bg-gray-900 text-white rounded-md text-sm hover:bg-gray-800"
            >
              Retry
            </button>
          </div>
        )}

        {!loading && !error && data && data.prices.length === 0 && (
          <div className="flex items-center justify-center h-[400px]">
            <p className="text-gray-500">
              No historical price data available for this asset.
            </p>
          </div>
        )}

        {!loading && !error && data && data.prices.length > 0 && (
          <>
            <div ref={chartContainerRef} />
            {/* Indicators */}
            <div className="mt-2 flex gap-4 text-xs text-gray-400">
              {chartMode === "line" && (
                <span>
                  Showing adjusted close only — full OHLC data not available for
                  this asset.
                </span>
              )}
              {data.interval !== "daily" && (
                <span>Showing {data.interval} data</span>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No type errors (warnings about unused vars are OK).

- [ ] **Step 4: Commit**

```bash
git add frontend/components/stock-chart.tsx frontend/package.json frontend/package-lock.json
git commit -m "feat(frontend): add StockChart component with lightweight-charts"
```

---

## Task 10: Frontend — Charts Page and Navigation

**Files:**

- Create: `frontend/app/charts/page.tsx`
- Modify: `frontend/app/layout.tsx`

- [ ] **Step 1: Create the charts page**

Create `frontend/app/charts/page.tsx`:

```tsx
import { getPortfolio } from "@/lib/api";
import { StockChart } from "@/components/stock-chart";

export const dynamic = "force-dynamic";

export default async function ChartsPage() {
  let holdings;
  try {
    const portfolio = await getPortfolio();
    holdings = portfolio.holdings;
  } catch {
    return (
      <div className="text-center py-8">
        <p className="text-red-500">
          Failed to load portfolio. Is the backend running?
        </p>
      </div>
    );
  }

  if (holdings.length === 0) {
    return (
      <div className="text-center py-8">
        <p className="text-gray-500">
          No holdings yet. Add a transaction to get started, then come back to
          see charts.
        </p>
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Price Charts</h1>
      <StockChart holdings={holdings} />
    </div>
  );
}
```

- [ ] **Step 2: Add "Charts" link to the navbar**

In `frontend/app/layout.tsx`, add a new `Link` after the "Transactions" link (line 28):

```tsx
<Link href="/charts" className="text-gray-600 hover:text-gray-900">
  Charts
</Link>
```

- [ ] **Step 3: Verify the build compiles**

Run: `cd frontend && npm run build`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/app/charts/page.tsx frontend/app/layout.tsx
git commit -m "feat(frontend): add charts page and navigation link"
```

---

## Task 11: Integration Smoke Test

- [ ] **Step 1: Start the full stack**

Run: `cd src && docker-compose up --build`
Expected: All 4 services start without errors.

- [ ] **Step 2: Add a test transaction (if no holdings exist)**

In a browser, go to `http://localhost:3000/add` and add a buy transaction for AAPL.

- [ ] **Step 3: Navigate to the Charts page**

Go to `http://localhost:3000/charts`.
Expected:

- Dropdown shows AAPL
- 1Y candlestick chart renders with volume bars
- Preset buttons work (click 1M, 5Y, Max)
- Crosshair shows date/price on hover
- Zoom with scroll, pan with drag, Reset Zoom button works

- [ ] **Step 4: Test line chart fallback**

Add a transaction for a mutual fund ticker (e.g., `VFIAX`). Select it in the dropdown.
Expected: Line chart renders with "Showing adjusted close only" indicator.

- [ ] **Step 5: Test empty state**

Add a transaction for a delisted/invalid ticker. Select it in the dropdown.
Expected: "No historical price data available" or error message with Retry button.

- [ ] **Step 6: Stop the stack**

Run: `cd src && docker-compose down`
