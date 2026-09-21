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
    camera.position.set(0, 1.65, 6.4);
    camera.lookAt(0, 0.55, 0);
    const burger = new THREE.Group();
    scene.add(burger);

    const mat = (color: number, roughness: number, metalness = 0) => new THREE.MeshStandardMaterial({ color, roughness, metalness });
    const bun = mat(0xd87922, 0.68);
    const bunTop = mat(0xf39a3d, 0.58);
    const patty = mat(0x321a0c, 0.92);
    const pattyEdge = mat(0x4c2812, 0.9);
    const cheese = mat(0xffc928, 0.38);
    const lettuce = mat(0x2ecb73, 0.78);
    const sesame = mat(0xffe4a1, 0.42);

    const add = (geometry: THREE.BufferGeometry, material: THREE.Material, position: THREE.Vector3, scale?: THREE.Vector3, rotation?: THREE.Euler) => {
      const mesh = new THREE.Mesh(geometry, material);
      mesh.position.copy(position);
      if (scale) mesh.scale.copy(scale);
      if (rotation) mesh.rotation.copy(rotation);
      mesh.castShadow = true; mesh.receiveShadow = true; burger.add(mesh); return mesh;
    };
    const cylinder = (radius: number, height: number, material: THREE.Material, y: number) => add(new THREE.CylinderGeometry(radius, radius * 0.96, height, 64), material, new THREE.Vector3(0, y, 0));
    const cheeseSlice = (y: number, rotationY: number) => {
      const geometry = new THREE.BoxGeometry(2.68, 0.13, 2.68);
      const position = geometry.attributes.position;
      // Lower the two front corners to create the characteristic soft,
      // gravity-pulled cheese edge while keeping a real solid thickness.
      for (let index = 0; index < position.count; index += 1) {
        const x = position.getX(index);
        const z = position.getZ(index);
        if (z > 0.9 && Math.abs(x) > 0.75) position.setY(index, position.getY(index) - 0.22);
      }
      position.needsUpdate = true;
      geometry.computeVertexNormals();
      return add(geometry, cheese, new THREE.Vector3(0, y, 0), undefined, new THREE.Euler(0, rotationY, 0));
    };
    add(new THREE.SphereGeometry(1.62, 64, 32, 0, Math.PI * 2, 0, Math.PI / 2), bunTop, new THREE.Vector3(0, 1.12, 0), new THREE.Vector3(1, 0.66, 1));
    cylinder(1.52, 0.2, bunTop, 0.72);
    cylinder(1.47, 0.45, pattyEdge, 0.34);
    cheeseSlice(0.08, 0.5);
    cylinder(1.43, 0.48, patty, -0.28);
    cheeseSlice(-0.62, -0.32);
    cylinder(1.42, 0.46, pattyEdge, -0.91);
    add(new THREE.TorusGeometry(1.43, 0.2, 14, 64), lettuce, new THREE.Vector3(0, 0.64, 0), undefined, new THREE.Euler(Math.PI / 2, 0, 0));
    for (let i = 0; i < 7; i++) {
      const angle = (i / 7) * Math.PI * 2;
      add(new THREE.SphereGeometry(0.29, 24, 14), lettuce, new THREE.Vector3(Math.cos(angle) * 1.38, 0.61 + Math.sin(angle * 2) * 0.05, Math.sin(angle) * 1.38), new THREE.Vector3(1.35, 0.38, 0.82));
    }
    add(new THREE.SphereGeometry(1.56, 64, 28, 0, Math.PI * 2, Math.PI / 2, Math.PI / 2), bun, new THREE.Vector3(0, -1.32, 0), new THREE.Vector3(1, 0.7, 1));
    cylinder(1.49, 0.18, bun, -1.13);
    const seeds = [[-0.7, 1.68, 0.55], [-0.12, 1.86, 0.75], [0.55, 1.68, 0.58], [0.86, 1.45, 0.2], [-0.35, 1.55, -0.55], [0.28, 1.8, -0.45], [-0.92, 1.45, -0.14]];
    seeds.forEach(([x, y, z], index) => add(new THREE.SphereGeometry(0.095, 16, 10), sesame, new THREE.Vector3(x, y, z), new THREE.Vector3(0.48, 0.16, 0.18), new THREE.Euler(0.25, index * 0.7, -0.28)));

    const key = new THREE.DirectionalLight(0xffe0bb, 4.1); key.position.set(-3, 6, 4); key.castShadow = true; key.shadow.mapSize.set(1024, 1024); scene.add(key);
    const fill = new THREE.DirectionalLight(0x8bd8ff, 1.8); fill.position.set(4, 2, -3); scene.add(fill);
    scene.add(new THREE.HemisphereLight(0xffe8cd, 0x17243b, 1.4));
    const ground = new THREE.Mesh(new THREE.CircleGeometry(2.2, 64), new THREE.ShadowMaterial({ color: 0x06111f, opacity: 0.34 }));
    ground.rotation.x = -Math.PI / 2; ground.position.y = -1.72; ground.receiveShadow = true; scene.add(ground);

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
