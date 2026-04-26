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
