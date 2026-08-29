import { Component, lazy, Suspense, useEffect, useMemo, useRef, useState, type ErrorInfo, type ReactNode } from "react";
import type { PointerEvent } from "react";
import type { GeoData } from "../types/stat";
import { countryName, logScale, topDomains, totalGeoRequests } from "../lib/stat-utils";
import type { GlobeCountry, GlobeGeometry } from "./globe-types";

import worldGeo from "../data/world.geo.json";

interface Feature {
  type: "Feature";
  properties?: {
    ISO_A2?: string;
    ISO_A2_EH?: string;
    WB_A2?: string;
    iso_a2?: string;
    NAME?: string;
    NAME_ZH?: string;
    name?: string;
    LABEL_X?: number;
    LABEL_Y?: number;
    center?: [number, number];
  };
  geometry: GlobeGeometry;
}

interface FeatureCollection {
  type: "FeatureCollection";
  features: Feature[];
}

const world = worldGeo as FeatureCollection;
const LazyGlobeCanvas = lazy(() => import("./GlobeCanvas").then((module) => ({ default: module.GlobeCanvas })));

function webGLAvailable(): boolean {
  if (typeof document === "undefined") return false;
  try {
    const canvas = document.createElement("canvas");
    return Boolean(canvas.getContext("webgl") || canvas.getContext("experimental-webgl"));
  } catch {
    return false;
  }
}

function project(lon: number, lat: number, rotation: number, width: number, height: number) {
  const longitude = ((lon - rotation + 540) % 360) - 180;
  const latitude = Math.max(-89, Math.min(89, lat));
  const phi = latitude * Math.PI / 180;
  const lambda = longitude * Math.PI / 180;
  const visible = Math.cos(phi) * Math.cos(lambda);
  const radius = Math.min(width, height) * 0.43;
  return {
    x: width / 2 + radius * Math.cos(phi) * Math.sin(lambda),
    y: height / 2 - radius * Math.sin(phi),
    visible: visible > -0.06,
  };
}

function coordinatesForFeature(feature: GlobeCountry): number[][][] {
  if (feature.geometry.type === "Polygon") return feature.geometry.coordinates;
  return feature.geometry.coordinates.flat();
}

function pointsForPolygon(polygon: number[][], rotation: number, width: number, height: number): string {
  return polygon.map(([lon, lat]) => {
    const point = project(lon, lat, rotation, width, height);
    return `${point.x.toFixed(1)},${point.y.toFixed(1)}`;
  }).join(" ");
}

function featureCode(feature: Feature): string {
  const candidates = [feature.properties?.ISO_A2, feature.properties?.ISO_A2_EH, feature.properties?.WB_A2, feature.properties?.iso_a2];
  return candidates.find((value) => typeof value === "string" && /^[a-z]{2}$/i.test(value))?.toUpperCase() ?? "";
}

function featureCenter(feature: Feature): [number, number] {
  if (feature.properties?.center) return feature.properties.center;
  const longitude = feature.properties?.LABEL_X;
  const latitude = feature.properties?.LABEL_Y;
  return Number.isFinite(longitude) && Number.isFinite(latitude) ? [longitude as number, latitude as number] : [0, 0];
}

function ringArea(ring: number[][]): number {
  return ring.reduce((area, [x, y], index) => {
    const [nextX, nextY] = ring[(index + 1) % ring.length];
    return area + x * nextY - nextX * y;
  }, 0) / 2;
}

function normalizeRing(ring: number[][], clockwise: boolean): number[][] {
  const isClockwise = ringArea(ring) < 0;
  return isClockwise === clockwise ? ring : [...ring].reverse();
}

function normalizeGeometry(geometry: GlobeGeometry): GlobeGeometry {
  if (geometry.type === "Polygon") {
    return { type: "Polygon", coordinates: geometry.coordinates.map((ring, index) => normalizeRing(ring, index === 0)) };
  }
  return { type: "MultiPolygon", coordinates: geometry.coordinates.map((polygon) => polygon.map((ring, index) => normalizeRing(ring, index === 0))) };
}

const preparedCountries = world.features.map((feature, index) => {
  const code = featureCode(feature);
  return {
    id: `${code || "unknown"}-${index}`,
    code,
    center: featureCenter(feature),
    geometry: normalizeGeometry(feature.geometry),
    properties: { name: feature.properties?.NAME_ZH ?? feature.properties?.NAME ?? feature.properties?.name },
  };
});

interface GlobeErrorBoundaryProps {
  children: ReactNode;
  onError: () => void;
}

interface GlobeErrorBoundaryState {
  hasError: boolean;
}

