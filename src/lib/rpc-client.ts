/**
 * RPC client for communicating with the Go sidecar via Tauri IPC.
 *
 * All frontend<->backend communication goes through this single module.
 * - call<P, R>(method, params): invoke a JSON-RPC method and await the result
 * - subscribe(method, handler): listen for server-pushed notifications
 */
import { invoke } from "@tauri-apps/api/core";
import { listen, type UnlistenFn } from "@tauri-apps/api/event";

export interface RpcError {
  code: string;
  message: string;
  data?: unknown;
}

type NotificationHandler = (params: unknown) => void;

const subscriptions = new Map<string, Set<NotificationHandler>>();
let unlistenRpcEvent: UnlistenFn | null = null;

async function initListener() {
  if (unlistenRpcEvent) return;
  unlistenRpcEvent = await listen<{ method: string; params: unknown }>(
    "rpc-event",
    (event) => {
      const { method, params } = event.payload;
      const handlers = subscriptions.get(method);
      if (handlers) {
        for (const handler of handlers) {
          handler(params);
        }
      }
    }
  );
}

/**
 * Call a JSON-RPC method on the Go backend.
 */
export async function call<TParams, TResult>(
  method: string,
  params: TParams
): Promise<TResult> {
  const result = await invoke<TResult>("rpc_call", { method, params });
  return result;
}

/**
 * Cancel an in-flight RPC request by its ID (best-effort).
 */
export async function cancel(id: string): Promise<void> {
  await invoke("rpc_cancel", { id });
}

/**
 * Subscribe to server-pushed notifications of a given method.
 * Returns an unsubscribe function.
 */
export function subscribe(
  method: string,
  handler: NotificationHandler
): () => void {
  initListener();
  if (!subscriptions.has(method)) {
    subscriptions.set(method, new Set());
  }
  subscriptions.get(method)!.add(handler);
  return () => {
    subscriptions.get(method)?.delete(handler);
  };
}

export const rpcClient = { call, cancel, subscribe };
