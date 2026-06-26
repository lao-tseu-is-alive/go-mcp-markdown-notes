/**
 * Maps NoteStatus enum values (numeric or proto string names) to UI badge metadata.
 * Mirrors notes.v1.NoteStatus in proto/notes/v1/notes.proto.
 */
import type { NoteStatusInfo } from "../types.ts";

const STATUS_INFO: Record<string | number, NoteStatusInfo> = {
    1: { key: "draft", label: "Draft" },
    NOTE_STATUS_DRAFT: { key: "draft", label: "Draft" },
    2: { key: "active", label: "Active" },
    NOTE_STATUS_ACTIVE: { key: "active", label: "Active" },
    3: { key: "final", label: "Final" },
    NOTE_STATUS_FINAL: { key: "final", label: "Final" },
    4: { key: "archived", label: "Archived" },
    NOTE_STATUS_ARCHIVED: { key: "archived", label: "Archived" },
};

const STATUS_TO_INT: Record<string, number> = {
    NOTE_STATUS_DRAFT: 1,
    NOTE_STATUS_ACTIVE: 2,
    NOTE_STATUS_FINAL: 3,
    NOTE_STATUS_ARCHIVED: 4,
};

/** Returns badge key/label for a wire status, or null when unknown. */
export function noteStatusInfo(status: string | number | undefined): NoteStatusInfo | null {
    return status !== undefined ? (STATUS_INFO[status] ?? null) : null;
}

/** Converts a wire status to the integer sent in CreateNote/UpdateNote requests. Defaults to Active (2). */
export function statusToInt(status: string | number | undefined): number {
    if (typeof status === "number" && status >= 1 && status <= 4) return status;
    return typeof status === "string" ? (STATUS_TO_INT[status] ?? 2) : 2;
}