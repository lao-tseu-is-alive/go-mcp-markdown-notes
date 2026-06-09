// Connect RPC Notes Client Logic

// --- CONSTANTS ---
const STORAGE_TOKEN_KEY = "notes_auth_token";
const STORAGE_TYPE_KEY = "notes_auth_type";

// --- STATE ---
let currentToken = localStorage.getItem(STORAGE_TOKEN_KEY) || "";
let authType = localStorage.getItem(STORAGE_TYPE_KEY) || "dev"; // "dev" or "jwt"
let currentNotesList: any[] = [];

// --- DOM ELEMENTS ---
const tabButtons = document.querySelectorAll<HTMLElement>("[data-auth-tab], [data-op-tab]");
const authPanels = document.querySelectorAll<HTMLElement>(".auth-container .tab-panel");
const opPanels = document.querySelectorAll<HTMLElement>(".card:nth-of-type(2) .tab-panel");

const devTokenInput = document.getElementById("dev-token-input") as HTMLInputElement;
const jwtTokenInput = document.getElementById("jwt-token-input") as HTMLInputElement;
const btnLoginDev = document.getElementById("btn-login-dev") as HTMLButtonElement;
const btnLogoutDev = document.getElementById("btn-logout-dev") as HTMLButtonElement;
const btnLoginJwt = document.getElementById("btn-login-jwt") as HTMLButtonElement;
const btnLogoutJwt = document.getElementById("btn-logout-jwt") as HTMLButtonElement;

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

const tabUpdateToggle = document.getElementById("tab-update-toggle") as HTMLElement;
const btnCancelUpdate = document.getElementById("btn-cancel-update") as HTMLButtonElement;

// --- JWT DECODER ---
interface JWTUser {
    name?: string;
    email?: string;
    id?: string | number;
    roles?: string[];
    provider?: string;
    scopes?: string[];
}

interface JWTPayload {
    user?: JWTUser;
    sub?: string;
    email?: string;
    name?: string;
    scopes?: string[];
    exp?: number;
}

