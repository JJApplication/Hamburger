import { createElement, forwardRef, useImperativeHandle } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GlobeOverview } from "./GlobeOverview";

interface MockGlobeControls {
  autoRotate: boolean;
  enableDamping: boolean;
  dampingFactor: number;
  autoRotateSpeed: number;
  enablePan: boolean;
}

interface MockGlobeMethods {
  pointOfView: ReturnType<typeof vi.fn>;
  controls: () => MockGlobeControls;
  renderer: () => { setPixelRatio: ReturnType<typeof vi.fn> };
  pauseAnimation: ReturnType<typeof vi.fn>;
  resumeAnimation: ReturnType<typeof vi.fn>;
}

const mockControls: MockGlobeControls = {
  autoRotate: false,
  enableDamping: false,
  dampingFactor: 0,
  autoRotateSpeed: 0,
  enablePan: true,
};
const mockRenderer = { setPixelRatio: vi.fn() };
const mockGlobeMethods: MockGlobeMethods = {
  pointOfView: vi.fn(),
  controls: () => mockControls,
  renderer: () => mockRenderer,
  pauseAnimation: vi.fn(),
  resumeAnimation: vi.fn(),
};

vi.mock("react-globe.gl", () => {
  const MockGlobe = forwardRef<MockGlobeMethods, Record<string, unknown>>((props, ref) => {
    useImperativeHandle(ref, () => mockGlobeMethods, []);
    const polygons = props.polygonsData as unknown[];
    const points = props.pointsData as unknown[];
    const china = polygons.find((country) => (country as { code?: string }).code === "CN");
    const france = polygons.find((country) => (country as { code?: string }).code === "FR");
    const chinaPoint = points.find((point) => (point as { code?: string }).code === "CN") as { lat?: number; lng?: number } | undefined;
    const polygonColor = props.polygonCapColor as ((country: object) => string);
    const polygonAltitude = props.polygonAltitude as ((country: object) => number);
    const pointAltitude = props.pointAltitude as ((point: object) => number);
    const pointRadius = props.pointRadius as ((point: object) => number);
    return createElement("div", {
      "data-testid": "mock-globe",
      "data-polygon-count": polygons.length,
      "data-point-count": points.length,
      "data-cn-color": china ? polygonColor(china) : "",
      "data-cn-altitude": china ? polygonAltitude(china) : "",
      "data-cn-lat": chinaPoint?.lat ?? "",
      "data-cn-lng": chinaPoint?.lng ?? "",
      "data-cn-point-altitude": chinaPoint ? pointAltitude(chinaPoint) : "",
      "data-cn-point-radius": chinaPoint ? pointRadius(chinaPoint) : "",
      "data-ring-count": (props.ringsData as unknown[]).length,
      "data-has-france": france ? "true" : "false",
      "data-globe-image": props.globeImageUrl as string,
      "data-bump-image": props.bumpImageUrl as string,
      "data-graticules": String(props.showGraticules),
    }, createElement("button", {
      type: "button",
      "data-testid": "mock-country-cn",
      onClick: () => (props.onPolygonClick as ((country: object) => void))(china as object),
    }, "mock China"));
  });
  return { default: MockGlobe };
});

describe("GEO overview", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", class { observe() {} disconnect() {} });
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    mockControls.autoRotate = false;
    vi.clearAllMocks();
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

  it("uses the static map when WebGL is unavailable", () => {
    render(<GlobeOverview geo={{ CN: 10 }} reducedMotion={false} />);
    expect(screen.queryByTestId("mock-globe")).toBeNull();
    expect(screen.getByRole("img", { name: "全球请求来源地图" })).toBeInTheDocument();
  });

  it("passes local country polygons and active request points to the WebGL globe", async () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({} as WebGLRenderingContext);
    render(<GlobeOverview geo={{ CN: 10, US: 2, ZZ: 1 }} reducedMotion={false} />);
    const globe = await screen.findByTestId("mock-globe");

    expect(globe).toHaveAttribute("data-polygon-count", "177");
    expect(globe).toHaveAttribute("data-point-count", "2");
    expect(globe).toHaveAttribute("data-cn-color", "rgba(48, 224, 157, 0.390)");
    expect(Number(globe.getAttribute("data-cn-altitude"))).toBeGreaterThan(0);
    expect(Number(globe.getAttribute("data-cn-altitude"))).toBeLessThan(0.01);
    expect(Number(globe.getAttribute("data-cn-lat"))).toBeGreaterThan(30);
    expect(Number(globe.getAttribute("data-cn-lng"))).toBeGreaterThan(90);
    expect(Number(globe.getAttribute("data-cn-point-altitude"))).toBeGreaterThanOrEqual(0.07);
    expect(Number(globe.getAttribute("data-cn-point-radius"))).toBeGreaterThanOrEqual(0.22);
    expect(globe).toHaveAttribute("data-ring-count", "2");
    expect(globe).toHaveAttribute("data-has-france", "true");
    expect(globe.getAttribute("data-globe-image")).toContain("earth-day.jpg");
    expect(globe.getAttribute("data-bump-image")).toContain("earth-topology.png");
    expect(globe).toHaveAttribute("data-graticules", "false");
    await waitFor(() => expect(mockRenderer.setPixelRatio).toHaveBeenCalledWith(1));
  });

  it("pauses the globe auto-rotation while the canvas is being used", async () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({} as WebGLRenderingContext);
    render(<GlobeOverview geo={{ CN: 10 }} reducedMotion={false} />);
    const globe = await screen.findByTestId("mock-globe");

    await waitFor(() => expect(mockControls.autoRotate).toBe(true));
    fireEvent.pointerEnter(globe);
    expect(mockControls.autoRotate).toBe(false);
    fireEvent.pointerLeave(globe);
    expect(mockControls.autoRotate).toBe(true);
  });

  it("keeps globe clicks connected to the country detail tooltip", async () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({} as WebGLRenderingContext);
    render(<GlobeOverview geo={{ CN: 10 }} reducedMotion={false} />);
    const globe = await screen.findByTestId("mock-globe");

    fireEvent.click(screen.getByTestId("mock-country-cn"));
    expect(screen.getByText("10 次累计请求")).toBeInTheDocument();
    expect(globe).toBeInTheDocument();
  });
});
