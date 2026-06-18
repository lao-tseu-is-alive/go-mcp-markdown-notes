// Connect RPC Notes Client Logic

// --- STATE ---
// authMode "jwt": tokens are silently minted from the auth service using the
// SSO session cookie and kept in memory only (never persisted in the browser).
// authMode "dev": a static dev token is entered manually (local hacking).
let authMode: "jwt" | "dev" = "jwt";
let authBaseUrl = "";
let currentToken = "";
let currentUser: TokenUser | null = null;
let remintTimer: ReturnType<typeof setTimeout> | null = null;
let currentNotesList: any[] = [];

interface TokenUser {
    user_id: number;
    email: string;
    name: string;
    avatar_url?: string;
    is_admin: boolean;
}

interface TokenResponse {
    token: string;
    expires_in_seconds: number;
    user: TokenUser;
}

// --- DOM ELEMENTS ---
const workspaceTabButtons = document.querySelectorAll<HTMLElement>("[data-workspace-tab]");
const workspacePanels = document.querySelectorAll<HTMLElement>("[data-workspace-panel]");

const ssoPanel = document.getElementById("auth-panel-sso") as HTMLElement;
const devPanel = document.getElementById("auth-panel-dev") as HTMLElement;
const devTokenInput = document.getElementById("dev-token-input") as HTMLInputElement;
const btnLoginDev = document.getElementById("btn-login-dev") as HTMLButtonElement;
const btnLogoutDev = document.getElementById("btn-logout-dev") as HTMLButtonElement;
const btnSignIn = document.getElementById("btn-sign-in") as HTMLButtonElement;
const btnSignOut = document.getElementById("btn-sign-out") as HTMLButtonElement;
const ssoHint = document.getElementById("sso-hint") as HTMLElement;
const manageTokensLink = document.getElementById("lnk-manage-tokens") as HTMLAnchorElement;

const authStatusBadge = document.getElementById("auth-status") as HTMLElement;
const authStatusText = document.getElementById("auth-status-text") as HTMLElement;
const userProfileCard = document.getElementById("user-profile-card") as HTMLElement;
const profileName = document.getElementById("profile-name") as HTMLElement;
const profileEmail = document.getElementById("profile-email") as HTMLElement;
const profileId = document.getElementById("profile-id") as HTMLElement;
const profileScopes = document.getElementById("profile-scopes") as HTMLElement;

const createNoteForm = document.getElementById("create-note-form") as HTMLFormElement;
const searchNoteForm = document.getElementById("search-note-form") as HTMLFormElement;
const updateNoteForm = document.getElementById("update-note-form") as HTMLFormElement;

const btnSubmitList = document.getElementById("btn-submit-list") as HTMLButtonElement;
const listLimitInput = document.getElementById("list-limit") as HTMLInputElement;

const notesContainer = document.getElementById("notes-container") as HTMLElement;
const notesCountLabel = document.getElementById("notes-count") as HTMLElement;

const debugOutput = document.getElementById("debug-output") as HTMLElement;
const btnClearConsole = document.getElementById("btn-clear-console") as HTMLElement;

const toastNotification = document.getElementById("toast-notification") as HTMLElement;
const toastMessage = document.getElementById("toast-message") as HTMLElement;

const createNotePanel = document.getElementById("op-panel-create") as HTMLElement;
const updateNotePanel = document.getElementById("op-panel-update") as HTMLElement;
const btnCancelUpdate = document.getElementById("btn-cancel-update") as HTMLButtonElement;

function escapeHtml(value: unknown): string {
    return String(value ?? "").replace(/[&<>"']/g, (char) => {
        const entities: Record<string, string> = {
            "&": "&amp;",
            "<": "&lt;",
            ">": "&gt;",
            "\"": "&quot;",
            "'": "&#39;",
        };
        return entities[char] ?? char;
    });
}

// --- DEBUG LOGGER ---
function logToConsole(message: string, isError = false) {
    const timestamp = new Date().toLocaleTimeString();
    const prefix = `[${timestamp}] `;
    const entryDiv = document.createElement("div");
    entryDiv.className = "console-entry";

    if (isError) {
        entryDiv.style.color = "var(--accent-red)";
    }

    entryDiv.textContent = prefix + message;
    debugOutput.insertBefore(entryDiv, debugOutput.firstChild);
}

