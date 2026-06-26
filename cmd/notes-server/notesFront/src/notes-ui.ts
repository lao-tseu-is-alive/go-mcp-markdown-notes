/**
 * Note list rendering, inline actions, and paginated list/search fetchers.
 *
 * populateNotesUI builds note cards from RPC results. List and search share the
 * same display and pagination bar; notesViewState tracks which RPC to re-call.
 */
import type { NoteRecord } from "./types.ts";
import { dom } from "./dom.ts";
import { callConnectRPC } from "./connect.ts";
import { escapeHtml } from "./lib/escape.ts";
import { renderMarkdown } from "./lib/markdown.ts";
import { noteStatusInfo, statusToInt } from "./lib/note-status.ts";
import {
    applyNotesPageResponse,
    goToNextNotesPage,
    goToPreviousNotesPage,
    notesPageSummary,
    notesViewState,
    resetNotesView,
    setSearchCriteria,
    shouldShowPagination,
} from "./lib/notes-pagination.ts";
import { activateWorkspaceTab } from "./ui/tabs.ts";
import { showToast } from "./ui/toast.ts";
import { logToConsole } from "./ui/console-log.ts";

/** In-memory copy of the currently displayed page (used by edit/delete handlers). */
export let currentNotesList: NoteRecord[] = [];

function updatePaginationBar(displayedCount: number) {
    const visible = shouldShowPagination(displayedCount);
    dom.notesPagination.style.display = visible ? "flex" : "none";
    dom.btnNotesPrev.disabled = notesViewState.pageTokenStack.length === 0;
    dom.btnNotesNext.disabled = notesViewState.nextPageToken === "";
    dom.notesPageInfo.textContent = notesPageSummary(displayedCount);
}

