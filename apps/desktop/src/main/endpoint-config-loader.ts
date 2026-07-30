import { app } from "electron";
import { readFile } from "fs/promises";
import { join } from "path";
import {
  DEFAULT_ENDPOINT_CONFIG,
  parseEndpointConfig,
  endpointConfigFromDevEnv,
  type EndpointConfig,
  type EndpointConfigEnv,
  type EndpointConfigResult,
} from "../shared/endpoint-config";

export async function loadEndpointConfig(options: {
  isDev: boolean;
  env: EndpointConfigEnv;
  configPath?: string;
}): Promise<EndpointConfigResult> {
  if (options.isDev) {
    try {
      return { ok: true, config: endpointConfigFromDevEnv(options.env) };
    } catch (err) {
      return { ok: false, error: { message: errorMessage(err) } };
    }
  }

  const configPath = options.configPath ?? desktopConfigPath();
  try {
    const raw = await readFile(configPath, "utf-8");
    return { ok: true, config: parseEndpointConfig(raw) };
  } catch (err) {
    if (isMissingFileError(err)) {
      return { ok: true, config: { ...DEFAULT_ENDPOINT_CONFIG } };
    }
    return {
      ok: false,
      error: {
        message: `Invalid ${configPath}: ${errorMessage(err)}`,
      },
    };
  }
}

export function desktopConfigPath(): string {
  return join(app.getPath("home"), ".multica", "desktop.json");
}

function isMissingFileError(err: unknown): boolean {
  return Boolean(
    err &&
      typeof err === "object" &&
      "code" in err &&
      (err as NodeJS.ErrnoException).code === "ENOENT",
  );
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export type { EndpointConfig, EndpointConfigResult };