function logRequest(url: string, headers: any, body: any) {
    const entryDiv = document.createElement("div");
    entryDiv.className = "console-entry";

    // Mask sensitive headers in display
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
    debugOutput.insertBefore(entryDiv, debugOutput.firstChild);
}

function logResponse(status: number, statusText: string, data: any) {
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
    debugOutput.insertBefore(entryDiv, debugOutput.firstChild);
}

// --- TOAST NOTIFICATION ---
let toastTimeout: Timer;
function showToast(message: string, type: "success" | "error" | "info" = "info") {
    clearTimeout(toastTimeout);
    toastMessage.textContent = message;
    toastNotification.className = `toast toast-${type} show`;

    toastTimeout = setTimeout(() => {
        toastNotification.classList.remove("show");
    }, 4000);
}

// --- MARKDOWN SIMPLE PARSER ---
function renderMarkdown(md: string): string {
    if (!md) return "<em>No content</em>";
    let html = md
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");

    // Code blocks (multiline)
    html = html.replace(/```([\s\S]*?)```/gm, (_, code) => `<pre style="font-family: var(--font-mono),monospace; background: rgba(0,0,0,0.5); padding: 0.5rem; border-radius: 4px; overflow-x: auto; margin: 0.5rem 0; font-size: 0.8rem; border: 1px solid rgba(255,255,255,0.05); color: #a5f3fc;">${code.trim()}</pre>`);

    // Headers
    html = html.replace(/^### (.*$)/gim, '<h3 style="margin: 0.75rem 0 0.25rem 0; font-size: 1rem; color: #e9d5ff;">$1</h3>');
    html = html.replace(/^## (.*$)/gim, '<h2 style="margin: 1rem 0 0.5rem 0; font-size: 1.15rem; color: #c084fc;">$1</h2>');
    html = html.replace(/^# (.*$)/gim, '<h1 style="margin: 1.25rem 0 0.75rem 0; font-size: 1.35rem; color: #d8b4fe; border-bottom: 1px solid rgba(255,255,255,0.05); padding-bottom: 0.25rem;">$1</h1>');

    // Bold
    html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');

    // Italic
    html = html.replace(/\*(.*?)\*/g, '<em>$1</em>');

    // Inline code
    html = html.replace(/`(.*?)`/g, '<code style="font-family: var(--font-mono),monospace; background: rgba(255,255,255,0.1); padding: 0.15rem 0.3rem; border-radius: 4px; font-size: 0.85rem; color: #f472b6;">$1</code>');

    // Linebreaks
    html = html.replace(/\n/g, "<br>");

    return html;
}

// --- AUTH STATE MANAGEMENT ---
function updateAuthStateUI() {
    const connected = currentToken !== "";
    authStatusBadge.className = connected ? "status-badge connected" : "status-badge";
    authStatusText.textContent = connected ? "CONNECTED" : "NOT CONNECTED";
    userProfileCard.style.display = connected ? "block" : "none";

    if (authMode === "dev") {
        devPanel.style.display = "block";
        ssoPanel.style.display = "none";
        btnLoginDev.style.display = connected ? "none" : "inline-flex";
        btnLogoutDev.style.display = connected ? "block" : "none";
        if (connected) {
            profileName.textContent = "Local Notes User";
            profileEmail.textContent = "dev@localhost";
            profileId.textContent = "1";
            profileScopes.textContent = "notes:read, notes:write, notes:mcp";
        }
        return;
    }

    // SSO (jwt) mode
    devPanel.style.display = "none";
    ssoPanel.style.display = "block";
    btnSignIn.style.display = connected ? "none" : "inline-flex";
    btnSignOut.style.display = connected ? "inline-flex" : "none";
    manageTokensLink.style.display = connected ? "inline-flex" : "none";
    ssoHint.textContent = connected
        ? "You are signed in. Tokens are refreshed silently from your SSO session."
        : "Sign in with your Google or GitHub account. You will be redirected to the auth service and back.";

    if (connected && currentUser) {
        profileName.textContent = currentUser.name || "-";
        profileEmail.textContent = currentUser.email || "-";
        profileId.textContent = String(currentUser.user_id ?? "-");
        profileScopes.textContent = currentUser.is_admin
            ? "notes:read, notes:write, notes:mcp, notes:admin"
            : "notes:read, notes:write, notes:mcp";
    }
}

// mintToken silently obtains a fresh short-lived JWT from the auth service
// using the SSO session cookie. Returns true when a token was obtained.
async function mintToken(): Promise<boolean> {
    try {
        const res = await fetch(`${authBaseUrl}/auth/token`, { credentials: "include" });
        if (res.status === 401) {
            clearAuth();
            logToConsole("No SSO session: sign-in required.");
            return false;
        }
        if (!res.ok) {
            logToConsole(`Token mint failed with HTTP ${res.status}`, true);
            return false;
        }
        const data: TokenResponse = await res.json();
        currentToken = data.token;
        currentUser = data.user;
        scheduleRemint(data.expires_in_seconds);
        updateAuthStateUI();
        logToConsole(`Token minted for ${data.user.email}, valid ${data.expires_in_seconds}s.`);
        return true;
    } catch (e: any) {
        logToConsole(`Token mint failed: ${e.message} (is the auth service up at ${authBaseUrl}?)`, true);
        return false;
    }
}

// scheduleRemint refreshes the JWT at ~80% of its lifetime so RPC calls never
// race against expiry.
function scheduleRemint(expiresInSeconds: number) {
    if (remintTimer) clearTimeout(remintTimer);
    const delayMs = Math.max(expiresInSeconds * 800, 30_000);
    remintTimer = setTimeout(() => { void mintToken(); }, delayMs);
}

function signIn() {
    window.location.href = `${authBaseUrl}/auth/login?redirect_uri=${encodeURIComponent(window.location.href)}`;
}

async function signOut() {
    try {
        await fetch(`${authBaseUrl}/auth/logout`, { method: "POST", credentials: "include" });
    } catch (e: any) {
        logToConsole(`Logout call failed: ${e.message}`, true);
    }
    clearAuth();
    showToast("Signed out.", "info");
}

function clearAuth() {
    currentToken = "";
    currentUser = null;
    if (remintTimer) clearTimeout(remintTimer);
    updateAuthStateUI();
}

// --- CONNECT SERVICE CALLS ---
async function callConnectRPC(methodName: string, requestData: any, isRetry = false): Promise<any> {
    const url = `/notes.v1.NotesService/${methodName}`;
    const headers: Record<string, string> = {
        "Content-Type": "application/json",
        "Connect-Protocol-Version": "1",
    };

    if (currentToken) {
        headers["Authorization"] = `Bearer ${currentToken}`;
    }

    logRequest(url, headers, requestData);

    let response: Response;
    try {
        response = await fetch(url, {
            method: "POST",
            headers: headers,
            body: JSON.stringify(requestData),
        });
    } catch (e: any) {
        logToConsole(`Fetch failed: ${e.message}`, true);
        throw e;
    }

    // An expired JWT yields 401: silently re-mint once and retry.
    if (response.status === 401 && authMode === "jwt" && !isRetry) {
        logToConsole("Got 401, re-minting token and retrying...");
        if (await mintToken()) {
            return callConnectRPC(methodName, requestData, true);
        }
    }

    let responseData: any;
    const contentType = response.headers.get("content-type") || "";

    if (contentType.includes("application/json")) {
        responseData = await response.json();
    } else {
        responseData = await response.text();
    }

    logResponse(response.status, response.statusText, responseData);

    if (!response.ok) {
        const errorMsg = responseData?.message || responseData?.error || `HTTP error ${response.status}`;
        showToast(`RPC Failed: ${errorMsg}`, "error");
        throw new Error(errorMsg);
    }

    return responseData;
}

// --- NOTE STATUS HELPERS ---
interface NoteStatusInfo {
    key: string;
    label: string;
}

function noteStatusInfo(status: string | number | undefined): NoteStatusInfo | null {
    const map: Record<string | number, NoteStatusInfo> = {
        1: { key: "draft", label: "Draft" },
        "NOTE_STATUS_DRAFT": { key: "draft", label: "Draft" },
        2: { key: "active", label: "Active" },
        "NOTE_STATUS_ACTIVE": { key: "active", label: "Active" },
        3: { key: "final", label: "Final" },
        "NOTE_STATUS_FINAL": { key: "final", label: "Final" },
        4: { key: "archived", label: "Archived" },
        "NOTE_STATUS_ARCHIVED": { key: "archived", label: "Archived" },
    };
    return status !== undefined ? (map[status] ?? null) : null;
}

function statusToInt(status: string | number | undefined): number {
    if (typeof status === "number" && status >= 1 && status <= 4) return status;
    const map: Record<string, number> = {
        "NOTE_STATUS_DRAFT": 1,
        "NOTE_STATUS_ACTIVE": 2,
        "NOTE_STATUS_FINAL": 3,
        "NOTE_STATUS_ARCHIVED": 4,
    };
    return typeof status === "string" ? (map[status] ?? 2) : 2;
}

// --- NOTE RENDER FUNCTIONS ---
function populateNotesUI(notes: any[], totalSize?: number) {
    notesContainer.innerHTML = "";
    currentNotesList = notes || [];
    const total = totalSize ?? currentNotesList.length;
    notesCountLabel.textContent = total > currentNotesList.length
        ? `${currentNotesList.length} of ${total} notes`
        : `${currentNotesList.length} notes loaded`;

    if (currentNotesList.length === 0) {
        notesContainer.innerHTML = `
            <div style="text-align: center; color: var(--text-muted); padding: 3rem 0;">
                <span style="font-size: 2.5rem; display: block; margin-bottom: 0.5rem;">📂</span>
                No notes found.
            </div>
        `;
        return;
    }

    currentNotesList.forEach((note: any) => {
        const card = document.createElement("div");
        card.className = "note-card";

        const noteId = String(note.id ?? "");
        const safeNoteId = escapeHtml(noteId);
        const categoryLabel = note.category ? `<span class="note-category">${escapeHtml(note.category)}</span>` : "";

        const si = noteStatusInfo(note.status);
        const statusBadge = si ? `<span class="note-status-${si.key}">${si.label}</span>` : "";

        const tags = note.tags || [];
        const tagsHtml = tags.map((t: string) => `<span class="tag-pill">${escapeHtml(t)}</span>`).join("");

        const updatedStr = note.updatedAt ? new Date(note.updatedAt).toLocaleString() : "unknown";

        card.innerHTML = `
            <div class="note-header">
                <div class="note-title-line">
                    <span class="note-title">${escapeHtml(note.title || "Untitled")}</span>
                    ${categoryLabel}${statusBadge}
                </div>
                <span class="note-id" data-copy-id="${safeNoteId}" title="Click to copy ID">ID: ${escapeHtml(noteId.substring(0, 8))}...</span>
            </div>
            <div class="note-body">${renderMarkdown(note.bodyMarkdown)}</div>
            <div class="note-footer">
                <div class="note-tags">${tagsHtml}</div>
                <div style="display: flex; gap: 0.5rem; align-items: center;">
                    <span style="font-size: 0.75rem; color: var(--text-muted);">Updated: ${escapeHtml(updatedStr)}</span>
                    <div class="note-actions">
                        <button class="btn-secondary btn-icon" data-action="add-tag" data-note-id="${safeNoteId}">+ Tag</button>
                        <button class="btn-accent btn-icon" data-action="edit" data-note-id="${safeNoteId}">Edit</button>
                        <button class="btn-danger btn-icon" data-action="delete" data-note-id="${safeNoteId}">Delete</button>
                    </div>
                </div>
            </div>
        `;

        notesContainer.appendChild(card);
    });

    // Attach card action listeners
    notesContainer.querySelectorAll("[data-copy-id]").forEach((el) => {
        el.addEventListener("click", (e) => {
            const id = (e.target as HTMLElement).getAttribute("data-copy-id") || "";
            void navigator.clipboard.writeText(id)
                .then(() => showToast("Copied note ID to clipboard!", "success"))
                .catch((err: any) => {
                    logToConsole(`Failed to copy note ID: ${err.message}`, true);
                    showToast("Failed to copy note ID.", "error");
                });
        });
    });

    notesContainer.querySelectorAll("[data-action='add-tag']").forEach((el) => {
        el.addEventListener("click", async (e) => {
            const noteId = (e.target as HTMLElement).getAttribute("data-note-id") || "";
            const tagsPrompt = prompt("Enter tag(s) to add (comma-separated):");
            if (tagsPrompt) {
                const parsedTags = tagsPrompt.split(",").map(t => t.trim()).filter(Boolean);
                if (parsedTags.length > 0) {
                    try {
                        await callConnectRPC("AddTags", {
                            noteId: noteId,
                            tags: parsedTags
                        });
                        showToast("Tags added successfully!", "success");
                        // Refresh notes list
                        await fetchNotesList();
                    } catch (err) {}
                }
            }
        });
    });

    notesContainer.querySelectorAll("[data-action='edit']").forEach((el) => {
        el.addEventListener("click", (e) => {
            const noteId = (e.target as HTMLElement).getAttribute("data-note-id") || "";
            const note = currentNotesList.find(n => n.id === noteId);
            if (note) {
                startEditMode(note);
            }
        });
    });

    notesContainer.querySelectorAll("[data-action='delete']").forEach((el) => {
        el.addEventListener("click", async (e) => {
            const noteId = (e.target as HTMLElement).getAttribute("data-note-id") || "";
            const note = currentNotesList.find(n => n.id === noteId);
            const title = note?.title || noteId;
            if (!confirm(`Delete note "${title}"? This cannot be undone.`)) return;
            try {
                await callConnectRPC("DeleteNote", { noteId: noteId });
                showToast("Note deleted.", "success");
                await fetchNotesList();
            } catch (err) {}
        });
    });
}

function startEditMode(note: any) {
    // Fill fields
    (document.getElementById("update-id") as HTMLInputElement).value = note.id;
    (document.getElementById("update-title") as HTMLInputElement).value = note.title || "";
    (document.getElementById("update-category") as HTMLInputElement).value = note.category || "";
    (document.getElementById("update-tags") as HTMLInputElement).value = (note.tags || []).join(", ");
    (document.getElementById("update-body") as HTMLTextAreaElement).value = note.bodyMarkdown || "";
    (document.getElementById("update-status") as HTMLSelectElement).value = String(statusToInt(note.status));

    // Show tab & navigate
    createNotePanel.classList.remove("active");
    updateNotePanel.classList.add("active");
    activateWorkspaceTab("create");
}

function stopEditMode() {
    updateNotePanel.classList.remove("active");
    createNotePanel.classList.add("active");
}

async function fetchNotesList() {
    const limit = parseInt(listLimitInput.value) || 10;
    try {
        const response = await callConnectRPC("ListRecentNotes", { limit: limit });
        populateNotesUI(response.notes);
        activateWorkspaceTab("notes");
        showToast("Notes list updated!", "success");
    } catch (err) {}
}

// --- INITIALIZE & ATTACH LISTENERS ---

// Tab Switching Helper
function initTabNavigation() {
    workspaceTabButtons.forEach((btn) => {
        btn.addEventListener("click", () => {
            const workspaceTab = btn.getAttribute("data-workspace-tab");
            if (workspaceTab) activateWorkspaceTab(workspaceTab);
        });
    });
}

function activateWorkspaceTab(workspaceTab: string) {
    workspaceTabButtons.forEach((btn) => {
        const active = btn.getAttribute("data-workspace-tab") === workspaceTab;
        btn.classList.toggle("active", active);
        btn.setAttribute("aria-selected", String(active));
    });

    workspacePanels.forEach((panel) => {
        panel.classList.toggle("active", panel.getAttribute("data-workspace-panel") === workspaceTab);
    });
}

// Setup Event Handlers
function setupEventHandlers() {
    // Console clear
    btnClearConsole.addEventListener("click", () => {
        debugOutput.innerHTML = "<div>Console cleared. Ready for next operation.</div>";
        showToast("Console cleared", "info");
    });

    // SSO sign in / out
    btnSignIn.addEventListener("click", signIn);
    btnSignOut.addEventListener("click", signOut);

    // Dev Auth Click
    btnLoginDev.addEventListener("click", () => {
        const token = devTokenInput.value.trim();
        if (!token) {
            showToast("Dev token cannot be empty", "error");
            return;
        }
        currentToken = token;
        updateAuthStateUI();
        showToast("Dev token applied!", "success");
    });

    btnLogoutDev.addEventListener("click", () => {
        clearAuth();
        logToConsole("Dev token cleared.");
    });

    // Create Note Form Submission
    createNoteForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        const title = (document.getElementById("create-title") as HTMLInputElement).value.trim();
        const category = (document.getElementById("create-category") as HTMLInputElement).value.trim();
        const tagsStr = (document.getElementById("create-tags") as HTMLInputElement).value.trim();
        const bodyMarkdown = (document.getElementById("create-body") as HTMLTextAreaElement).value.trim();

        const tags = tagsStr.split(",").map(t => t.trim()).filter(Boolean);
        const status = parseInt((document.getElementById("create-status") as HTMLSelectElement).value) || 2;

        try {
            await callConnectRPC("CreateNote", {
                title,
                category,
                tags,
                bodyMarkdown,
                status,
            });

            showToast("Note created successfully!", "success");
            createNoteForm.reset();
            await fetchNotesList();
        } catch (err) {}
    });

    // List Notes Fetch Click
    btnSubmitList.addEventListener("click", () => { void fetchNotesList(); });

    // Search Notes Form Submission
    searchNoteForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        const query = (document.getElementById("search-query") as HTMLInputElement).value.trim();
        const category = (document.getElementById("search-category") as HTMLInputElement).value.trim();
        const tagsStr = (document.getElementById("search-tags") as HTMLInputElement).value.trim();
        const limit = parseInt((document.getElementById("search-limit") as HTMLInputElement).value) || 10;

        const tags = tagsStr.split(",").map(t => t.trim()).filter(Boolean);

        try {
            const response = await callConnectRPC("SearchNotes", {
                query,
                category,
                tags,
                limit
            });

            const total: number = response.pageResponse?.totalSize ?? response.notes?.length ?? 0;
            populateNotesUI(response.notes, total);
            activateWorkspaceTab("notes");
            showToast(`Search: ${response.notes?.length || 0} of ${total} notes`, "success");
        } catch (err) {}
    });

    // Cancel Edit/Update
    btnCancelUpdate.addEventListener("click", stopEditMode);

    // Save Edit/Update Form Submission
    updateNoteForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        const noteId = (document.getElementById("update-id") as HTMLInputElement).value;
        const title = (document.getElementById("update-title") as HTMLInputElement).value.trim();
        const category = (document.getElementById("update-category") as HTMLInputElement).value.trim();
        const tagsStr = (document.getElementById("update-tags") as HTMLInputElement).value.trim();
        const bodyMarkdown = (document.getElementById("update-body") as HTMLTextAreaElement).value.trim();

        const tags = tagsStr.split(",").map(t => t.trim()).filter(Boolean);
        const status = parseInt((document.getElementById("update-status") as HTMLSelectElement).value) || 2;

        try {
            await callConnectRPC("UpdateNote", {
                noteId,
                title,
                category,
                tags,
                bodyMarkdown,
                status,
            });

            showToast("Note updated successfully!", "success");
            stopEditMode();
            await fetchNotesList();
        } catch (err) {}
    });
}

// loadAppConfig asks the notes-server how authentication works.
async function loadAppConfig() {
    try {
        const res = await fetch("/config");
        if (!res.ok) {
            logToConsole(`Failed to load /config (HTTP ${res.status}), assuming jwt mode.`, true);
            return;
        }
        const cfg = await res.json();
        authMode = cfg.authMode === "dev" ? "dev" : "jwt";
        authBaseUrl = (cfg.authBaseUrl || "").replace(/\/+$/, "");
        manageTokensLink.href = `${authBaseUrl}/tokens.html`;
        logToConsole(`Config loaded: authMode=${authMode}${authMode === "jwt" ? `, authBaseUrl=${authBaseUrl}` : ""}`);
    } catch (e: any) {
        logToConsole(`Failed to load /config (${e.message}), assuming jwt mode.`, true);
    }
}

// On Page Load
window.addEventListener("DOMContentLoaded", async () => {
    initTabNavigation();
    setupEventHandlers();

    await loadAppConfig();
    updateAuthStateUI();

    if (authMode === "jwt") {
        // Silent SSO: try to mint a token from the existing session cookie.
        // After the login redirect lands back here, this picks the session up.
        const ok = await mintToken();
        if (ok) await fetchNotesList();
    }
});
