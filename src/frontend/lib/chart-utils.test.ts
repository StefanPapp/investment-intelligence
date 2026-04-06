import { describe, expect, it } from "vitest";
import { determineChartMode, hasVolumeData } from "./chart-utils";
import { HistoricalPricePoint } from "./api";

function makePrice(overrides: Partial<HistoricalPricePoint> = {}): HistoricalPricePoint {
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
    expect(determineChartMode(prices)).toBe("candlestick");
  });

  it("returns line for empty array", () => {
    expect(determineChartMode([])).toBe("line");
  });

  it("returns candlestick when only some fields are null (not all three)", () => {
    const prices = [
      makePrice({ open: null }),
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
