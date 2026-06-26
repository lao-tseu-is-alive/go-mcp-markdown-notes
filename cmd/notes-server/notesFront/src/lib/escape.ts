/**
 * Escape untrusted text before inserting into HTML templates.
 * Used for note fields and debug console output (XSS mitigation).
 */
export function escapeHtml(value: unknown): string {
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