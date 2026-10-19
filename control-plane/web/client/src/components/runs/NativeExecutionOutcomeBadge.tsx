import { useQuery } from "@tanstack/react-query";
import { getNativeExecutionOutcome } from "@/services/nativeExecutionOutcome";

export function NativeExecutionOutcomeBadge({ executionId }: { executionId?: string }) {
  const { data, error, isError } = useQuery({
    queryKey: ["native-execution-outcome", executionId],
    queryFn: () => getNativeExecutionOutcome(executionId!),
    enabled: Boolean(executionId),
    refetchInterval: 15_000,
    retry: false,
  });

  if (!executionId) return null;
  if (isError) {
    return <span role="status" title={error instanceof Error ? error.message : "Native API error"}
      className="rounded border border-destructive/40 px-2 py-0.5 text-xs text-destructive">
      App result unavailable
    </span>;
  }
  if (!data) return <span className="text-xs text-muted-foreground">App result loading</span>;
  if (data.applicationSuccess === null) {
    return <span title={`Transport: ${data.transportStatus}; result.success not reported`}
      className="rounded border border-border px-2 py-0.5 text-xs text-muted-foreground">
      App: {data.transportStatus === "running" ? "pending" : "not reported"}
    </span>;
  }
  return <span role="status"
    title={`Transport: ${data.transportStatus}; result.success=${data.applicationSuccess}`}
    className={data.applicationSuccess
      ? "rounded border border-emerald-500/30 px-2 py-0.5 text-xs text-emerald-500"
      : "rounded border border-destructive/40 px-2 py-0.5 text-xs text-destructive"}>
    App: {data.applicationSuccess ? "passed" : "failed"}
  </span>;
}
