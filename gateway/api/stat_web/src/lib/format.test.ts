import { describe, expect, it } from "vitest";
import { formatBytes, formatMilliseconds, formatPercent, formatRate } from "./format";

describe("unit formatting", () => {
  it("formats bytes using readable binary units", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(1024)).toBe("1.0 KB");
    expect(formatBytes(1024 * 1024 * 2.5)).toBe("2.5 MB");
  });

  it("formats rates, percentages and latency", () => {
    expect(formatRate(1.234)).toContain("1.23");
    expect(formatPercent(0.125)).toBe("12.5%");
    expect(formatMilliseconds(0.4)).toBe("400 μs");
    expect(formatMilliseconds(1500)).toBe("1.50 s");
  });
});
