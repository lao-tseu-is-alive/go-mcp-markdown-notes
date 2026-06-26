/**
 * Application wiring: form handlers, config bootstrap, and startup sequence.
 *
 * Startup order: tabs → pagination controls → event handlers → /config → auth UI
 * → (jwt mode) silent token mint → initial note list.
 */
import type { AppConfig } from "./types.ts";
import { dom } from "./dom.ts";
import {
    applyDevToken,
    authState,
    clearAuth,
    mintToken,
    signIn,
    signOut,
    updateAuthStateUI,
} from "./auth.ts";
import { callConnectRPC } from "./connect.ts";
import { parseCommaSeparatedTags } from "./lib/tags.ts";
import { fetchNotesList, initNotesPaginationControls, searchNotes, stopEditMode } from "./notes-ui.ts";
import { logToConsole } from "./ui/console-log.ts";
import { showToast } from "./ui/toast.ts";
import { initTabNavigation } from "./ui/tabs.ts";

/** Binds all interactive controls declared in index.html. */
export function setupEventHandlers() {
    dom.btnClearConsole.addEventListener("click", () => {
        dom.debugOutput.innerHTML = "<div>Console cleared. Ready for next operation.</div>";
        showToast("Console cleared", "info");
    });

    dom.btnSignIn.addEventListener("click", signIn);
    dom.btnSignOut.addEventListener("click", () => { void signOut(); });

    dom.btnLoginDev.addEventListener("click", () => {
        const token = dom.devTokenInput.value.trim();
        if (!token) {
            showToast("Dev token cannot be empty", "error");
            return;
        }
        applyDevToken(token);
        showToast("Dev token applied!", "success");
    });

    dom.btnLogoutDev.addEventListener("click", () => {
        clearAuth();
        logToConsole("Dev token cleared.");
    });

    dom.createNoteForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        const title = (document.getElementById("create-title") as HTMLInputElement).value.trim();
        const category = (document.getElementById("create-category") as HTMLInputElement).value.trim();
        const tagsStr = (document.getElementById("create-tags") as HTMLInputElement).value.trim();
        const bodyMarkdown = (document.getElementById("create-body") as HTMLTextAreaElement).value.trim();
        const tags = parseCommaSeparatedTags(tagsStr);
        const status = parseInt((document.getElementById("create-status") as HTMLSelectElement).value) || 2;

        try {
            await callConnectRPC("CreateNote", { title, category, tags, bodyMarkdown, status });
            showToast("Note created successfully!", "success");
            dom.createNoteForm.reset();
            await fetchNotesList("", true);
        } catch {
            // RPC layer already surfaced the error.
        }
    });

    dom.btnSubmitList.addEventListener("click", () => { void fetchNotesList("", true); });

    dom.searchNoteForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        await searchNotes("", true);
    });

    dom.btnCancelUpdate.addEventListener("click", stopEditMode);

    dom.updateNoteForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        const noteId = (document.getElementById("update-id") as HTMLInputElement).value;
        const title = (document.getElementById("update-title") as HTMLInputElement).value.trim();
        const category = (document.getElementById("update-category") as HTMLInputElement).value.trim();
        const tagsStr = (document.getElementById("update-tags") as HTMLInputElement).value.trim();
        const bodyMarkdown = (document.getElementById("update-body") as HTMLTextAreaElement).value.trim();
        const tags = parseCommaSeparatedTags(tagsStr);
        const status = parseInt((document.getElementById("update-status") as HTMLSelectElement).value) || 2;

        try {
            await callConnectRPC("UpdateNote", { noteId, title, category, tags, bodyMarkdown, status });
            showToast("Note updated successfully!", "success");
            stopEditMode();
            await fetchNotesList("", true);
        } catch {
            // RPC layer already surfaced the error.
        }
    });
}

/** Loads auth mode and auth service base URL from the embedded notes-server. */
export async function loadAppConfig() {
    try {
        const res = await fetch("/config");
        if (!res.ok) {
            logToConsole(`Failed to load /config (HTTP ${res.status}), assuming jwt mode.`, true);
            return;
        }
        const cfg: AppConfig = await res.json();
        authState.mode = cfg.authMode === "dev" ? "dev" : "jwt";
        authState.authBaseUrl = (cfg.authBaseUrl || "").replace(/\/+$/, "");
        dom.manageTokensLink.href = `${authState.authBaseUrl}/tokens.html`;
        logToConsole(`Config loaded: authMode=${authState.mode}${authState.mode === "jwt" ? `, authBaseUrl=${authState.authBaseUrl}` : ""}`);
    } catch (e) {
        const message = e instanceof Error ? e.message : String(e);
        logToConsole(`Failed to load /config (${message}), assuming jwt mode.`, true);
    }
}

export async function bootstrap() {
    initTabNavigation();
    initNotesPaginationControls();
    setupEventHandlers();

    await loadAppConfig();
    updateAuthStateUI();

    if (authState.mode === "jwt") {
        const ok = await mintToken();
        if (ok) await fetchNotesList("", true);
    }
}