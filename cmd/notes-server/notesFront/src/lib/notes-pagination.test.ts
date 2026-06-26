import { describe, expect, test, beforeEach } from "bun:test";
import {
    applyNotesPageResponse,
    currentPageNumber,
    goToNextNotesPage,
    goToPreviousNotesPage,
    notesPageSummary,
    notesViewState,
    resetNotesView,
    shouldShowPagination,
} from "./notes-pagination.ts";

beforeEach(() => {
    resetNotesView("list", 10);
});

describe("notes pagination state", () => {
    test("tracks page transitions", () => {
        applyNotesPageResponse(25, "10", 10);
        expect(goToNextNotesPage()).toBe("10");
        expect(currentPageNumber()).toBe(2);
        expect(goToPreviousNotesPage()).toBe("");
        expect(currentPageNumber()).toBe(1);
    });

    test("builds page summary from totals", () => {
        notesViewState.pageTokenStack.push("");
        notesViewState.pageToken = "10";
        applyNotesPageResponse(25, "20", 10);
        expect(notesPageSummary(10)).toBe("Showing 11–20 of 25");
    });

    test("shows pagination when more pages exist", () => {
        applyNotesPageResponse(25, "10", 10);
        expect(shouldShowPagination(10)).toBe(true);
        applyNotesPageResponse(5, "", 10);
        expect(shouldShowPagination(5)).toBe(false);
    });
});