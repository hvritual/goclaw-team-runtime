export interface EndpointConfig {
  schemaVersion: 1;
  apiUrl: string;
  wsUrl: string;
  appUrl: string;
}

export interface EndpointConfigError {
  message: string;
}

export type EndpointConfigResult =
  | { ok: true; config: EndpointConfig }
  | { ok: false; error: EndpointConfigError };

export const DEFAULT_ENDPOINT_CONFIG: EndpointConfig = Object.freeze({
  schemaVersion: 1,
  apiUrl: "https://api.multica.ai",
  wsUrl: "wss://api.multica.ai/ws",
  appUrl: "https://multica.ai",
});

const LOCAL_DEV_ENDPOINT_CONFIG: EndpointConfig = Object.freeze({
  schemaVersion: 1,
  apiUrl: "http://localhost:8080",
  wsUrl: "ws://localhost:8080/ws",
  appUrl: "http://localhost:3000",
});

export interface EndpointConfigEnv {
  apiUrl?: string;
  wsUrl?: string;
  appUrl?: string;
}

export function endpointConfigFromDevEnv(env: EndpointConfigEnv): EndpointConfig {
  const apiUrl = normalizeHttpUrl(
    env.apiUrl || LOCAL_DEV_ENDPOINT_CONFIG.apiUrl,
    "VITE_API_URL",
  );
  return {
    schemaVersion: 1,
    apiUrl,
    wsUrl: env.wsUrl
      ? normalizeWsUrl(env.wsUrl, "VITE_WS_URL")
      : deriveWsUrl(apiUrl),
    appUrl: env.appUrl
      ? normalizeHttpUrl(env.appUrl, "VITE_APP_URL")
      : deriveDevAppUrl(apiUrl),
  };
}

export function parseEndpointConfig(raw: string): EndpointConfig {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    throw new Error(
      `Invalid desktop endpoint config JSON: ${err instanceof Error ? err.message : "parse failed"}`,
    );
  }

  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("Invalid desktop endpoint config: expected a JSON object");
  }

  const obj = parsed as Record<string, unknown>;
  if (obj.schemaVersion !== 1) {
    throw new Error("Unsupported desktop endpoint config schemaVersion: expected 1");
  }

  const apiUrl = requiredString(obj.apiUrl, "apiUrl");
  const appUrl = optionalString(obj.appUrl, "appUrl");
  const wsUrl = optionalString(obj.wsUrl, "wsUrl");

  const normalizedApiUrl = normalizeHttpUrl(apiUrl, "apiUrl");
  return {
    schemaVersion: 1,
    apiUrl: normalizedApiUrl,
    wsUrl: wsUrl ? normalizeWsUrl(wsUrl, "wsUrl") : deriveWsUrl(normalizedApiUrl),
    appUrl: appUrl ? normalizeHttpUrl(appUrl, "appUrl") : deriveAppUrl(normalizedApiUrl),
  };
}

export function deriveWsUrl(apiUrl: string): string {
  const url = new URL(apiUrl);
  if (url.protocol === "https:") url.protocol = "wss:";
  else if (url.protocol === "http:") url.protocol = "ws:";
  else throw new Error("apiUrl must use http or https");
  url.pathname = joinPath(url.pathname, "/ws");
  url.search = "";
  url.hash = "";
  return trimTrailingSlash(url.toString());
}

// Convention: api hosts are exposed at `api.<web-host>` (api.multica.ai →
// multica.ai, api.test.multica.ai → test.multica.ai). Strip the leading
// `api.` label so a single `apiUrl` configuration produces the right
// shareable web URL. Hosts that don't match the convention (no leading
// `api.` label, or short two-label hosts like `api.local`) fall through
// untouched — those deployments must set `appUrl` explicitly.
export function deriveAppUrl(apiUrl: string): string {
  const url = new URL(apiUrl);
  url.pathname = "";
  url.search = "";
  url.hash = "";
  if (url.hostname.startsWith("api.") && url.hostname.split(".").length >= 3) {
    url.hostname = url.hostname.slice("api.".length);
  }
  return trimTrailingSlash(url.toString());
}

// Dev variant: when the api host is the local backend (`localhost:8080` /
// `127.0.0.1:8080`), the renderer is served from a different port (3000),
// so deriving by host alone is wrong. Fall back to the local dev web URL
// in that case; for any non-local host (e.g. a remote test environment),
// trust the production-style derivation so `apiUrl=https://api.test.x`
// yields `appUrl=https://test.x` without a separate VITE_APP_URL.
export function deriveDevAppUrl(apiUrl: string): string {
  const url = new URL(apiUrl);
  if (url.hostname === "localhost" || url.hostname === "127.0.0.1") {
    return LOCAL_DEV_ENDPOINT_CONFIG.appUrl;
  }
  return deriveAppUrl(apiUrl);
}

function requiredString(value: unknown, field: string): string {
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`Invalid desktop endpoint config: ${field} must be a non-empty string`);
  }
  return value;
}

function optionalString(value: unknown, field: string): string | undefined {
  if (value === undefined) return undefined;
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`Invalid desktop endpoint config: ${field} must be a non-empty string when set`);
  }
  return value;
}

function normalizeHttpUrl(value: string, field: string): string {
  let url: URL;
  try {
    url = new URL(value.trim());
  } catch {
    throw new Error(`Invalid desktop endpoint config: ${field} must be a valid URL`);
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new Error(`Invalid desktop endpoint config: ${field} must use http or https`);
  }
  url.search = "";
  url.hash = "";
  return trimTrailingSlash(url.toString());
}

function normalizeWsUrl(value: string, field: string): string {
  let url: URL;
  try {
    url = new URL(value.trim());
  } catch {
    throw new Error(`Invalid desktop endpoint config: ${field} must be a valid URL`);
  }
  if (url.protocol !== "ws:" && url.protocol !== "wss:") {
    throw new Error(`Invalid desktop endpoint config: ${field} must use ws or wss`);
  }
  url.search = "";
  url.hash = "";
  return trimTrailingSlash(url.toString());
}

function joinPath(base: string, suffix: string): string {
  const normalizedBase = base.endsWith("/") ? base.slice(0, -1) : base;
  return `${normalizedBase}${suffix}`;
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}
