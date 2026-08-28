import { useEffect, useMemo, useRef, useState } from "react";
import type { PointerEvent } from "react";
import type { GeoData } from "../types/stat";
import { countryName, logScale, topDomains, totalGeoRequests } from "../lib/stat-utils";

interface Feature {
  type: "Feature";
  properties?: { ISO_A2?: string; iso_a2?: string; name?: string; center?: [number, number] };
  geometry: { type: "Polygon" | "MultiPolygon"; coordinates: number[][][] | number[][][][] };
}

interface FeatureCollection {
  type: "FeatureCollection";
  features: Feature[];
}

import worldGeo from "../data/world.geo.json";

const world = worldGeo as FeatureCollection;

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

function coordinatesForFeature(feature: Feature): number[][][] {
  if (feature.geometry.type === "Polygon") return feature.geometry.coordinates as number[][][];
  return (feature.geometry.coordinates as number[][][][]).flat();
}

function pointsForPolygon(polygon: number[][], rotation: number, width: number, height: number): string {
  return polygon.map(([lon, lat]) => {
    const point = project(lon, lat, rotation, width, height);
    return `${point.x.toFixed(1)},${point.y.toFixed(1)}`;
  }).join(" ");
}

function featureCode(feature: Feature): string {
  return (feature.properties?.ISO_A2 ?? feature.properties?.iso_a2 ?? "").toUpperCase();
}

function WebGLSphere({ reducedMotion, interacting }: { reducedMotion: boolean; interacting: boolean }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    let gl: WebGLRenderingContext | null = null;
    try {
      gl = canvas.getContext("webgl", { alpha: true, antialias: true });
    } catch {
      return;
    }
    if (!gl) return;
    const vertex = gl.createShader(gl.VERTEX_SHADER);
    const fragment = gl.createShader(gl.FRAGMENT_SHADER);
    if (!vertex || !fragment) return;
    gl.shaderSource(vertex, "attribute vec2 position; void main(){gl_Position=vec4(position,0.0,1.0);}");
    gl.compileShader(vertex);
    gl.shaderSource(fragment, `precision mediump float; uniform float u_time; void main(){ vec2 p=gl_FragCoord.xy/vec2(${Math.max(canvas.clientWidth, 1)}.0,${Math.max(canvas.clientHeight, 1)}.0)*2.0-1.0; p.x*=1.35; float d=dot(p,p); if(d>1.0) discard; float z=sqrt(1.0-d); vec3 n=normalize(vec3(p,z)); float light=max(dot(n,normalize(vec3(-0.45,0.55,1.0))),0.0); float lon=atan(n.y,n.x); float lat=asin(n.z); float grid=(smoothstep(0.018,0.0,abs(sin(lon*10.0+u_time*0.08)))+smoothstep(0.018,0.0,abs(sin(lat*8.0))))*0.12; vec3 base=vec3(0.025,0.17,0.15)+vec3(0.02,0.32,0.27)*light+vec3(0.02,0.35,0.30)*grid; gl_FragColor=vec4(base,0.82); }`);
    gl.compileShader(fragment);
    const program = gl.createProgram();
    if (!program) return;
    gl.attachShader(program, vertex);
    gl.attachShader(program, fragment);
    gl.linkProgram(program);
    const buffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]), gl.STATIC_DRAW);
    gl.useProgram(program);
    const position = gl.getAttribLocation(program, "position");
    gl.enableVertexAttribArray(position);
    gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);
    const timeLocation = gl.getUniformLocation(program, "u_time");
    let frame = 0;
    const started = performance.now();
    const draw = (now: number) => {
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const width = Math.max(1, Math.floor(canvas.clientWidth * ratio));
      const height = Math.max(1, Math.floor(canvas.clientHeight * ratio));
      if (canvas.width !== width || canvas.height !== height) {
        canvas.width = width;
        canvas.height = height;
        gl.viewport(0, 0, width, height);
      }
      gl.clearColor(0, 0, 0, 0);
      gl.clear(gl.COLOR_BUFFER_BIT);
      gl.uniform1f(timeLocation, reducedMotion || interacting ? 0 : (now - started) / 1000);
      gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
      if (!reducedMotion) frame = requestAnimationFrame(draw);
    };
    draw(started);
    return () => cancelAnimationFrame(frame);
  }, [interacting, reducedMotion]);
  return <canvas ref={canvasRef} className="globe-webgl" aria-hidden="true" />;
}