/** Renders note cards into #notes-container and updates count/pagination labels. */
export function populateNotesUI(notes: NoteRecord[], totalSize?: number, nextPageToken = "") {
    dom.notesContainer.innerHTML = "";
    currentNotesList = notes || [];
    const total = totalSize ?? currentNotesList.length;
    applyNotesPageResponse(total, nextPageToken, notesViewState.limit);

    dom.notesCountLabel.textContent = total > currentNotesList.length
        ? `${currentNotesList.length} of ${total} notes`
        : `${currentNotesList.length} notes loaded`;

    if (currentNotesList.length === 0) {
        dom.notesContainer.innerHTML = `
            <div style="text-align: center; color: var(--text-muted); padding: 3rem 0;">
                <span style="font-size: 2.5rem; display: block; margin-bottom: 0.5rem;">📂</span>
                No notes found.
            </div>
        `;
        updatePaginationBar(0);
        return;
    }

    currentNotesList.forEach((note) => {
        const card = document.createElement("div");
        card.className = "note-card";

        const noteId = String(note.id ?? "");
        const safeNoteId = escapeHtml(noteId);
        const categoryLabel = note.category ? `<span class="note-category">${escapeHtml(note.category)}</span>` : "";

        const si = noteStatusInfo(note.status);
        const statusBadge = si ? `<span class="note-status-${si.key}">${si.label}</span>` : "";

        const tags = note.tags || [];
        const tagsHtml = tags.map((tag) => `<span class="tag-pill">${escapeHtml(tag)}</span>`).join("");

        const updatedStr = note.updatedAt ? new Date(note.updatedAt).toLocaleString() : "unknown";

        card.innerHTML = `
            <div class="note-header">
                <div class="note-title-line">
                    <span class="note-title">${escapeHtml(note.title || "Untitled")}</span>
                    ${categoryLabel}${statusBadge}
                </div>
                <span class="note-id" data-copy-id="${safeNoteId}" title="Click to copy ID">ID: ${escapeHtml(noteId.substring(0, 8))}...</span>
            </div>
            <div class="note-body">${renderMarkdown(note.bodyMarkdown || "")}</div>
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

        dom.notesContainer.appendChild(card);
    });

    attachNoteCardListeners();
    updatePaginationBar(currentNotesList.length);
}

/** Wires per-card copy, add-tag, edit, and delete actions after each render. */
function attachNoteCardListeners() {
    dom.notesContainer.querySelectorAll("[data-copy-id]").forEach((el) => {
        el.addEventListener("click", (e) => {
            const id = (e.target as HTMLElement).getAttribute("data-copy-id") || "";
            void navigator.clipboard.writeText(id)
                .then(() => showToast("Copied note ID to clipboard!", "success"))
                .catch((err) => {
                    const message = err instanceof Error ? err.message : String(err);
                    logToConsole(`Failed to copy note ID: ${message}`, true);
                    showToast("Failed to copy note ID.", "error");
                });
        });
    });

    dom.notesContainer.querySelectorAll("[data-action='add-tag']").forEach((el) => {
        el.addEventListener("click", async (e) => {
            const noteId = (e.target as HTMLElement).getAttribute("data-note-id") || "";
            const tagsPrompt = prompt("Enter tag(s) to add (comma-separated):");
            if (!tagsPrompt) return;

            const parsedTags = tagsPrompt.split(",").map((tag) => tag.trim()).filter(Boolean);
            if (parsedTags.length === 0) return;

            try {
                await callConnectRPC("AddTags", { noteId, tags: parsedTags });
                showToast("Tags added successfully!", "success");
                await refreshCurrentNotesView();
            } catch {
                // RPC layer already surfaced the error.
            }
        });
    });

    dom.notesContainer.querySelectorAll("[data-action='edit']").forEach((el) => {
        el.addEventListener("click", (e) => {
            const noteId = (e.target as HTMLElement).getAttribute("data-note-id") || "";
            const note = currentNotesList.find((item) => item.id === noteId);
            if (note) startEditMode(note);
        });
    });

    dom.notesContainer.querySelectorAll("[data-action='delete']").forEach((el) => {
        el.addEventListener("click", async (e) => {
            const noteId = (e.target as HTMLElement).getAttribute("data-note-id") || "";
            const note = currentNotesList.find((item) => item.id === noteId);
            const title = note?.title || noteId;
            if (!confirm(`Delete note "${title}"? This cannot be undone.`)) return;

            try {
                await callConnectRPC("DeleteNote", { noteId });
                showToast("Note deleted.", "success");
                await refreshCurrentNotesView();
            } catch {
                // RPC layer already surfaced the error.
            }
        });
    });
}

/** Switches the Create tab into full-replacement update mode for the given note. */
export function startEditMode(note: NoteRecord) {
    (document.getElementById("update-id") as HTMLInputElement).value = note.id || "";
    (document.getElementById("update-title") as HTMLInputElement).value = note.title || "";
    (document.getElementById("update-category") as HTMLInputElement).value = note.category || "";
    (document.getElementById("update-tags") as HTMLInputElement).value = (note.tags || []).join(", ");
    (document.getElementById("update-body") as HTMLTextAreaElement).value = note.bodyMarkdown || "";
    (document.getElementById("update-status") as HTMLSelectElement).value = String(statusToInt(note.status));

    dom.createNotePanel.classList.remove("active");
    dom.updateNotePanel.classList.add("active");
    activateWorkspaceTab("create");
}

export function stopEditMode() {
    dom.updateNotePanel.classList.remove("active");
    dom.createNotePanel.classList.add("active");
}

function listRequestPayload(pageToken: string) {
    const payload: { limit: number; pageToken?: string } = { limit: notesViewState.limit };
    if (pageToken) payload.pageToken = pageToken;
    return payload;
}

function searchRequestPayload(pageToken: string) {
    const payload: {
        query: string;
        category: string;
        tags: string[];
        limit: number;
        pageToken?: string;
    } = {
        query: notesViewState.searchQuery,
        category: notesViewState.searchCategory,
        tags: notesViewState.searchTags,
        limit: notesViewState.limit,
    };
    if (pageToken) payload.pageToken = pageToken;
    return payload;
}

/**
 * Calls ListRecentNotes. Pass reset=true when starting a fresh list (clears page stack).
 */
export async function fetchNotesList(pageToken = "", reset = false) {
    const limit = parseInt(dom.listLimitInput.value) || 10;
    if (reset) resetNotesView("list", limit);
    notesViewState.limit = limit;
    notesViewState.pageToken = pageToken;

    try {
        const response = await callConnectRPC("ListRecentNotes", listRequestPayload(pageToken));
        const total = response.pageResponse?.totalSize ?? response.notes?.length ?? 0;
        const nextToken = response.pageResponse?.nextPageToken ?? "";
        populateNotesUI(response.notes, total, nextToken);
        activateWorkspaceTab("notes");
        showToast("Notes list updated!", "success");
    } catch {
        // RPC layer already surfaced the error.
    }
}

/**
 * Calls SearchNotes. Pass reset=true on form submit (clears page stack and stores criteria).
 */
export async function searchNotes(pageToken = "", reset = false) {
    const query = (document.getElementById("search-query") as HTMLInputElement).value.trim();
    const category = (document.getElementById("search-category") as HTMLInputElement).value.trim();
    const tagsStr = (document.getElementById("search-tags") as HTMLInputElement).value.trim();
    const limit = parseInt((document.getElementById("search-limit") as HTMLInputElement).value) || 10;
    const tags = tagsStr.split(",").map((tag) => tag.trim()).filter(Boolean);

    if (reset) {
        resetNotesView("search", limit);
        setSearchCriteria(query, category, tags);
    }
    notesViewState.limit = limit;
    notesViewState.pageToken = pageToken;

    try {
        const response = await callConnectRPC("SearchNotes", searchRequestPayload(pageToken));
        const total = response.pageResponse?.totalSize ?? response.notes?.length ?? 0;
        const nextToken = response.pageResponse?.nextPageToken ?? "";
        populateNotesUI(response.notes, total, nextToken);
        activateWorkspaceTab("notes");
        showToast(`Search: ${response.notes?.length || 0} of ${total} notes`, "success");
    } catch {
        // RPC layer already surfaced the error.
    }
}

/** Re-fetches the current list or search page after a mutating card action. */
export async function refreshCurrentNotesView() {
    if (notesViewState.mode === "search") {
        await searchNotes(notesViewState.pageToken);
        return;
    }
    await fetchNotesList(notesViewState.pageToken);
}

/** Binds Previous/Next buttons to the active list or search fetcher. */
export function initNotesPaginationControls() {
    dom.btnNotesPrev.addEventListener("click", () => {
        const token = goToPreviousNotesPage();
        if (token === null) return;
        void (notesViewState.mode === "search"
            ? searchNotes(token)
            : fetchNotesList(token));
    });

    dom.btnNotesNext.addEventListener("click", () => {
        const token = goToNextNotesPage();
        if (token === null) return;
        void (notesViewState.mode === "search"
            ? searchNotes(token)
            : fetchNotesList(token));
    });
}