class GlobeErrorBoundary extends Component<GlobeErrorBoundaryProps, GlobeErrorBoundaryState> {
  state: GlobeErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): GlobeErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(_error: Error, _errorInfo: ErrorInfo) {
    this.props.onError();
  }

  render() {
    return this.state.hasError ? null : this.props.children;
  }
}

function StaticGlobe({
  countries,
  geo,
  maximum,
  rotation,
  size,
  selectedCode,
  selected,
  onHover,
  onSelect,
}: {
  countries: GlobeCountry[];
  geo: GeoData;
  maximum: number;
  rotation: number;
  size: { width: number; height: number };
  selectedCode: string | null;
  selected: string | null;
  onHover: (code: string | null) => void;
  onSelect: (code: string) => void;
}) {
  const renderedFeatures = useMemo(() => countries.map((country) => {
    const polygons = coordinatesForFeature(country);
    const points = polygons.map((polygon) => pointsForPolygon(polygon, rotation, size.width, size.height));
    const centerPoint = project(country.center[0], country.center[1], rotation, size.width, size.height);
    return { country, points, centerPoint };
  }), [countries, rotation, size]);

  return (
    <svg className="globe-svg" viewBox={`0 0 ${size.width} ${size.height}`} role="img" aria-label="全球请求来源地图">
      <ellipse cx={size.width / 2} cy={size.height / 2} rx={Math.min(size.width, size.height) * 0.43} ry={Math.min(size.width, size.height) * 0.43} className="globe-outline" />
      {renderedFeatures.map(({ country, points, centerPoint }) => {
        const intensity = logScale(country.requests, maximum);
        const isActive = country.code === selectedCode;
        const barHeight = intensity > 0 ? 5 + intensity * 42 : 0;
        return (
          <g
            key={country.id}
            className={isActive ? "country-group active" : "country-group"}
            tabIndex={country.code ? 0 : undefined}
            role={country.code ? "button" : undefined}
            aria-label={country.code ? `${countryName(country.code)} ${formatCount(geo[country.code] ?? 0)} 次请求` : undefined}
            onFocus={() => { if (country.code) onHover(country.code); }}
            onBlur={() => onHover(null)}
            onPointerMove={() => { if (country.code) onHover(country.code); }}
            onPointerLeave={() => { if (!selected) onHover(null); }}
            onKeyDown={(event) => {
              if (country.code && (event.key === "Enter" || event.key === " ")) {
                event.preventDefault();
                onSelect(country.code);
              }
            }}
            onClick={() => { if (country.code) onSelect(country.code); }}
          >
            {points.map((polygon, index) => <polygon key={`${country.id}-${index}`} points={polygon} fill={intensity > 0 ? `rgba(55, 218, 154, ${0.12 + intensity * 0.38})` : "rgba(96, 142, 84, 0.42)"} stroke={isActive ? "#ffe09a" : "rgba(228,244,255,.24)"} strokeWidth={isActive ? 1.8 : 0.55} />)}
            {centerPoint.visible && barHeight > 0 && <line x1={centerPoint.x} y1={centerPoint.y} x2={centerPoint.x} y2={centerPoint.y - barHeight} className="country-bar" />}
          </g>
        );
      })}
    </svg>
  );
}