function decodeJWT(token: string): JWTPayload | null {
    try {
        const parts = token.split(".");
        if (parts.length !== 3) return null;
        const base64Url = parts[1];
        if (!base64Url) return null;
        const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
        const jsonPayload = decodeURIComponent(
            atob(base64)
                .split("")
                .map((c) => "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2))
                .join("")
        );
        return JSON.parse(jsonPayload);
    } catch (e) {
        return null;
    }
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
    const timestamp = new Date().toLocaleTimeString();
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
        <div class="console-req-header">➡️ POST ${url}</div>
        <div><strong>Headers:</strong> ${JSON.stringify(displayHeaders, null, 2)}</div>
        <div><strong>Request Body:</strong></div>
        <pre class="console-data">${JSON.stringify(body, null, 2)}</pre>
    `;
    debugOutput.insertBefore(entryDiv, debugOutput.firstChild);
}

function logResponse(status: number, statusText: string, data: any) {
    const timestamp = new Date().toLocaleTimeString();
    const entryDiv = document.createElement("div");
    entryDiv.className = "console-entry";
    const isErr = status >= 400;

    entryDiv.innerHTML = `
        <div class="console-resp-header" style="color: ${isErr ? "var(--accent-red)" : "var(--accent-green)"}">
            ⬅️ RESPONSE: ${status} ${statusText}
        </div>
        <div><strong>Body:</strong></div>
        <pre class="console-data">${typeof data === "string" ? data : JSON.stringify(data, null, 2)}</pre>
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
    html = html.replace(/```([\s\S]*?)```/gm, (_, code) => `<pre style="font-family: var(--font-mono); background: rgba(0,0,0,0.5); padding: 0.5rem; border-radius: 4px; overflow-x: auto; margin: 0.5rem 0; font-size: 0.8rem; border: 1px solid rgba(255,255,255,0.05); color: #a5f3fc;">${code.trim()}</pre>`);
    
    // Headers
    html = html.replace(/^### (.*$)/gim, '<h3 style="margin: 0.75rem 0 0.25rem 0; font-size: 1rem; color: #e9d5ff;">$1</h3>');
    html = html.replace(/^## (.*$)/gim, '<h2 style="margin: 1rem 0 0.5rem 0; font-size: 1.15rem; color: #c084fc;">$1</h2>');
    html = html.replace(/^# (.*$)/gim, '<h1 style="margin: 1.25rem 0 0.75rem 0; font-size: 1.35rem; color: #d8b4fe; border-bottom: 1px solid rgba(255,255,255,0.05); padding-bottom: 0.25rem;">$1</h1>');
    
    // Bold
    html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
    
    // Italic
    html = html.replace(/\*(.*?)\*/g, '<em>$1</em>');
    
    // Inline code
    html = html.replace(/`(.*?)`/g, '<code style="font-family: var(--font-mono); background: rgba(255,255,255,0.1); padding: 0.15rem 0.3rem; border-radius: 4px; font-size: 0.85rem; color: #f472b6;">$1</code>');
    
    // Linebreaks
    html = html.replace(/\n/g, "<br>");
    
    return html;
}

// --- AUTH STATE MANAGEMENT ---
function updateAuthStateUI() {
    if (currentToken) {
        authStatusBadge.className = "status-badge connected";
        authStatusText.textContent = "CONNECTED";
        userProfileCard.style.display = "block";

        if (authType === "dev") {
            btnLoginDev.style.display = "none";
            btnLogoutDev.style.display = "block";
            btnLoginJwt.style.display = "inline-flex";
            btnLogoutJwt.style.display = "none";
            
            profileName.textContent = "Local Notes User";
            profileEmail.textContent = "dev@localhost";
            profileId.textContent = "1";
            profileScopes.textContent = "notes:read, notes:write, notes:mcp";
            
            devTokenInput.value = currentToken;
        } else {
            btnLoginDev.style.display = "inline-flex";
            btnLogoutDev.style.display = "none";
            btnLoginJwt.style.display = "none";
            btnLogoutJwt.style.display = "block";

            jwtTokenInput.value = currentToken;

            const payload = decodeJWT(currentToken);
            if (payload) {
                profileName.textContent = payload.user?.name || payload.name || "JWT User";
                profileEmail.textContent = payload.user?.email || payload.email || "-";
                profileId.textContent = String(payload.user?.id || payload.sub || "-");
                
                const scopes = payload.user?.scopes || payload.scopes || [];
                profileScopes.textContent = scopes.length > 0 ? scopes.join(", ") : "no explicit scopes";
            } else {
                profileName.textContent = "JWT Token Holder";
                profileEmail.textContent = "-";
                profileId.textContent = "unknown";
                profileScopes.textContent = "n/a (token is not standard claims JWT)";
            }
        }
    } else {
        authStatusBadge.className = "status-badge";
        authStatusText.textContent = "NOT CONNECTED";
        userProfileCard.style.display = "none";
        
        btnLoginDev.style.display = "inline-flex";
        btnLogoutDev.style.display = "none";
        btnLoginJwt.style.display = "inline-flex";
        btnLogoutJwt.style.display = "none";
    }
}

function setToken(token: string, type: "dev" | "jwt") {
    currentToken = token;
    authType = type;
    localStorage.setItem(STORAGE_TOKEN_KEY, token);
    localStorage.setItem(STORAGE_TYPE_KEY, type);
    updateAuthStateUI();
    logToConsole(`Authentication set to ${type.toUpperCase()} mode.`);
}

function clearAuth() {
    currentToken = "";
    localStorage.removeItem(STORAGE_TOKEN_KEY);
    localStorage.removeItem(STORAGE_TYPE_KEY);
    updateAuthStateUI();
    logToConsole("Logged out, token cleared.");
}

// --- CONNECT SERVICE CALLS ---
async function callConnectRPC(methodName: string, requestData: any) {
    const url = `/notes.v1.NotesService/${methodName}`;
    const headers: Record<string, string> = {
        "Content-Type": "application/json",
        "Connect-Protocol-Version": "1",
    };

    if (currentToken) {
        headers["Authorization"] = `Bearer ${currentToken}`;
    }

    logRequest(url, headers, requestData);

    try {
        const response = await fetch(url, {
            method: "POST",
            headers: headers,
            body: JSON.stringify(requestData),
        });

        let responseData: any = null;
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
    } catch (e: any) {
        logToConsole(`Fetch failed: ${e.message}`, true);
        throw e;
    }
}

// --- NOTE RENDER FUNCTIONS ---
function populateNotesUI(notes: any[]) {
    notesContainer.innerHTML = "";
    currentNotesList = notes || [];
    notesCountLabel.textContent = `${currentNotesList.length} notes loaded`;

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
        
        const categoryLabel = note.category ? `<span class="note-category">${note.category}</span>` : "";
        
        const tags = note.tags || [];
        const tagsHtml = tags.map((t: string) => `<span class="tag-pill">${t}</span>`).join("");

        const createdStr = note.createdAt ? new Date(note.createdAt).toLocaleString() : "unknown";
        const updatedStr = note.updatedAt ? new Date(note.updatedAt).toLocaleString() : "unknown";
        
        card.innerHTML = `
            <div class="note-header">
                <div class="note-title-line">
                    <span class="note-title">${note.title || "Untitled"}</span>
                    ${categoryLabel}
                </div>
                <span class="note-id" data-copy-id="${note.id}" title="Click to copy ID">ID: ${note.id.substring(0, 8)}...</span>
            </div>
            <div class="note-body">${renderMarkdown(note.bodyMarkdown)}</div>
            <div class="note-footer">
                <div class="note-tags">${tagsHtml}</div>
                <div style="display: flex; gap: 0.5rem; align-items: center;">
                    <span style="font-size: 0.75rem; color: var(--text-muted);">Updated: ${updatedStr}</span>
                    <div class="note-actions">
                        <button class="btn-secondary btn-icon" data-action="add-tag" data-note-id="${note.id}">+ Tag</button>
                        <button class="btn-accent btn-icon" data-action="edit" data-note-id="${note.id}">Edit</button>
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
            navigator.clipboard.writeText(id);
            showToast("Copied note ID to clipboard!", "success");
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
                        const result = await callConnectRPC("AddTags", {
                            noteId: noteId,
                            tags: parsedTags
                        });
                        showToast("Tags added successfully!", "success");
                        // Refresh notes list
                        fetchNotesList();
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
}

function startEditMode(note: any) {
    // Fill fields
    (document.getElementById("update-id") as HTMLInputElement).value = note.id;
    (document.getElementById("update-title") as HTMLInputElement).value = note.title || "";
    (document.getElementById("update-category") as HTMLInputElement).value = note.category || "";
    (document.getElementById("update-tags") as HTMLInputElement).value = (note.tags || []).join(", ");
    (document.getElementById("update-body") as HTMLTextAreaElement).value = note.bodyMarkdown || "";

    // Show tab & navigate
    tabUpdateToggle.style.display = "block";
    tabUpdateToggle.click();
}

function stopEditMode() {
    tabUpdateToggle.style.display = "none";
    // Go to Create Note tab
    const tabCreate = document.querySelector<HTMLElement>("[data-op-tab='create']");
    if (tabCreate) tabCreate.click();
}

async function fetchNotesList() {
    const limit = parseInt(listLimitInput.value) || 10;
    try {
        const response = await callConnectRPC("ListRecentNotes", { limit: limit });
        populateNotesUI(response.notes);
        showToast("Notes list updated!", "success");
    } catch (err) {}
}

// --- INITIALIZE & ATTACH LISTENERS ---

// Tab Switching Helper
function initTabNavigation() {
    tabButtons.forEach((btn) => {
        btn.addEventListener("click", () => {
            const authTab = btn.getAttribute("data-auth-tab");
            const opTab = btn.getAttribute("data-op-tab");

            if (authTab) {
                // Switch auth tabs
                document.querySelectorAll("[data-auth-tab]").forEach(b => b.classList.remove("active"));
                btn.classList.add("active");
                authPanels.forEach((p) => {
                    p.classList.remove("active");
                    if (p.id === `auth-panel-${authTab}`) p.classList.add("active");
                });
            } else if (opTab) {
                // Switch op tabs
                document.querySelectorAll("[data-op-tab]").forEach(b => b.classList.remove("active"));
                btn.classList.add("active");
                opPanels.forEach((p) => {
                    p.classList.remove("active");
                    if (p.id === `op-panel-${opTab}`) p.classList.add("active");
                });
            }
        });
    });
}

// Setup Event Handlers
function setupEventHandlers() {
    // Console clear
    btnClearConsole.addEventListener("click", () => {
        debugOutput.innerHTML = "<div>Console cleared. Ready for next operation.</div>";
        showToast("Console cleared", "info");
    });

    // Dev Auth Click
    btnLoginDev.addEventListener("click", () => {
        const token = devTokenInput.value.trim();
        if (!token) {
            showToast("Dev token cannot be empty", "error");
            return;
        }
        setToken(token, "dev");
        showToast("Dev token applied!", "success");
    });

    btnLogoutDev.addEventListener("click", clearAuth);

    // JWT Auth Click
    btnLoginJwt.addEventListener("click", () => {
        const token = jwtTokenInput.value.trim();
        if (!token) {
            showToast("JWT token cannot be empty", "error");
            return;
        }
        setToken(token, "jwt");
        showToast("JWT token applied!", "success");
    });

    btnLogoutJwt.addEventListener("click", clearAuth);

    // Create Note Form Submission
    createNoteForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        const title = (document.getElementById("create-title") as HTMLInputElement).value.trim();
        const category = (document.getElementById("create-category") as HTMLInputElement).value.trim();
        const tagsStr = (document.getElementById("create-tags") as HTMLInputElement).value.trim();
        const bodyMarkdown = (document.getElementById("create-body") as HTMLTextAreaElement).value.trim();

        const tags = tagsStr.split(",").map(t => t.trim()).filter(Boolean);

        try {
            const response = await callConnectRPC("CreateNote", {
                title,
                category,
                tags,
                bodyMarkdown
            });
            
            showToast("Note created successfully!", "success");
            createNoteForm.reset();
            
            // Switch to list notes and refresh
            const tabList = document.querySelector<HTMLElement>("[data-op-tab='list']");
            if (tabList) tabList.click();
            fetchNotesList();
        } catch (err) {}
    });

    // List Notes Fetch Click
    btnSubmitList.addEventListener("click", fetchNotesList);

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

            populateNotesUI(response.notes);
            showToast(`Search returned ${response.notes?.length || 0} notes!`, "success");
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

        try {
            await callConnectRPC("UpdateNote", {
                noteId,
                title,
                category,
                tags,
                bodyMarkdown
            });

            showToast("Note updated successfully!", "success");
            stopEditMode();
            fetchNotesList();
        } catch (err) {}
    });
}

// On Page Load
window.addEventListener("DOMContentLoaded", () => {
    initTabNavigation();
    setupEventHandlers();
    
    // Check if we have standard params in URL callback (e.g. from OAuth callbacks, although not directly used, nice to log)
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.has("token") || urlParams.has("jwt")) {
        const token = urlParams.get("token") || urlParams.get("jwt") || "";
        if (token) {
            setToken(token, "jwt");
            window.history.replaceState({}, document.title, window.location.pathname);
            showToast("JWT token extracted from URL callback!", "success");
        }
    } else {
        updateAuthStateUI();
    }
});
