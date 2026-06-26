import { describe, expect, test } from "bun:test";
import { noteStatusInfo, statusToInt } from "./note-status.ts";

describe("noteStatusInfo", () => {
    test("maps numeric and proto enum values", () => {
        expect(noteStatusInfo(2)?.key).toBe("active");
        expect(noteStatusInfo("NOTE_STATUS_ARCHIVED")?.label).toBe("Archived");
    });

    test("returns null for unknown values", () => {
        expect(noteStatusInfo(99)).toBeNull();
    });
});

describe("statusToInt", () => {
    test("preserves valid numeric statuses", () => {
        expect(statusToInt(3)).toBe(3);
    });

    test("maps proto names and defaults to active", () => {
        expect(statusToInt("NOTE_STATUS_DRAFT")).toBe(1);
        expect(statusToInt(undefined)).toBe(2);
    });
});