export function GlobeOverview({ geo, reducedMotion }: { geo: GeoData; reducedMotion: boolean }) {
  const [rotation, setRotation] = useState(0);
  const [hovered, setHovered] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [size, setSize] = useState({ width: 720, height: 420 });
  const [webgl, setWebgl] = useState(false);
  const [globeFailed, setGlobeFailed] = useState(false);
  const mapRef = useRef<HTMLDivElement>(null);
  const interactionRef = useRef(false);
  const dragRef = useRef<{ startX: number; startRotation: number } | null>(null);

  useEffect(() => setWebgl(webGLAvailable()), []);
  useEffect(() => {
    const element = mapRef.current;
    if (!element) return;
    const update = () => setSize({ width: Math.max(280, element.clientWidth), height: Math.max(260, element.clientHeight) });
    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    if (reducedMotion || (webgl && !globeFailed)) return;
    let frame = 0;
    const tick = () => {
      if (!interactionRef.current) setRotation((value) => (value + 0.05) % 360);
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [globeFailed, reducedMotion, webgl]);

  const maximum = Math.max(...Object.values(geo), 1);
  const countriesCount = Object.keys(geo).length;
  const total = totalGeoRequests(geo);
  const ranking = topDomains(Object.entries(geo).map(([domain, requests]) => ({ domain, requests, errors: 0, request_bytes: 0, response_bytes: 0 })), 10);
  const selectedCode = selected ?? hovered;
  const selectedValue = selectedCode ? geo[selectedCode] ?? 0 : 0;
  const selectedIndex = ranking.findIndex((item) => item.domain === selectedCode);
  const countries = useMemo<GlobeCountry[]>(() => preparedCountries.map((country) => ({
    ...country,
    requests: country.code ? geo[country.code] ?? 0 : 0,
  })), [geo]);
  const activePoints = useMemo(() => countries.filter((country) => country.requests > 0).map((country) => ({
    id: country.id,
    code: country.code,
    requests: country.requests,
    lat: country.center[1],
    lng: country.center[0],
  })), [countries]);
  const shouldRenderWebgl = webgl && !reducedMotion && !globeFailed;

  const setPointerState = (value: boolean) => {
    interactionRef.current = value;
  };

  const startDrag = (event: PointerEvent<HTMLDivElement>) => {
    dragRef.current = { startX: event.clientX, startRotation: rotation };
    event.currentTarget.setPointerCapture?.(event.pointerId);
    setPointerState(true);
  };

  const moveDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (!dragRef.current) return;
    setRotation((dragRef.current.startRotation + (event.clientX - dragRef.current.startX) * 0.32 + 360) % 360);
  };

  const stopDrag = (event: PointerEvent<HTMLDivElement>) => {
    dragRef.current = null;
    event.currentTarget.releasePointerCapture?.(event.pointerId);
    setPointerState(false);
  };

  return (
    <div className="globe-layout">
      <div
        className={`globe-map ${shouldRenderWebgl ? "is-webgl" : "is-static"}`}
        ref={mapRef}
        onPointerEnter={shouldRenderWebgl ? undefined : () => setPointerState(true)}
        onPointerLeave={shouldRenderWebgl ? undefined : () => { setPointerState(false); setHovered(null); dragRef.current = null; }}
        onPointerDown={shouldRenderWebgl ? undefined : startDrag}
        onPointerMove={shouldRenderWebgl ? undefined : moveDrag}
        onPointerUp={shouldRenderWebgl ? undefined : stopDrag}
        onPointerCancel={shouldRenderWebgl ? undefined : stopDrag}
      >
        {shouldRenderWebgl ? (
          <GlobeErrorBoundary onError={() => setGlobeFailed(true)}>
            <Suspense fallback={<div className="globe-loading">正在加载全球来源图层…</div>}>
              <LazyGlobeCanvas countries={countries} points={activePoints} maximum={maximum} width={size.width} height={size.height} selectedCode={selectedCode} onCountryHover={setHovered} onCountrySelect={setSelected} onInteractionChange={setPointerState} />
            </Suspense>
          </GlobeErrorBoundary>
        ) : (
          <StaticGlobe countries={countries} geo={geo} maximum={maximum} rotation={rotation} size={size} selectedCode={selectedCode} selected={selected} onHover={setHovered} onSelect={setSelected} />
        )}
        {selectedCode && <div className="globe-tooltip" style={{ left: "50%", top: "10%" }}><strong>{countryName(selectedCode)}</strong><span>{formatCount(selectedValue)} 次累计请求</span><span>{total ? `${((selectedValue / total) * 100).toFixed(2)}% 全球占比` : "0% 全球占比"}</span>{selectedIndex >= 0 && <small>Top {selectedIndex + 1}</small>}</div>}
        {!countriesCount && <div className="globe-no-data">暂无 GEO 数据</div>}
      </div>
      <aside className="geo-summary">
        <div className="geo-stat-grid">
          <div><span>覆盖国家</span><strong>{formatCount(countriesCount)}</strong></div>
          <div><span>累计请求</span><strong>{formatCount(total)}</strong></div>
        </div>
        <div className="geo-heading"><span>Top 10 来源</span><span>累计总览</span></div>
        <div className="geo-ranking">
          {ranking.length ? ranking.map((item, index) => <button type="button" className={item.domain === selectedCode ? "geo-rank selected" : "geo-rank"} key={item.domain} onClick={() => setSelected(item.domain)}><span className="rank-number">{String(index + 1).padStart(2, "0")}</span><span className="rank-country"><strong>{countryName(item.domain)}</strong><small>{item.domain}</small></span><span className="rank-value">{formatCount(item.requests)}</span></button>) : <div className="geo-empty">等待累计请求数据</div>}
        </div>
        <p className="geo-note">地图边界数据来源：<a href="https://github.com/nvkelso/natural-earth-vector/blob/master/geojson/ne_110m_admin_0_countries.geojson" target="_blank" rel="noreferrer">natural-earth-vector</a></p>
      </aside>
    </div>
  );
}

function formatCount(value: number): string {
  return new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 0 }).format(value);
}
