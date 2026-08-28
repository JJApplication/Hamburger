import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, fetchStat } from "./api";

describe("stat API client", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("maps a response and sends range/domain query parameters", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ total: 2, summary: { total_requests: 2 } }) });
    vi.stubGlobal("fetch", fetchMock);
    const result = await fetchStat("5h", "Example.COM");
    expect(result.total).toBe(2);
    expect(result.summary.total_requests).toBe(2);
    expect(fetchMock.mock.calls[0][0]).toContain("/api/stat?range=5h&domain=Example.COM");
  });

  it("surfaces API errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 400, json: async () => ({ message: "bad range" }) }));
    await expect(fetchStat("1h")).rejects.toMatchObject({ status: 400, message: "bad range" } satisfies Pick<ApiError, "status" | "message">);
  });
});
