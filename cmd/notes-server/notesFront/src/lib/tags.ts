/** Normalizes the comma-separated tag inputs used by create/update/search forms. */
export function parseCommaSeparatedTags(raw: string): string[] {
    return raw.split(",").map((tag) => tag.trim()).filter(Boolean);
}