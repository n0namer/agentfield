import { describe, expect, it, vi, afterEach } from "vitest";
import { getNativeExecutionOutcome, parseNativeExecutionOutcome } from "../../services/nativeExecutionOutcome";

vi.mock("../../services/api", () => ({ getGlobalApiKey: () => null }));

afterEach(() => vi.unstubAllGlobals());

describe("native execution outcome", () => {
  it("does not turn transport succeeded into application success", () => {
    expect(parseNativeExecutionOutcome({ status: "succeeded", result: { success: false } }))
      .toEqual({ transportStatus: "succeeded", applicationSuccess: false });
    expect(parseNativeExecutionOutcome({ status: "succeeded", result: { success: true } }))
      .toEqual({ transportStatus: "succeeded", applicationSuccess: true });
  });

  it("distinguishes missing application success from false", () => {
    expect(parseNativeExecutionOutcome({ status: "running" }))
      .toEqual({ transportStatus: "running", applicationSuccess: null });
    expect(parseNativeExecutionOutcome({ status: "succeeded", result: { output: "ok" } }).applicationSuccess)
      .toBeNull();
  });

  it("reads actual native endpoint and exposes only aggregate outcome", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ status: "succeeded", result: { success: false, secret: "never-return-this" } }),
    });
    vi.stubGlobal("fetch", mockFetch);
    const result = await getNativeExecutionOutcome("exec_test");
    expect(result).toEqual({ transportStatus: "succeeded", applicationSuccess: false });
    expect(mockFetch.mock.calls[0][0]).toBe("/api/v1/executions/exec_test");
  });

  it("surfaces API failures rather than showing a successful empty result", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 502 }));
    await expect(getNativeExecutionOutcome("exec_test")).rejects.toThrow("HTTP 502");
  });
});
