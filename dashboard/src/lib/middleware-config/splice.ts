import { serializeHamburgerBlock, type HamburgerValue } from "@/lib/domain-config/parse";

interface FieldRange {
  start: number;
  key: number;
  end: number;
}

function skipString(source: string, index: number): number {
  let cursor = index + 1;
  while (cursor < source.length) {
    if (source[cursor] === "\\") {
      cursor += 2;
      continue;
    }
    if (source[cursor] === '"') return cursor + 1;
    cursor += 1;
  }
  return cursor;
}

function skipLine(source: string, index: number): number {
  while (index < source.length && source[index] !== "\n") index += 1;
  return index;
}

function skipSpace(source: string, index: number): number {
  let cursor = index;
  while (cursor < source.length) {
    const char = source[cursor];
    if (char === " " || char === "\t" || char === "\r" || char === "\n") {
      cursor += 1;
      continue;
    }
    if (char === "#") {
      cursor = skipLine(source, cursor);
      continue;
    }
    if (char === "/" && source[cursor + 1] === "/") {
      cursor = skipLine(source, cursor);
      continue;
    }
    break;
  }
  return cursor;
}

function skipValue(source: string, index: number): number {
  const open = source[index];
  if (open !== "{" && open !== "[") {
    let cursor = index;
    while (cursor < source.length && source[cursor] !== "\n" && source[cursor] !== ",") cursor += 1;
    return cursor;
  }
  const close = open === "{" ? "}" : "]";
  let depth = 0;
  let cursor = index;
  while (cursor < source.length) {
    const char = source[cursor];
    if (char === '"') {
      cursor = skipString(source, cursor);
      continue;
    }
    if (char === "#") {
      cursor = skipLine(source, cursor);
      continue;
    }
    if (char === "/" && source[cursor + 1] === "/") {
      cursor = skipLine(source, cursor);
      continue;
    }
    if (char === open) depth += 1;
    else if (char === close) {
      depth -= 1;
      if (depth === 0) return cursor + 1;
    }
    cursor += 1;
  }
  throw new Error("middleware block is not closed");
}

function findRootField(source: string, field: string): FieldRange | null {
  let cursor = 0;
  let depth = 0;
  let lineStart = 0;
  while (cursor < source.length) {
    const char = source[cursor];
    if (char === "\n") {
      lineStart = cursor + 1;
      cursor += 1;
      continue;
    }
    if (char === " " || char === "\t" || char === "\r") {
      cursor += 1;
      continue;
    }
    if (char === "#") {
      cursor = skipLine(source, cursor);
      continue;
    }
    if (char === "/" && source[cursor + 1] === "/") {
      cursor = skipLine(source, cursor);
      continue;
    }
    if (char === '"') {
      cursor = skipString(source, cursor);
      continue;
    }
    if (char === "{") {
      depth += 1;
      cursor += 1;
      continue;
    }
    if (char === "}") {
      depth -= 1;
      cursor += 1;
      continue;
    }
    if (depth === 1 && /[A-Za-z_]/.test(char)) {
      const keyStart = cursor;
      while (cursor < source.length && /[A-Za-z0-9_-]/.test(source[cursor])) cursor += 1;
      const key = source.slice(keyStart, cursor);
      cursor = skipSpace(source, cursor);
      if (source[cursor] !== ":") continue;
      cursor = skipSpace(source, cursor + 1);
      if (key !== field) continue;
      return { start: lineStart, key: keyStart, end: skipValue(source, cursor) };
    }
    cursor += 1;
  }
  return null;
}

export function replaceRootField(source: string, field: string, value: HamburgerValue): string {
  const range = findRootField(source, field);
  if (!range) throw new Error(`configuration field ${field} does not exist`);
  const indent = source.slice(range.start, range.key);
  const rendered = `${indent}${field}: ${serializeHamburgerBlock(value, 1)}`;
  return source.slice(0, range.start) + rendered + source.slice(range.end);
}
