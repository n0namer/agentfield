import { getGlobalApiKey } from "./api";

export interface NativeExecutionOutcome {
  transportStatus: string;
  applicationSuccess: boolean | null;
}

export function parseNativeExecutionOutcome(payload: unknown): NativeExecutionOutcome {
  const execution = payload && typeof payload === "object"
    ? payload as Record<string, unknown> : {};
  const result = execution.result && typeof execution.result === "object"
    ? execution.result as Record<string, unknown> : {};
  return {
    transportStatus: typeof execution.status === "string" ? execution.status : "unknown",
    applicationSuccess: typeof result.success === "boolean" ? result.success : null,
  };
}

export async function getNativeExecutionOutcome(executionId: string): Promise<NativeExecutionOutcome> {
  const headers = new Headers();
  const apiKey = getGlobalApiKey();
  if (apiKey) headers.set("X-API-Key", apiKey);
  const response = await fetch(`/api/v1/executions/${encodeURIComponent(executionId)}`, { headers });
  if (!response.ok) throw new Error(`Native execution API: HTTP ${response.status}`);
  // Parse the existing native API; do not render private inputs or full results.
  return parseNativeExecutionOutcome(await response.json());
}
