import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";

const response = {
  total: 3,
  api: 2,
  static: 1,
  fail: 0,
  today: 3,
  meta: { range: "1h", bucket_seconds: 60, capabilities: {} },
  summary: {
    total_requests: 3,
    frontend_requests: 1,
    backend_requests: 2,
    error_requests: 0,
    rps: 0.001,
    error_rate: 0,
    status: { "1xx": 0, "2xx": 3, "3xx": 0, "4xx": 0, "5xx": 0 },
    latency: { avg_ms: 2, p95_ms: 5, max_ms: 8 },
    total_traffic: { request_bytes: 10, response_bytes: 20, total_bytes: 30 },
  },
  series: { requests: [], traffic: [], system: [], process: [] },
  connections: {},
  domains: [{ domain: "api.example.com", requests: 3, errors: 0, request_bytes: 10, response_bytes: 20 }],
  domain_series: [{ timestamp: "2026-08-28T12:00:00Z", requests: 3, errors: 0, request_bytes: 10, response_bytes: 20 }],
};

describe("Stat Web page", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation((input: string) => Promise.resolve({
      ok: true,
      json: async () => input.includes("/api/geo") ? { CN: 3 } : response,
    })));
    vi.stubGlobal("ResizeObserver", class { observe() {} disconnect() {} });
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    Object.defineProperty(HTMLElement.prototype, "clientWidth", { configurable: true, value: 640 });
    Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, value: 240 });
    window.localStorage.clear();
  });

  it("loads the dashboard and switches time windows", async () => {
    render(<App />);
    expect(await screen.findByText("总请求")).toBeInTheDocument();
    expect(screen.getByText("累计统计")).toBeInTheDocument();
    expect(screen.getByText("总请求数")).toBeInTheDocument();
    expect(screen.getByText("失败请求数")).toBeInTheDocument();
    expect(screen.getByText("前端请求数")).toBeInTheDocument();
    expect(screen.getByText("后端请求数")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "24h" }));
    await waitFor(() => expect(vi.mocked(fetch).mock.calls.some(([input]) => String(input).includes("range=24h"))).toBe(true));
    expect(screen.getByText("全球请求来源")).toBeInTheDocument();
  });

  it("loads a selected domain detail series", async () => {
    render(<App />);
    await screen.findByRole("button", { name: "api.example.com" });
    fireEvent.click(screen.getByRole("button", { name: "api.example.com" }));
    expect(await screen.findByText("SELECTED DOMAIN")).toBeInTheDocument();
    expect(screen.getAllByText("api.example.com").length).toBeGreaterThanOrEqual(2);
  });
});
