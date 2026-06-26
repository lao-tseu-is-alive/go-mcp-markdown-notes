/**
 * Connect protocol debugger output — prepends timestamped request/response entries.
 * Authorization headers are masked in the display copy.
 */
import { dom } from "../dom.ts";
import { escapeHtml } from "../lib/escape.ts";

export function logToConsole(message: string, isError = false) {
    const timestamp = new Date().toLocaleTimeString();
    const entryDiv = document.createElement("div");
    entryDiv.className = "console-entry";
    if (isError) {
        entryDiv.style.color = "var(--accent-red)";
    }
    entryDiv.textContent = `[${timestamp}] ${message}`;
    dom.debugOutput.insertBefore(entryDiv, dom.debugOutput.firstChild);
}

export function logRequest(url: string, headers: Record<string, string>, body: unknown) {
    const entryDiv = document.createElement("div");
    entryDiv.className = "console-entry";

    const displayHeaders = { ...headers };
    if (displayHeaders["Authorization"]) {
        const authVal = displayHeaders["Authorization"];
        if (authVal.length > 20) {
            displayHeaders["Authorization"] = authVal.substring(0, 15) + "... [masked]";
        }
    }

    entryDiv.innerHTML = `
        <div class="console-req-header">➡️ POST ${escapeHtml(url)}</div>
        <div><strong>Headers:</strong> ${escapeHtml(JSON.stringify(displayHeaders, null, 2))}</div>
        <div><strong>Request Body:</strong></div>
        <pre class="console-data">${escapeHtml(JSON.stringify(body, null, 2))}</pre>
    `;
    dom.debugOutput.insertBefore(entryDiv, dom.debugOutput.firstChild);
}

export function logResponse(status: number, statusText: string, data: unknown) {
    const entryDiv = document.createElement("div");
    entryDiv.className = "console-entry";
    const isErr = status >= 400;

    entryDiv.innerHTML = `
        <div class="console-resp-header" style="color: ${isErr ? "var(--accent-red)" : "var(--accent-green)"}">
            ⬅️ RESPONSE: ${escapeHtml(status)} ${escapeHtml(statusText)}
        </div>
        <div><strong>Body:</strong></div>
        <pre class="console-data">${escapeHtml(typeof data === "string" ? data : JSON.stringify(data, null, 2))}</pre>
    `;
    dom.debugOutput.insertBefore(entryDiv, dom.debugOutput.firstChild);
}