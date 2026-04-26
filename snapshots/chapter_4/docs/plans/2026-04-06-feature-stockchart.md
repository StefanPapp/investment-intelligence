# 📋 Feature Requirements Document: Stock Chart

Status: Not started
Type: feature
Created: April 6, 2026 6:19 PM
Last edited time: April 6, 2026 6:19 PM

## Overview

Display an interactive price chart for any asset in the portfolio. The chart shows historical price data sourced from yfinance via the Python data service. The user selects an asset from their portfolio and a date range. The chart renders candlestick bars (OHLC) when full data is available, and falls back to a line chart (adjusted close) when it is not.

---

## User Stories

1. As a portfolio owner, I want to see a price chart for any asset I hold so I can visually assess price trends.
2. As a portfolio owner, I want to choose a date range so I can focus on the period that matters to me.
3. As a portfolio owner, I want the chart to degrade gracefully when data is incomplete so I am never confused by broken visuals.

---

## Functional Requirements

### FR-1: Asset selection

The user picks an asset from a dropdown populated by their portfolio holdings. The dropdown displays the ticker symbol and asset name. Selecting an asset triggers a data fetch for the default date range.

### FR-2: Date range selection

**Default range:** 1 year from today, backward.

**Preset buttons:** 1M, 3M, 6M, YTD, 1Y, 5Y, Max.

**Custom range:** Date picker for start and end dates.

**Constraints:**

- Start date must not be before the asset's earliest available trading date (IPO date or first record in yfinance).
- End date must not be in the future.
- If the user selects a range where no trading data exists, display an informational message: "No trading data available for this date range."
- Weekend and holiday dates silently snap to the nearest trading day.

### FR-3: Chart rendering

**Primary mode — Candlestick (OHLC + Volume):**

- Render when yfinance returns all four OHLC columns with non-null values.
- Green candle for close ≥ open; red candle for close < open.
- Volume bars rendered in a secondary axis below the price chart, height-proportional, same color coding as the candle.

**Fallback mode — Line chart (Adjusted Close):**

- Render when OHLC data is incomplete (e.g., mutual funds, some ETFs).
- Single line using the `Adj Close` column.
- No volume subplot (volume is typically unavailable when OHLC is missing).
- Display a subtle indicator: "Showing adjusted close only — full OHLC data not available for this asset."

**Volume suppression:**

- If volume data is all zeros or all null for the selected range, hide the volume subplot entirely rather than showing empty bars.

### FR-4: Interactivity

- Crosshair cursor showing date, OHLC values (or close), and volume on hover.
- Zoom via mouse scroll or pinch gesture.
- Pan via click-drag.
- Reset zoom button.

### FR-5: Data sourcing

- The frontend requests data from the Go backend via `GET /api/v1/assets/{id}/prices?start={date}&end={date}`.
- The backend delegates to the Python data service, which calls `yfinance.download(ticker, start, end)`.
- Response payload: JSON array of `{date, open, high, low, close, adj_close, volume}` objects.
- Null fields are permitted; the frontend uses them to decide candlestick vs. line mode.

---

## Edge Cases

### EC-1: yfinance returns incomplete data

**Scenario:** yfinance returns rows where some OHLC fields are `NaN` or `None` (common for mutual funds, newly listed assets, or thinly traded securities).

**Handling:**

- If >20% of rows in the response have null OHLC fields, switch the entire chart to line mode.
- If ≤20% of rows have null fields, render candlestick and interpolate missing candles as thin horizontal lines at the close price (visual indicator that data is imputed).
- Log a warning to the console with the count of imputed rows.

### EC-2: yfinance returns no data at all

**Scenario:** Ticker is delisted, yfinance API is down, or network error.

**Handling:**

- Display an empty chart area with the message: "Unable to load price data. The asset may be delisted or the data provider may be temporarily unavailable."
- Show a "Retry" button.
- The backend returns HTTP 502 (upstream failure) or 404 (unknown ticker) with a JSON error body `{error: string, retryable: boolean}`.

