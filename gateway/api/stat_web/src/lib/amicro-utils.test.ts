import { describe, expect, it } from "vitest";
import { makeHourlyActivityCells } from "./amicro-utils";
import type { StatResponse } from "../types/stat";

function statWithRequests(requests: Array<{ timestamp: string; total: number }>): StatResponse {
  return { series: { requests } } as StatResponse;
}

describe("amicro overview data", () => {
  it("aggregates quarter-hour request buckets into the latest 168 hours", () => {
    const cells = makeHourlyActivityCells(statWithRequests([
      { timestamp: "2026-08-21T00:05:00Z", total: 9 },
      { timestamp: "2026-08-28T10:00:00Z", total: 4 },
      { timestamp: "2026-08-28T10:15:00Z", total: 6 },
      { timestamp: "2026-08-28T11:00:00Z", total: 8 },
    ]));

    expect(cells).toHaveLength(168);
    expect(cells.find((cell) => cell.timestamp === "2026-08-28T10:00:00.000Z")?.requests).toBe(10);
    expect(cells.at(-1)?.requests).toBe(8);
    expect(cells.at(-1)?.level).toBe(4);
  });

  it("returns an empty heatmap for missing or invalid timestamps", () => {
    expect(makeHourlyActivityCells()).toEqual([]);
    expect(makeHourlyActivityCells(statWithRequests([{ timestamp: "not-a-date", total: 10 }]))).toEqual([]);
  });
});
