import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AnimatedNumber } from "./AnimatedNumber";

const scheduledFrames = new Map<number, FrameRequestCallback>();
let nextFrameId = 0;

function flushFrames(timestamp: number) {
  const frames = [...scheduledFrames.entries()];
  scheduledFrames.clear();
  frames.forEach(([, callback]) => callback(timestamp));
}

describe("AnimatedNumber", () => {
  beforeEach(() => {
    nextFrameId = 0;
    scheduledFrames.clear();
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      const id = ++nextFrameId;
      scheduledFrames.set(id, callback);
      return id;
    });
    vi.stubGlobal("cancelAnimationFrame", (id: number) => scheduledFrames.delete(id));
    vi.spyOn(performance, "now").mockReturnValue(0);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("quickly counts from zero to the target value", () => {
    render(<AnimatedNumber value={120} duration={1000} />);
    expect(screen.getByText("0")).toBeInTheDocument();

    act(() => flushFrames(500));
    expect(screen.getByText("105")).toBeInTheDocument();

    act(() => flushFrames(1000));
    expect(screen.getByText("120")).toBeInTheDocument();
  });

  it("shows the target immediately when reduced motion is enabled", () => {
    render(<AnimatedNumber value={120} reducedMotion />);
    expect(screen.getByText("120")).toBeInTheDocument();
    expect(scheduledFrames.size).toBe(0);
  });
});
