/**
 * Error taxonomy for DevTerm RPC errors.
 * Maps backend error codes to appropriate frontend handling.
 */

export type ErrorCode =
  | "AUTH_FAILED"
  | "CONN_UNREACHABLE"
  | "CONN_LOST"
  | "VAULT_UNAVAILABLE"
  | "SFTP_ERROR"
  | "VALIDATION"
  | "INTERNAL";

export interface RpcError {
  code: ErrorCode;
  message: string;
  data?: unknown;
}

export function parseRpcError(error: unknown): RpcError {
  if (typeof error === "string") {
    // Try to parse as JSON
    try {
      const parsed = JSON.parse(error);
      if (parsed.code && parsed.message) {
        return parsed as RpcError;
      }
    } catch {
      // Not JSON, treat as generic
    }

    // Check for known patterns
    if (error.includes("auth") || error.includes("password") || error.includes("key rejected")) {
      return { code: "AUTH_FAILED", message: error };
    }
    if (error.includes("unreachable") || error.includes("timeout") || error.includes("connection refused")) {
      return { code: "CONN_UNREACHABLE", message: error };
    }
    if (error.includes("connection lost") || error.includes("broken pipe") || error.includes("EOF")) {
      return { code: "CONN_LOST", message: error };
    }
    if (error.includes("vault") || error.includes("keychain")) {
      return { code: "VAULT_UNAVAILABLE", message: error };
    }
    if (error.includes("permission") || error.includes("not found") || error.includes("no such file")) {
      return { code: "SFTP_ERROR", message: error };
    }
    if (error.includes("required") || error.includes("invalid")) {
      return { code: "VALIDATION", message: error };
    }
    return { code: "INTERNAL", message: error };
  }

  if (error instanceof Error) {
    return parseRpcError(error.message);
  }

  return { code: "INTERNAL", message: String(error) };
}

/**
 * Returns a user-friendly message for an error code.
 */
export function friendlyMessage(code: ErrorCode): string {
  switch (code) {
    case "AUTH_FAILED":
      return "Authentication failed. Check your credentials.";
    case "CONN_UNREACHABLE":
      return "Could not reach the host. Check the hostname and network.";
    case "CONN_LOST":
      return "Connection to the host was lost.";
    case "VAULT_UNAVAILABLE":
      return "Cannot access secure storage. Credentials cannot be saved.";
    case "SFTP_ERROR":
      return "File operation failed.";
    case "VALIDATION":
      return "Invalid input.";
    case "INTERNAL":
      return "An unexpected error occurred.";
  }
}
