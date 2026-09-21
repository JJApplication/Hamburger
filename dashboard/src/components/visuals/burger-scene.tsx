"use client";
/* eslint-disable react-hooks/set-state-in-effect */

import { useEffect, useRef, useState } from "react";
import * as THREE from "three";

export function BurgerScene({ compact = false, paused = false }: { compact?: boolean; paused?: boolean }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const pausedRef = useRef(paused);
  const [fallback, setFallback] = useState(false);

  useEffect(() => { pausedRef.current = paused; }, [paused]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    let renderer: THREE.WebGLRenderer;
    try { renderer = new THREE.WebGLRenderer({ canvas, alpha: true, antialias: true }); }
    catch { setFallback(true); return; }
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.shadowMap.enabled = true;
    renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.toneMappingExposure = 1.12;

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(30, 1, 0.1, 100);
    camera.position.set(0, 1.45, 6.65);
    camera.lookAt(0, 0.42, 0);
    const burger = new THREE.Group();
    scene.add(burger);

    const mat = (color: number, roughness: number, metalness = 0) => new THREE.MeshStandardMaterial({ color, roughness, metalness });
    const bunBottom = mat(0xd87525, 0.72);
    const bunTop = mat(0xf1a147, 0.62);
    const bunCrust = mat(0xb95719, 0.8);
    const patty = mat(0x3a1b0d, 0.96);
    const pattyHighlight = mat(0x5a2b12, 0.9);
    const cheese = mat(0xffc72c, 0.42);
    const lettuce = mat(0x2bc66d, 0.84);
    const lettuceCore = mat(0x159d54, 0.9);
    const sesame = mat(0xffe5aa, 0.48);

    const add = (geometry: THREE.BufferGeometry, material: THREE.Material, position: THREE.Vector3, scale?: THREE.Vector3, rotation?: THREE.Euler) => {
      const mesh = new THREE.Mesh(geometry, material);
      mesh.position.copy(position);
      if (scale) mesh.scale.copy(scale);
      if (rotation) mesh.rotation.copy(rotation);
      mesh.castShadow = true; mesh.receiveShadow = true; burger.add(mesh); return mesh;
    };
    const irregularCylinder = (radius: number, height: number, phase: number, amplitude = 0.04) => {
      const geometry = new THREE.CylinderGeometry(radius * 0.98, radius, height, 64, 4, false);
      const position = geometry.attributes.position;
      for (let index = 0; index < position.count; index += 1) {
        const x = position.getX(index);
        const y = position.getY(index);
        const z = position.getZ(index);
        const distance = Math.hypot(x, z);
        if (distance < 0.01) continue;
        const angle = Math.atan2(z, x);
        const edge = 1 + Math.sin(angle * 5 + phase) * amplitude + Math.sin(angle * 11 - phase) * amplitude * 0.38;
        const middleBulge = 1 + Math.cos((y / height) * Math.PI) * 0.025;
        position.setX(index, x * edge * middleBulge);
        position.setZ(index, z * edge * middleBulge);
        position.setY(index, y + Math.sin(angle * 7 + phase) * 0.018);
      }
      position.needsUpdate = true;
      geometry.computeVertexNormals();
      return geometry;
    };
    const cheeseSlice = (y: number, rotationY: number, phase: number) => {
      const geometry = new THREE.BoxGeometry(2.58, 0.1, 2.58, 8, 1, 8);
      const position = geometry.attributes.position;
      for (let index = 0; index < position.count; index += 1) {
        const x = position.getX(index);
        const localY = position.getY(index);
        const z = position.getZ(index);
        const front = THREE.MathUtils.smoothstep(z, 0.52, 1.29);
        const corner = THREE.MathUtils.smoothstep(Math.abs(x), 0.68, 1.29);
        const ripple = (Math.sin(x * 4.2 + phase) + 1) * 0.025;
        const drop = front * (0.06 + corner * 0.24 + ripple);
        position.setY(index, localY - drop);
      }
      position.needsUpdate = true;
      geometry.computeVertexNormals();
      return add(geometry, cheese, new THREE.Vector3(0, y, 0), undefined, new THREE.Euler(0, rotationY, 0));
    };
    const lettuceRuffle = (y: number) => {
      const geometry = new THREE.TorusGeometry(1.37, 0.2, 14, 96);
      const position = geometry.attributes.position;
      for (let index = 0; index < position.count; index += 1) {
        const x = position.getX(index);
        const localY = position.getY(index);
        const z = position.getZ(index);
        const angle = Math.atan2(localY, x);
        const ruffle = 1 + Math.sin(angle * 7 + 0.6) * 0.075 + Math.sin(angle * 13 - 0.4) * 0.025;
        position.setX(index, x * ruffle);
        position.setY(index, localY * ruffle);
        position.setZ(index, z * 0.68 + Math.sin(angle * 9) * 0.045);
      }
      position.needsUpdate = true;
      geometry.computeVertexNormals();
      add(geometry, lettuce, new THREE.Vector3(0, y, 0), undefined, new THREE.Euler(Math.PI / 2, 0, 0));
    };

    const topBun = new THREE.LatheGeometry([
      new THREE.Vector2(0, -0.06), new THREE.Vector2(1.55, -0.02), new THREE.Vector2(1.58, 0.13),
      new THREE.Vector2(1.47, 0.34), new THREE.Vector2(1.2, 0.59), new THREE.Vector2(0.85, 0.78),
      new THREE.Vector2(0.42, 0.92), new THREE.Vector2(0, 0.96),
    ], 96);
    add(topBun, bunTop, new THREE.Vector3(0, 0.82, 0));
    add(new THREE.CylinderGeometry(1.51, 1.55, 0.08, 64), bunCrust, new THREE.Vector3(0, 0.77, 0));

    lettuceRuffle(0.73);
    add(irregularCylinder(1.31, 0.1, 2.1, 0.055), lettuceCore, new THREE.Vector3(0, 0.7, 0));
    add(irregularCylinder(1.43, 0.42, 0.4), pattyHighlight, new THREE.Vector3(0, 0.45, 0));
    cheeseSlice(0.19, 0.4, 0.2);
    add(irregularCylinder(1.4, 0.43, 2.6), patty, new THREE.Vector3(0, -0.08, 0));
    cheeseSlice(-0.35, -0.3, 2.3);

    const bottomBun = new THREE.LatheGeometry([
      new THREE.Vector2(0, -0.5), new THREE.Vector2(0.56, -0.47), new THREE.Vector2(1.08, -0.36),
      new THREE.Vector2(1.42, -0.18), new THREE.Vector2(1.55, 0.04), new THREE.Vector2(1.51, 0.18),
      new THREE.Vector2(0, 0.22),
    ], 96);
    add(bottomBun, bunBottom, new THREE.Vector3(0, -0.58, 0));
    add(new THREE.CylinderGeometry(1.47, 1.51, 0.08, 64), bunCrust, new THREE.Vector3(0, -0.39, 0));

    const seeds: Array<[number, number, number]> = [
      [-0.82, 0.35, -0.35], [-0.52, 0.6, 0.45], [-0.28, 0.25, -0.2], [0.02, 0.68, 0.35],
      [0.28, 0.35, -0.5], [0.56, 0.54, 0.2], [0.84, 0.28, -0.15], [-0.7, -0.08, 0.72],
      [-0.18, -0.38, 0.82], [0.38, -0.18, 0.76], [0.78, -0.34, 0.56],
    ];
    seeds.forEach(([x, twist, z], index) => {
      const radiusSquared = Math.min(1, (x * x + z * z) / 2.38);
      const y = 0.9 + 0.88 * Math.sqrt(1 - radiusSquared);
      add(new THREE.SphereGeometry(0.095, 16, 10), sesame, new THREE.Vector3(x, y, z), new THREE.Vector3(0.54, 0.18, 0.22), new THREE.Euler(0.24, twist + index * 0.32, -0.2));
    });

    const key = new THREE.DirectionalLight(0xffe0bb, 4.1); key.position.set(-3, 6, 4); key.castShadow = true; key.shadow.mapSize.set(1024, 1024); scene.add(key);
    const fill = new THREE.DirectionalLight(0x8bd8ff, 1.8); fill.position.set(4, 2, -3); scene.add(fill);
    scene.add(new THREE.HemisphereLight(0xffe8cd, 0x17243b, 1.4));
    const ground = new THREE.Mesh(new THREE.CircleGeometry(2.2, 64), new THREE.ShadowMaterial({ color: 0x06111f, opacity: 0.34 }));
    ground.rotation.x = -Math.PI / 2; ground.position.y = -1.14; ground.receiveShadow = true; scene.add(ground);

    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const pointer = { x: 0, y: 0 }, target = { x: 0, y: 0 };
    const dragRotation = { x: 0, y: 0 };
    let dragging = false;
    let lastPoint = { x: 0, y: 0 };
    let spin = 0;
    const onPointerDown = (event: PointerEvent) => { dragging = true; lastPoint = { x: event.clientX, y: event.clientY }; canvas.setPointerCapture(event.pointerId); };
    const onPointer = (event: PointerEvent) => {
      if (dragging) {
        dragRotation.y += (event.clientX - lastPoint.x) * 0.012;
        dragRotation.x += (event.clientY - lastPoint.y) * 0.008;
        lastPoint = { x: event.clientX, y: event.clientY };
        return;
      }
      const rect = canvas.getBoundingClientRect(); target.x = ((event.clientX - rect.left) / rect.width - 0.5) * 0.3; target.y = ((event.clientY - rect.top) / rect.height - 0.5) * -0.18;
    };
    const onPointerUp = (event: PointerEvent) => { dragging = false; if (canvas.hasPointerCapture(event.pointerId)) canvas.releasePointerCapture(event.pointerId); };
    canvas.addEventListener("pointermove", onPointer);
    canvas.addEventListener("pointerdown", onPointerDown);
    canvas.addEventListener("pointerup", onPointerUp);
    canvas.addEventListener("pointercancel", onPointerUp);
    let frame = 0; let visible = !document.hidden; let last = performance.now();
    const onVisibility = () => { visible = !document.hidden; };
    document.addEventListener("visibilitychange", onVisibility);
    const resize = () => { const rect = canvas.getBoundingClientRect(); renderer.setSize(rect.width, rect.height, false); camera.aspect = rect.width / Math.max(rect.height, 1); camera.updateProjectionMatrix(); };
    const observer = new ResizeObserver(resize); observer.observe(canvas); resize();
    const animate = (now: number) => { frame = requestAnimationFrame(animate); if (!visible || pausedRef.current) return; const delta = Math.min(0.05, (now - last) / 1000); last = now; pointer.x += (target.x - pointer.x) * 4 * delta; pointer.y += (target.y - pointer.y) * 4 * delta; spin += (reduced ? 0.04 : 0.22) * delta; burger.rotation.y = spin + dragRotation.y; burger.rotation.x = pointer.y + dragRotation.x; burger.rotation.z = pointer.x * 0.25; burger.position.y = Math.sin(now * 0.0014) * (reduced ? 0.025 : 0.09); renderer.render(scene, camera); };
    frame = requestAnimationFrame(animate);
    return () => { cancelAnimationFrame(frame); observer.disconnect(); canvas.removeEventListener("pointermove", onPointer); canvas.removeEventListener("pointerdown", onPointerDown); canvas.removeEventListener("pointerup", onPointerUp); canvas.removeEventListener("pointercancel", onPointerUp); document.removeEventListener("visibilitychange", onVisibility); scene.traverse((object) => { if (object instanceof THREE.Mesh) { object.geometry.dispose(); if (Array.isArray(object.material)) object.material.forEach((item) => item.dispose()); else object.material.dispose(); } }); renderer.dispose(); };
  }, []);

  if (fallback) return <div className={`burger-fallback ${compact ? "compact" : ""}`} aria-label="Hamburger" role="img"><span className="burger-fallback-bun" /><span className="burger-fallback-lettuce" /><span className="burger-fallback-patty" /><span className="burger-fallback-cheese" /><span className="burger-fallback-patty" /><span className="burger-fallback-bun bottom" /></div>;
  return <canvas ref={canvasRef} className={`burger-canvas ${compact ? "compact" : ""}`} aria-label="Hamburger 3D" />;
}
