import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import Globe, { type GlobeMethods } from "react-globe.gl";
import { MeshPhongMaterial } from "three";
import type { GlobeCountry, GlobePoint } from "./globe-types";

const earthImageUrl = new URL("../../node_modules/three-globe/example/img/earth-day.jpg", import.meta.url).href;
const earthBumpUrl = new URL("../../node_modules/three-globe/example/img/earth-topology.png", import.meta.url).href;

interface GlobeCanvasProps {
  countries: GlobeCountry[];
  points: GlobePoint[];
  maximum: number;
  width: number;
  height: number;
  selectedCode: string | null;
  onCountryHover: (code: string | null) => void;
  onCountrySelect: (code: string) => void;
  onInteractionChange: (interacting: boolean) => void;
}

function countryFromObject(value: object | null): GlobeCountry | null {
  return value ? value as GlobeCountry : null;
}

function pointFromObject(value: object | null): GlobePoint | null {
  return value ? value as GlobePoint : null;
}

function countryColor(intensity: number): string {
  const amount = Math.max(0, Math.min(1, intensity));
  const alpha = amount > 0 ? 0.09 + amount * 0.3 : 0.025;
  return `rgba(48, 224, 157, ${alpha.toFixed(3)})`;
}

export function GlobeCanvas({
  countries,
  points,
  maximum,
  width,
  height,
  selectedCode,
  onCountryHover,
  onCountrySelect,
  onInteractionChange,
}: GlobeCanvasProps) {
  const globeRef = useRef<GlobeMethods | undefined>(undefined);
  const containerRef = useRef<HTMLDivElement>(null);
  const mountedRef = useRef(false);
  const [ready, setReady] = useState(false);
  const [visible, setVisible] = useState(true);
  const [pageHidden, setPageHidden] = useState(() => typeof document !== "undefined" && document.hidden);

  const globeMaterial = useMemo(() => new MeshPhongMaterial({
    color: 0xffffff,
    emissive: 0x010305,
    emissiveIntensity: 0.08,
    shininess: 12,
    specular: 0x68869d,
    bumpScale: 0.65,
  }), []);

  const intensityFor = (requests: number) => {
    if (requests <= 0 || maximum <= 0) return 0;
    return Math.log1p(requests) / Math.log1p(maximum);
  };

  useEffect(() => () => globeMaterial.dispose(), [globeMaterial]);

  useLayoutEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    const element = containerRef.current;
    if (!element || typeof IntersectionObserver === "undefined") return;
    const observer = new IntersectionObserver(([entry]) => setVisible(entry.isIntersecting));
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const updateVisibility = () => setPageHidden(document.hidden);
    document.addEventListener("visibilitychange", updateVisibility);
    return () => document.removeEventListener("visibilitychange", updateVisibility);
  }, []);

  useEffect(() => {
    if (ready) return;
    const timer = window.setTimeout(() => setReady(true), 500);
    return () => window.clearTimeout(timer);
  }, [ready]);

  useEffect(() => {
    const globe = globeRef.current;
    if (!globe || !ready) return;
    const controls = globe.controls();
    controls.enablePan = false;
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;
    controls.autoRotateSpeed = 0.22;
    controls.autoRotate = visible && !pageHidden;
    const renderer = globe.renderer();
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.5));
    if (visible && !pageHidden) {
      globe.resumeAnimation();
    } else {
      globe.pauseAnimation();
    }
  }, [pageHidden, ready, visible]);

  const handleReady = () => {
    if (!mountedRef.current) return;
    const globe = globeRef.current;
    if (globe) {
      globe.pointOfView({ lat: 20, lng: 105, altitude: 2.15 }, 0);
      globe.controls().autoRotate = visible && !pageHidden;
    }
    setReady(true);
  };

  const handleInteractionStart = () => {
    onInteractionChange(true);
    const globe = globeRef.current;
    if (globe) globe.controls().autoRotate = false;
  };

  const handleInteractionEnd = () => {
    onInteractionChange(false);
    const globe = globeRef.current;
    if (globe) globe.controls().autoRotate = visible && !pageHidden;
  };

  return (
    <div
      ref={containerRef}
      className="globe-canvas-shell"
      role="img"
      aria-label="可交互的全球请求来源地球"
      onPointerEnter={handleInteractionStart}
      onPointerLeave={() => {
        handleInteractionEnd();
        onCountryHover(null);
      }}
      onPointerDown={handleInteractionStart}
      onPointerUp={handleInteractionEnd}
      onPointerCancel={handleInteractionEnd}
    >
      <Globe
        ref={globeRef}
        width={width}
        height={height}
        backgroundColor="rgba(0,0,0,0)"
        globeImageUrl={earthImageUrl}
        bumpImageUrl={earthBumpUrl}
        globeMaterial={globeMaterial}
        showAtmosphere
        atmosphereColor="#79c9ff"
        atmosphereAltitude={0.12}
        showGraticules={false}
        polygonsData={countries}
        polygonGeoJsonGeometry="geometry"
        polygonCapColor={(country) => {
          const item = country as GlobeCountry;
          return item.code === selectedCode ? "rgba(255, 214, 112, 0.56)" : countryColor(intensityFor(item.requests));
        }}
        polygonSideColor={() => "rgba(3, 22, 32, 0.08)"}
        polygonStrokeColor={(country) => {
          const item = country as GlobeCountry;
          if (item.code === selectedCode) return "rgba(255, 230, 157, 0.96)";
          return item.requests > 0 ? "rgba(134, 255, 213, 0.42)" : "rgba(235, 247, 255, 0.14)";
        }}
        polygonAltitude={(country) => {
          const item = country as GlobeCountry;
          if (item.code === selectedCode) return 0.0042;
          return item.requests > 0 ? 0.0008 + intensityFor(item.requests) * 0.0024 : 0.0002;
        }}
        polygonCapCurvatureResolution={2}
        polygonsTransitionDuration={260}
        pointsData={points}
        pointLat="lat"
        pointLng="lng"
        pointColor={(point) => (point as GlobePoint).code === selectedCode ? "#ffd36a" : "#54ffc2"}
        pointAltitude={(point) => 0.07 + intensityFor((point as GlobePoint).requests) * 0.15}
        pointRadius={(point) => 0.22 + intensityFor((point as GlobePoint).requests) * 0.28}
        pointResolution={12}
        pointsMerge={false}
        pointsTransitionDuration={260}
        ringsData={points}
        ringLat="lat"
        ringLng="lng"
        ringColor={(point: object) => () => (point as GlobePoint).code === selectedCode ? "rgba(255, 211, 106, 0.9)" : "rgba(84, 255, 194, 0.72)"}
        ringMaxRadius={(point) => 0.8 + intensityFor((point as GlobePoint).requests) * 1.3}
        ringPropagationSpeed={0.55}
        ringRepeatPeriod={1800}
        onPolygonHover={(country) => onCountryHover(countryFromObject(country)?.code ?? null)}
        onPolygonClick={(country) => {
          const item = countryFromObject(country);
          if (item) onCountrySelect(item.code);
        }}
        onPointHover={(point) => onCountryHover(pointFromObject(point)?.code ?? null)}
        onPointClick={(point) => {
          const item = pointFromObject(point);
          if (item) onCountrySelect(item.code);
        }}
        enablePointerInteraction
        onGlobeReady={handleReady}
      />
    </div>
  );
}
