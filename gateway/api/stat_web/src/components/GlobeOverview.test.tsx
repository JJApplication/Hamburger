import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GlobeOverview } from "./GlobeOverview";

describe("GEO overview", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", class { observe() {} disconnect() {} });
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
  });
  afterEach(() => vi.restoreAllMocks());

  it("shows zero-count countries and keeps unknown ISO codes in the ranking", () => {
    const { container } = render(<GlobeOverview geo={{ CN: 10, ZZ: 2 }} reducedMotion />);
    expect(screen.getByRole("button", { name: /美国 0 次请求/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /国家 ZZ/ })).toBeInTheDocument();
    expect(container.querySelector(".globe-webgl")).toBeNull();
  });

  it("renders a complete empty fallback", () => {
    render(<GlobeOverview geo={{}} reducedMotion />);
    expect(screen.getByText("暂无 GEO 数据")).toBeInTheDocument();
    expect(screen.getByText("等待累计请求数据")).toBeInTheDocument();
  });
});
