"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  createChart,
  IChartApi,
  ColorType,
  CandlestickData,
  LineData,
  HistogramData,
  Time,
  CandlestickSeries,
  LineSeries,
  HistogramSeries,
} from "lightweight-charts";
import {
  Holding,
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
        attributionLogo: false,
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
    } as Parameters<typeof createChart>[1]);

    chartRef.current = chart;

    const showVolume = hasVolumeData(data.prices);

    if (chartMode === "candlestick") {
      const candleSeries = chart.addSeries(CandlestickSeries, {
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
      const lineSeries = chart.addSeries(LineSeries, {
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
      const volumeSeries = chart.addSeries(HistogramSeries, {
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