### EC-3: Newly added asset with no trading history

**Scenario:** User adds a position manually (e.g., a private holding or a very recent IPO) and no yfinance data exists.

**Handling:**

- The chart area displays: "No historical price data available for this asset."
- The asset still appears in the dropdown but the chart is gracefully empty.
- No error toast — this is an expected state, not a failure.

### EC-4: Splits and dividends

**Scenario:** A stock split or dividend occurs within the selected date range.

**Handling:**

- Use `Adj Close` for the line chart mode (already split-adjusted by yfinance).
- For candlestick mode, use raw OHLC (not adjusted). Add a callout annotation on the chart at the split date: "2:1 split" or similar, sourced from yfinance's `actions` data.
- This is a stretch goal for v1; acceptable to ship without split annotations initially.

### EC-5: Rate limiting from yfinance

**Scenario:** Too many requests in a short window cause yfinance to throttle or return errors.

**Handling:**

- The Python data service implements exponential backoff with a maximum of 3 retries.
- If all retries fail, return a 503 to the backend, which forwards it as a retryable error to the frontend.
- The frontend displays: "Data provider is temporarily busy. Please try again in a moment."

### EC-6: Very long date ranges (Max)

**Scenario:** User selects "Max" for an asset with 30+ years of history. Response is large.

**Handling:**

- The Python data service resamples to weekly OHLC for ranges >5 years and monthly OHLC for ranges >15 years.
- The frontend indicates the resampling: "Showing weekly data" or "Showing monthly data".

---

## Non-Functional Requirements

- **Latency:** Chart renders within 2 seconds for a 1-year daily range on a typical broadband connection.
- **Caching:** The backend caches yfinance responses for 15 minutes (price data doesn't change intraday for our use case since we fetch daily bars).
- **Charting library:** Use [TradingView Lightweight Charts](https://github.com/nickvuleli/lightweight-charts) for the frontend. It supports candlestick, line, and volume natively, renders via Canvas (performant), and has a small bundle size. Alternative: if React wrapper is needed, use `react-lightweight-charts`.
- **Responsive:** The chart resizes to fill its container on desktop and mobile viewports.

---

## Out of Scope for v1

- Technical indicators (Bollinger Bands, moving averages) — covered separately in Section 5.2.4.
- Catalyst annotations (earnings, events) — covered in Section 5.2.3.
- Comparison overlay (benchmark vs. asset) — covered in Section 5.4.
- Real-time / streaming price updates.

---

## Definition of Done

- [ ] User can select any portfolio asset from a dropdown and see a price chart.
- [ ] Default date range is 1 year; preset buttons (1M, 3M, 6M, YTD, 1Y, 5Y, Max) work correctly.
- [ ] Candlestick + volume renders when full OHLC data is available.
- [ ] Line chart renders as fallback when OHLC is incomplete.
- [ ] Volume subplot is hidden when volume data is all zeros/null.
- [ ] Empty state displays a clear message for assets with no price history.
- [ ] Error state displays a message and retry button when yfinance fails.
- [ ] Resampling activates for ranges >5 years (weekly) and >15 years (monthly).
- [ ] Chart hover shows crosshair with date and price values.
- [ ] Zoom, pan, and reset-zoom work on desktop and mobile.
- [ ] Backend endpoint `GET /api/v1/assets/{id}/prices` returns correct JSON payload.
- [ ] Python data service handles yfinance failures with exponential backoff (3 retries).
- [ ] Backend caches responses for 15 minutes.
- [ ] Page loads and renders chart within 2 seconds for a 1-year range.
- [ ] Unit tests cover: candlestick mode, line fallback, empty data, error responses, resampling logic.
- [ ] Manual smoke test on at least one mutual fund (line fallback), one stock (candlestick), and one delisted ticker (empty state).
