/**
 * Connect JSON-over-HTTP client for notes.v1.NotesService.
 *
 * Uses the Connect protocol (application/json POST, Connect-Protocol-Version header).
 * In JWT mode, a single 401 triggers silent re-mint and one automatic retry.
 */
import { authState, mintToken } from "./auth.ts";
import { logRequest, logResponse, logToConsole } from "./ui/console-log.ts";
import { showToast } from "./ui/toast.ts";

export async function callConnectRPC(methodName: string, requestData: unknown, isRetry = false): Promise<any> {
    const url = `/notes.v1.NotesService/${methodName}`;
    const headers: Record<string, string> = {
        "Content-Type": "application/json",
        "Connect-Protocol-Version": "1",
    };

    if (authState.currentToken) {
        headers["Authorization"] = `Bearer ${authState.currentToken}`;
    }

    logRequest(url, headers, requestData);

    let response: Response;
    try {
        response = await fetch(url, {
            method: "POST",
            headers,
            body: JSON.stringify(requestData),
        });
    } catch (e) {
        const message = e instanceof Error ? e.message : String(e);
        logToConsole(`Fetch failed: ${message}`, true);
        throw e;
    }

    if (response.status === 401 && authState.mode === "jwt" && !isRetry) {
        logToConsole("Got 401, re-minting token and retrying...");
        if (await mintToken()) {
            return callConnectRPC(methodName, requestData, true);
        }
    }

    let responseData: unknown;
    const contentType = response.headers.get("content-type") || "";

    if (contentType.includes("application/json")) {
        responseData = await response.json();
    } else {
        responseData = await response.text();
    }

    logResponse(response.status, response.statusText, responseData);

    if (!response.ok) {
        const payload = responseData as { message?: string; error?: string } | undefined;
        const errorMsg = payload?.message || payload?.error || `HTTP error ${response.status}`;
        showToast(`RPC Failed: ${errorMsg}`, "error");
        throw new Error(errorMsg);
    }

    return responseData;
}