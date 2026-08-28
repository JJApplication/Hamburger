import { describe, expect, it } from "vitest";
import { countryName, filterDomains, logScale, normalizeGeoData, normalizeStatResponse, topDomains, totalGeoRequests } from "./stat-utils";

describe("stat response mapping", () => {
  it("keeps legacy counters while filling missing structured fields", () => {
    const result = normalizeStatResponse({ total: 12, api: 7, static: 5, domains: [{ domain: "api.example", requests: 3 }] });
    expect(result.total).toBe(12);
    expect(result.api).toBe(7);
    expect(result.summary.total_requests).toBe(0);
    expect(result.summary.status["2xx"]).toBe(0);
    expect(result.series.requests).toEqual([]);
    expect(result.domains[0].domain).toBe("api.example");
  });

  it("normalizes geo ISO codes and ignores invalid values", () => {
    const geo = normalizeGeoData({ cn: 10, " us ": 5, XX: -1, bad: "not-a-number" });
    expect(geo).toEqual({ CN: 10, US: 5, BAD: 0 });
    expect(totalGeoRequests(geo)).toBe(15);
    expect(countryName("cn")).toBe("中国");
    expect(countryName("ZZ")).toContain("ZZ");
  });
});

describe("domain and GEO helpers", () => {
  const domains = [
    { domain: "z.example", requests: 2, errors: 0, request_bytes: 0, response_bytes: 0 },
    { domain: "api.example", requests: 9, errors: 1, request_bytes: 0, response_bytes: 0 },
    { domain: "static.example", requests: 4, errors: 0, request_bytes: 0, response_bytes: 0 },
  ];

  it("filters and ranks domains", () => {
    expect(filterDomains(domains, "API").map((item) => item.domain)).toEqual(["api.example"]);
    expect(topDomains(domains, 2).map((item) => item.domain)).toEqual(["api.example", "static.example"]);
  });

  it("uses logarithmic values for globe intensity", () => {
    expect(logScale(0, 100)).toBe(0);
    expect(logScale(100, 100)).toBe(1);
    expect(logScale(10, 100)).toBeLessThan(0.6);
  });
});
