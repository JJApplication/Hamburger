export type HamburgerValue =
  | null
  | boolean
  | number
  | string
  | HamburgerValue[]
  | { [key: string]: HamburgerValue };

export interface ProxyRule {
  api: string;
  service?: string;
  useRewrite: boolean;
  rewrite?: string;
  staticRoot?: string;
  proxyHost?: string;
  proxyPort?: number;
}

export interface DomainServiceDef {
  name: string;
  type: string;
  host?: string;
  port?: number;
  domain?: string;
  preCheck?: boolean;
  proxyPass: ProxyRule[];
}

export interface DomainBinding {
  domain: string;
  service: string;
}

export interface DomainConfigModel {
  source: string;
  runtimeMap?: string;
  bindings: DomainBinding[];
  services: DomainServiceDef[];
  error?: string;
  version: string;
}

export function parseHamburger(input: string): HamburgerValue {
  let index = 0;

  const skip = () => {
    while (index < input.length) {
      const char = input[index];
      if (char === " " || char === "\t" || char === "\n" || char === "\r") {
        index += 1;
        continue;
      }
      if (char === "#") {
        while (index < input.length && input[index] !== "\n") index += 1;
        continue;
      }
      if (char === "/" && input[index + 1] === "/") {
        while (index < input.length && input[index] !== "\n") index += 1;
        continue;
      }
      break;
    }
  };

  const fail = (message: string): never => {
    const line = input.slice(0, index).split("\n").length;
    throw new Error(`${message}（第 ${line} 行）`);
  };

  const parseValue = (): HamburgerValue => {
    skip();
    if (index >= input.length) fail("配置意外结束");
    const char = input[index];
    if (char === "{") return parseObject();
    if (char === "[") return parseArray();
    if (char === '"') return parseString();
    if (char === "-" || (char >= "0" && char <= "9")) return parseNumberOrExpression();
    return parseWord();
  };

  const parseObject = (): { [key: string]: HamburgerValue } => {
    index += 1;
    const result: { [key: string]: HamburgerValue } = {};
    skip();
    if (input[index] === "}") {
      index += 1;
      return result;
    }
    while (index < input.length) {
      const key = parseKey();
      skip();
      if (input[index] !== ":") fail("对象字段后缺少冒号");
      index += 1;
      result[key] = parseValue();
      skip();
      if (input[index] === ",") {
        index += 1;
        skip();
        if (input[index] === "}") {
          index += 1;
          return result;
        }
        continue;
      }
      if (input[index] === "}") {
        index += 1;
        return result;
      }
      return fail("对象字段后缺少逗号或结束括号");
    }
    return fail("对象没有结束");
  };

  const parseArray = (): HamburgerValue[] => {
    index += 1;
    const result: HamburgerValue[] = [];
    skip();
    if (input[index] === "]") {
      index += 1;
      return result;
    }
    while (index < input.length) {
      result.push(parseValue());
      skip();
      if (input[index] === ",") {
        index += 1;
        skip();
        if (input[index] === "]") {
          index += 1;
          return result;
        }
        continue;
      }
      if (input[index] === "]") {
        index += 1;
        return result;
      }
      return fail("数组元素后缺少逗号或结束括号");
    }
    return fail("数组没有结束");
  };

  const parseString = (): string => {
    index += 1;
    let value = "";
    while (index < input.length) {
      const char = input[index];
      if (char === '"') {
        index += 1;
        return value;
      }
      if (char === "\\") {
        const next = input[index + 1];
        const escaped: Record<string, string> = { n: "\n", t: "\t", r: "\r", '"': '"', "\\": "\\" };
        value += escaped[next] ?? next ?? "";
        index += 2;
        continue;
      }
      value += char;
      index += 1;
    }
    return fail("字符串没有结束");
  };

  const parseNumber = (): number => {
    const start = index;
    if (input[index] === "-") index += 1;
    while (index < input.length && /[0-9.]/.test(input[index])) index += 1;
    const raw = input.slice(start, index);
    const value = Number(raw);
    if (!raw || Number.isNaN(value)) fail(`无法解析数字 ${raw}`);
    return value;
  };

  const readOperator = (from: number): number | null => {
    const operators = ["<<", ">>", "*", "/", "%", "+", "-", "&", "|"];
    for (const operator of operators) {
      if (input.startsWith(operator, from)) return from + operator.length;
    }
    return null;
  };

  // Keep formulas such as `50 * 1<<20` as source text. A bare number stays numeric.
  const parseNumberOrExpression = (): HamburgerValue => {
    const start = index;
    const first = parseNumber();
    let cursor = index;
    while (input[cursor] === " " || input[cursor] === "\t") cursor += 1;
    if (!readOperator(cursor)) return first;
    index = cursor;
    while (index < input.length) {
      while (input[index] === " " || input[index] === "\t") index += 1;
      const operator = readOperator(index);
      if (operator) {
        index = operator;
        continue;
      }
      const current = input[index];
      if (current === "(" || current === ")") {
        index += 1;
        continue;
      }
      if (current >= "0" && current <= "9") {
        parseNumber();
        continue;
      }
      if (current === "@" || (current !== undefined && /[A-Za-z_]/.test(current))) {
        if (current === "@") index += 1;
        while (index < input.length && /[A-Za-z0-9_]/.test(input[index])) index += 1;
        continue;
      }
      break;
    }
    return input.slice(start, index).trim();
  };

  const parseWord = (): HamburgerValue => {
    const start = index;
    if (input[index] === "@") index += 1;
    while (index < input.length && /[A-Za-z0-9_+\-./]/.test(input[index])) index += 1;
    const word = input.slice(start, index);
    if (word === "true") return true;
    if (word === "false") return false;
    if (word === "null") return null;
    if (!word) fail("遇到无法识别的内容");
    return word;
  };

  const parseKey = (): string => {
    skip();
    if (input[index] === '"') return parseString();
    const start = index;
    if (!/[A-Za-z_]/.test(input[index] ?? "")) fail("缺少字段名");
    while (index < input.length && /[A-Za-z0-9_.-]/.test(input[index])) index += 1;
    const key = input.slice(start, index);
    if (!key) fail("缺少字段名");
    return key;
  };

  const value = parseValue();
  skip();
  if (index < input.length) fail("配置结束后还有多余内容");
  return value;
}

