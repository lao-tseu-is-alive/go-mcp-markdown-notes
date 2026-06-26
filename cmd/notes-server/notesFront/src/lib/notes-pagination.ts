/**
 * Client-side pagination state for ListRecentNotes and SearchNotes.
 *
 * page_token values are opaque offset cursors returned by the server as next_page_token.
 * pageTokenStack records prior tokens so "Previous" can walk back without re-fetching metadata.
 */

export type NotesViewMode = "list" | "search";

export interface NotesViewState {
    mode: NotesViewMode;
    limit: number;
    pageToken: string;
    nextPageToken: string;
    totalSize: number;
    pageTokenStack: string[];
    searchQuery: string;
    searchCategory: string;
    searchTags: string[];
}

/** Mutable singleton — shared by list/search fetchers and the pagination bar. */
export const notesViewState: NotesViewState = {
    mode: "list",
    limit: 10,
    pageToken: "",
    nextPageToken: "",
    totalSize: 0,
    pageTokenStack: [],
    searchQuery: "",
    searchCategory: "",
    searchTags: [],
};

/** Starts a new list or search from page one, clearing the back-stack. */
export function resetNotesView(mode: NotesViewMode, limit: number) {
    notesViewState.mode = mode;
    notesViewState.limit = limit;
    notesViewState.pageToken = "";
    notesViewState.nextPageToken = "";
    notesViewState.totalSize = 0;
    notesViewState.pageTokenStack = [];
}

/** Persists search field values so Next/Previous re-use the same filter. */
export function setSearchCriteria(query: string, category: string, tags: string[]) {
    notesViewState.searchQuery = query;
    notesViewState.searchCategory = category;
    notesViewState.searchTags = tags;
}

/** Updates totals after a successful RPC; drives the pagination bar visibility. */
export function applyNotesPageResponse(totalSize: number, nextPageToken: string, pageSize: number) {
    notesViewState.totalSize = totalSize;
    notesViewState.nextPageToken = nextPageToken;
    notesViewState.limit = pageSize;
}

export function currentPageNumber(): number {
    return notesViewState.pageTokenStack.length + 1;
}

/** Advances to the next page; returns the token to pass to the RPC, or null when exhausted. */
export function goToNextNotesPage(): string | null {
    if (!notesViewState.nextPageToken) return null;
    notesViewState.pageTokenStack.push(notesViewState.pageToken);
    notesViewState.pageToken = notesViewState.nextPageToken;
    return notesViewState.pageToken;
}

/** Walks back one page using the token stack; returns the token to pass to the RPC. */
export function goToPreviousNotesPage(): string | null {
    if (notesViewState.pageTokenStack.length === 0) return null;
    notesViewState.pageToken = notesViewState.pageTokenStack.pop() ?? "";
    return notesViewState.pageToken;
}

/** Human-readable range label for the pagination bar. */
export function notesPageSummary(displayedCount: number): string {
    if (notesViewState.totalSize <= 0) {
        return displayedCount > 0 ? `Page ${currentPageNumber()}` : "";
    }
    const offset = notesViewState.pageTokenStack.length * notesViewState.limit;
    const start = offset + 1;
    const end = offset + displayedCount;
    return `Showing ${start}–${end} of ${notesViewState.totalSize}`;
}

export function shouldShowPagination(displayedCount: number): boolean {
    return displayedCount > 0 && (
        notesViewState.pageTokenStack.length > 0 ||
        notesViewState.nextPageToken !== "" ||
        notesViewState.totalSize > displayedCount
    );
}