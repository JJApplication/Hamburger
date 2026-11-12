export interface CompressionDraft {
  enabled: boolean;
  level: string;
  threshold: string;
  types: string[];
}

export interface ErrorPageRow {
  key: string;
  code: string;
  path: string;
}

export interface ErrorConfigDraft {
  mode: string;
  cache: boolean;
  pages: ErrorPageRow[];
}

export interface MiddlewareDraft {
  cors: { enabled: boolean; methods: string[]; origins: string[]; headers: string[] };
  trace: { enabled: boolean; traceId: string };
  secureHeader: boolean;
  noCache: boolean;
  gzip: CompressionDraft;
  zstd: CompressionDraft;
  sanitizer: boolean;
  domainCheck: boolean;
  imageProtect: { enabled: boolean; imageTypes: string[]; allowReferers: string[] };
}

export function blankCompression(): CompressionDraft {
  return { enabled: false, level: "6", threshold: "2048", types: [] };
}

export function blankMiddleware(): MiddlewareDraft {
  return {
    cors: { enabled: false, methods: [], origins: [], headers: [] },
    trace: { enabled: false, traceId: "" },
    secureHeader: false,
    noCache: false,
    gzip: blankCompression(),
    zstd: blankCompression(),
    sanitizer: false,
    domainCheck: false,
    imageProtect: { enabled: false, imageTypes: [], allowReferers: [] },
  };
}

export function blankErrorConfig(): ErrorConfigDraft {
  return { mode: "html", cache: false, pages: [] };
}