function isRecord(value: HamburgerValue | undefined): value is { [key: string]: HamburgerValue } {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function asString(value: HamburgerValue | undefined): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeProxy(value: HamburgerValue): ProxyRule | null {
  if (!isRecord(value)) return null;
  const staticDirect = isRecord(value.static_direct) ? value.static_direct : null;
  const proxyDirect = isRecord(value.proxy_direct) ? value.proxy_direct : null;
  const api = asString(value.api);
  const service = asString(value.service);
  const staticRoot = staticDirect ? asString(staticDirect.static_root) : "";
  const proxyHost = proxyDirect ? asString(proxyDirect.proxy_host) : "";
  const proxyPort = proxyDirect && typeof proxyDirect.proxy_port === "number" ? proxyDirect.proxy_port : undefined;
  if (!api && !service && !staticRoot && !proxyHost) return null;
  return {
    api,
    service: service || undefined,
    useRewrite: value.use_rewrite === true,
    rewrite: asString(value.rewrite) || undefined,
    staticRoot: staticRoot || undefined,
    proxyHost: proxyHost || undefined,
    proxyPort,
  };
}

function normalizeService(value: HamburgerValue): DomainServiceDef | null {
  if (!isRecord(value)) return null;
  const name = asString(value.service_name);
  if (!name) return null;
  const proxyPass = Array.isArray(value.proxy_pass) ? value.proxy_pass.flatMap((item) => {
    const rule = normalizeProxy(item);
    return rule ? [rule] : [];
  }) : [];
  return {
    name,
    type: asString(value.service_type) || "custom",
    host: asString(value.host) || undefined,
    port: typeof value.port === "number" ? value.port : undefined,
    domain: asString(value.service_domain) || undefined,
    preCheck: isRecord(value.pre_check) ? value.pre_check.enabled === true : undefined,
    proxyPass,
  };
}

export function isBareHamburger(value: string): boolean {
  if (/^@[A-Za-z_][A-Za-z0-9_]*$/.test(value)) return true;
  const body = value.startsWith("-") ? value.slice(1) : value;
  if (!/^\d/.test(body)) return false;
  if (!/[+\-*/%&|<>]/.test(body)) return false;
  return /^[\d\s+\-*/%<>()|&@A-Za-z0-9_.]+$/.test(value);
}

function serializeKey(key: string): string {
  return /^[A-Za-z_][A-Za-z0-9_.-]*$/.test(key) ? key : JSON.stringify(key);
}

function serializeValue(value: HamburgerValue, level: number): string {
  if (value === null) return "null";
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number") return String(value);
  if (typeof value === "string") return isBareHamburger(value) ? value : JSON.stringify(value);
  const indent = "  ".repeat(level + 1);
  const end = "  ".repeat(level);
  if (Array.isArray(value)) {
    if (value.length === 0) return "[]";
    return `[\n${value.map((item) => `${indent}${serializeValue(item, level + 1)}`).join(",\n")}\n${end}]`;
  }
  const entries = Object.entries(value);
  if (entries.length === 0) return "{}";
  return `{\n${entries.map(([key, child]) => `${indent}${serializeKey(key)}: ${serializeValue(child, level + 1)}`).join(",\n")}\n${end}}`;
}

export function serializeHamburgerBlock(value: HamburgerValue, level = 0): string {
  return serializeValue(value, level);
}

export function serializeHamburger(value: HamburgerValue): string {
  return `${serializeHamburgerBlock(value, 0)}\n`;
}

export function normalizeDomainConfig(value: HamburgerValue, source: string, runtimeMap?: string): Omit<DomainConfigModel, "version"> {
  if (!isRecord(value)) throw new Error("域名配置根节点必须是对象");
  const services = Array.isArray(value.services) ? value.services.flatMap((item) => {
    const service = normalizeService(item);
    return service ? [service] : [];
  }) : [];
  const bindings = Array.isArray(value.domain_service)
    ? value.domain_service.flatMap((item) => {
        if (!isRecord(item)) return [];
        const domain = asString(item.domain);
        const service = asString(item.service);
        return domain && service ? [{ domain, service }] : [];
      })
    : services.flatMap((service) => (service.domain ? [{ domain: service.domain, service: service.name }] : []));
  return { source, runtimeMap, bindings, services };
}