export function GlobeOverview({ geo, reducedMotion }: { geo: GeoData; reducedMotion: boolean }) {
  const [rotation, setRotation] = useState(0);
  const [hovered, setHovered] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [interacting, setInteracting] = useState(false);
  const [size, setSize] = useState({ width: 720, height: 420 });
  const [webgl, setWebgl] = useState(false);
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
    if (reducedMotion) return;
    let frame = 0;
    const tick = () => {
      if (!interactionRef.current) setRotation((value) => (value + 0.05) % 360);
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [reducedMotion]);

  const maximum = Math.max(...Object.values(geo), 1);
  const countries = Object.keys(geo).length;
  const total = totalGeoRequests(geo);
  const ranking = topDomains(Object.entries(geo).map(([domain, requests]) => ({ domain, requests, errors: 0, request_bytes: 0, response_bytes: 0 })), 10);
  const selectedCode = selected ?? hovered;
  const selectedValue = selectedCode ? geo[selectedCode] ?? 0 : 0;
  const selectedIndex = ranking.findIndex((item) => item.domain === selectedCode);

  const renderedFeatures = useMemo(() => world.features.map((feature) => {
    const code = featureCode(feature);
    const polygons = coordinatesForFeature(feature);
    const points = polygons.map((polygon) => pointsForPolygon(polygon, rotation, size.width, size.height));
    const center = feature.properties?.center ?? [0, 0];
    const centerPoint = project(center[0], center[1], rotation, size.width, size.height);
    return { feature, code, points, centerPoint };
  }), [rotation, size]);

  const setPointerState = (value: boolean) => {
    interactionRef.current = value;
    setInteracting(value);
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
      <div className={`globe-map ${webgl && !reducedMotion ? "is-webgl" : "is-static"}`} ref={mapRef} onPointerEnter={() => setPointerState(true)} onPointerLeave={() => { setPointerState(false); setHovered(null); dragRef.current = null; }} onPointerDown={startDrag} onPointerMove={moveDrag} onPointerUp={stopDrag} onPointerCancel={stopDrag}>
        {webgl && !reducedMotion && <WebGLSphere reducedMotion={reducedMotion} interacting={interacting} />}
        <svg className="globe-svg" viewBox={`0 0 ${size.width} ${size.height}`} role="img" aria-label="全球请求来源地图">
          <ellipse cx={size.width / 2} cy={size.height / 2} rx={Math.min(size.width, size.height) * 0.43} ry={Math.min(size.width, size.height) * 0.43} className="globe-outline" />
          {renderedFeatures.map(({ feature, code, points, centerPoint }) => {
            const intensity = logScale(geo[code] ?? 0, maximum);
            const isActive = code === selectedCode;
            const barHeight = intensity > 0 ? 5 + intensity * 42 : 0;
            return (
              <g key={code || feature.properties?.name} className={isActive ? "country-group active" : "country-group"}>
                {points.map((polygon, index) => <polygon key={`${code}-${index}`} points={polygon} fill={`rgba(98, 232, 193, ${0.12 + intensity * 0.7})`} stroke={isActive ? "#f9ffdb" : "rgba(158,255,224,.35)"} strokeWidth={isActive ? 1.8 : 0.65} tabIndex={0} role="button" aria-label={`${countryName(code)} ${formatCount(geo[code] ?? 0)} 次请求`} onFocus={() => setHovered(code)} onBlur={() => setHovered(null)} onPointerMove={() => setHovered(code)} onPointerLeave={() => { if (!selected) setHovered(null); }} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); setSelected(code); } }} onClick={() => setSelected(code)} />)}
                {centerPoint.visible && barHeight > 0 && <line x1={centerPoint.x} y1={centerPoint.y} x2={centerPoint.x} y2={centerPoint.y - barHeight} className="country-bar" />}
              </g>
            );
          })}
        </svg>
        {selectedCode && <div className="globe-tooltip" style={{ left: "50%", top: "12%" }}><strong>{countryName(selectedCode)}</strong><span>{formatCount(selectedValue)} 次累计请求</span><span>{total ? `${((selectedValue / total) * 100).toFixed(2)}% 全球占比` : "0% 全球占比"}</span>{selectedIndex >= 0 && <small>Top {selectedIndex + 1}</small>}</div>}
        {!countries && <div className="globe-no-data">暂无 GEO 数据</div>}
      </div>
      <aside className="geo-summary">
        <div className="geo-stat-grid">
          <div><span>覆盖国家</span><strong>{formatCount(countries)}</strong></div>
          <div><span>累计请求</span><strong>{formatCount(total)}</strong></div>
        </div>
        <div className="geo-heading"><span>Top 10 来源</span><span>累计总览</span></div>
        <div className="geo-ranking">
          {ranking.length ? ranking.map((item, index) => <button type="button" className={item.domain === selectedCode ? "geo-rank selected" : "geo-rank"} key={item.domain} onClick={() => setSelected(item.domain)}><span className="rank-number">{String(index + 1).padStart(2, "0")}</span><span className="rank-country"><strong>{countryName(item.domain)}</strong><small>{item.domain}</small></span><span className="rank-value">{formatCount(item.requests)}</span></button>) : <div className="geo-empty">等待累计请求数据</div>}
        </div>
        <p className="geo-note">地球数据来自 <code>/api/geo</code>，不随时间窗口切换；WebGL 不可用或开启减弱动画时显示静态地图。</p>
      </aside>
    </div>
  );
}

function formatCount(value: number): string {
  return new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 0 }).format(value);
}
