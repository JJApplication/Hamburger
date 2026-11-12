export interface SecurityDraft {
  strictMode: boolean;
  allowIps: string[];
  denyIps: string[];
  rateLimit: string;
  maxQuerySize: string;
  whiteList: string[];
  hsts: boolean;
  hstsSubdomain: boolean;
  hstsPreload: boolean;
  xss: boolean;
  iframe: boolean;
  sameSite: boolean;
}

export function blankSecurity(): SecurityDraft {
  return {
    strictMode: false,
    allowIps: [],
    denyIps: [],
    rateLimit: "1000",
    maxQuerySize: "2048",
    whiteList: [],
    hsts: false,
    hstsSubdomain: false,
    hstsPreload: false,
    xss: false,
    iframe: false,
    sameSite: false,
  };
}
