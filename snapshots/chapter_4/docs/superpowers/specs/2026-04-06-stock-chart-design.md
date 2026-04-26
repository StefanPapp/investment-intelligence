# Stock Chart Feature — Design Spec

**Status:** Approved
**Date:** 2026-04-06
**Requirements:** `docs/plans/2026-04-06-feature-stockchart.md`

---

## 1. API & Data Flow

**Endpoint:** `GET /api/prices/{ticker}/history?start=YYYY-MM-DD&end=YYYY-MM-DD`

Uses ticker-based routing consistent with the existing `GET /api/prices/{ticker}` endpoint (deviates from the requirements doc's `/api/v1/assets/{id}/prices` — decided during brainstorming to avoid a version prefix and UUID resolution not used elsewhere).

```
Frontend (client-side fetch from chart component)
  → Go Backend: GET /api/prices/{ticker}/history?start=...&end=...
    → validate dates, determine resampling interval
    → check in-memory cache (ticker:start:end key, 15-min TTL)
    → if miss: call Python: GET /price/{ticker}/history?start=...&end=...
      → yfinance.Ticker(ticker).history(start, end)
    → cache response, return JSON
  → Frontend renders candlestick or line chart
```

**Response payload:**

```json
{
  "ticker": "AAPL",
  "currency": "USD",
  "interval": "daily",
  "prices": [
    {
      "date": "2025-04-07",
      "open": 150.0,
      "high": 152.0,
      "low": 149.0,
      "close": 151.5,
      "adj_close": 151.5,
      "volume": 48000000
    }
  ]
}
```

- Null OHLC fields signal line-chart fallback on the frontend.
- `interval` is `"daily"`, `"weekly"`, or `"monthly"` reflecting resampling.

---

## 2. Python Data Service

### New endpoint

`GET /price/{ticker}/history?start=YYYY-MM-DD&end=YYYY-MM-DD`

### Service method

`MarketDataService.get_historical_prices(ticker, start_date, end_date) -> dict`

- Uses `yf.Ticker(ticker).history(start=start, end=end)`.
- Converts DataFrame to list of dicts. NaN values become `None` (JSON null).
- Returns dict with ticker, currency, interval, and prices list.

### Resampling (EC-6)

- Range > 5 years: resample to weekly OHLC via pandas `.resample('W').agg(...)`.
- Range > 15 years: resample to monthly OHLC.
- Aggregation: open=first, high=max, low=min, close=last, volume=sum.

### Retry logic (EC-5)

- `tenacity` with exponential backoff, 3 retries max.
- Final failure raises exception mapped to HTTP 503.

### Error mapping

| Condition                      | HTTP Status | `retryable` |
| ------------------------------ | ----------- | ----------- |
| Invalid ticker / no data       | 404         | false       |
| yfinance failure after retries | 503         | true        |

### New Pydantic models

- `HistoricalPricePoint`: date (str), open/high/low/close/adj_close/volume (Optional[float]).
- `HistoricalPriceResponse`: ticker, currency, interval, prices (list[HistoricalPricePoint]).

---

## 3. Go Backend

### Handler

`PortfolioHandler.GetPriceHistory(w, r)` — parses ticker from path, start/end from query params.

### Validation

| Rule                | Default    |
| ------------------- | ---------- |
| `start` missing     | 1 year ago |
| `end` missing       | today      |
| `end` in the future | HTTP 400   |
| `start` >= `end`    | HTTP 400   |
| Invalid date format | HTTP 400   |

### Caching

In-memory TTL cache in the service layer:

- `map[string]cachedEntry` keyed by `ticker:start:end`.
- 15-minute TTL, protected by `sync.RWMutex`.
- No new database table or migration needed.

### Client method

`DataServiceClient.GetPriceHistory(ticker, start, end string) (*model.HistoricalPriceResponse, error)`

- Calls Python `GET /price/{ticker}/history?start=...&end=...`.
- HTTP client timeout bumped to 30s for this call (large ranges).

### New model

```go
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

Nullable floats via `*float64` to preserve null semantics from Python.

### Route

```go
r.Get("/prices/{ticker}/history", portfolioHandler.GetPriceHistory)
```

### Error forwarding

| Python status | Go status              |
| ------------- | ---------------------- |
| 404           | 404                    |
| 503           | 502 (upstream failure) |
| Timeout       | 504                    |

---

## 4. Frontend

### Page structure

- `/app/charts/page.tsx` — Server Component. Fetches portfolio holdings for the dropdown, renders the client component.
- `/components/stock-chart.tsx` — Client Component (`"use client"`). Contains all interactive chart logic.

### Asset selection (FR-1)

Dropdown populated from portfolio holdings (passed as props). Displays `TICKER — Name`. Selecting triggers fetch for default 1Y range.

### Date range (FR-2)

Preset buttons: **1M, 3M, 6M, YTD, 1Y, 5Y, Max** — styled as a button group. No custom date picker in v1. Selecting recalculates start date relative to today and re-fetches.

### Charting library

**TradingView `lightweight-charts`** (as spec recommends):

- ~45KB gzipped, Canvas-based.
- Built-in candlestick, line, and histogram series.
- No React wrapper — mount via `useRef` + `useEffect`, cleanup on unmount.
- Built-in crosshair, zoom/pan, resize.

### Chart mode (FR-3 + EC-1)

Logic extracted to `lib/chart-utils.ts` → `determineChartMode(prices)`:

- Count rows where open, high, low are all null.
- If >20% null → line chart using `adj_close` + indicator text.
- If <=20% null → candlestick; imputed candles as thin horizontal lines at close.

### Volume subplot

- Hidden if all volume values are null or zero.
- Otherwise rendered as histogram series below the price chart with green/red color coding matching the candle.

### Interactivity (FR-4)

- Crosshair with OHLCV tooltip — built into lightweight-charts.
- Zoom via scroll, pan via drag — built-in.
- Reset zoom button — calls `chart.timeScale().fitContent()`.

### Resampling indicator

When `interval` is `"weekly"` or `"monthly"`, display text below the chart.

### States

| State         | Display                                                                         |
| ------------- | ------------------------------------------------------------------------------- |
| Loading       | Spinner in chart area                                                           |
| Empty (EC-3)  | "No historical price data available for this asset."                            |
| Error (EC-2)  | Error message + Retry button                                                    |
| Line fallback | Subtle indicator: "Showing adjusted close only — full OHLC data not available." |

### API client

New function in `lib/api.ts`: `getHistoricalPrices(ticker, start, end)` — client-side fetch (not Server Action, since this is read-only interactive data).

### Navigation

Add "Charts" link to navbar in `app/layout.tsx`.

---

## 5. Testing

### Python data service

- Mock `yf.Ticker().history()` DataFrame return.
- Cases: valid full OHLC, partial nulls, invalid ticker (404), empty response, resampling (>5yr, >15yr), retry behavior.

### Go backend

- **Handler tests:** Mock service interface. Test param validation (missing, invalid format, end-in-future, start>=end). Test success response shape.
- **Client tests:** `httptest.NewServer` mocking Python. Test success + error forwarding.
- **Service tests:** Mock client, verify cache hit/miss behavior.

### Frontend

- Unit test `determineChartMode()` in `lib/chart-utils.ts` with Vitest (pure function, no DOM).
- Chart component itself is Canvas-based — not unit-testable in jsdom. Verified via manual smoke test.
- Smoke test matrix: one stock (candlestick), one mutual fund (line fallback), one delisted ticker (empty state).

---

## 6. Out of Scope

Per requirements doc: technical indicators, catalyst annotations, benchmark overlay, real-time streaming, split annotations (stretch goal, not v1).

Custom date picker deferred — preset buttons cover all specified ranges.
