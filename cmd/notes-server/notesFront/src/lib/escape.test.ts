import { describe, expect, test } from "bun:test";
import { escapeHtml } from "./escape.ts";

describe("escapeHtml", () => {
    test("escapes HTML metacharacters", () => {
        expect(escapeHtml(`<script>"'&"</script>`)).toBe("&lt;script&gt;&quot;&#39;&amp;&quot;&lt;/script&gt;");
    });

    test("handles nullish values", () => {
        expect(escapeHtml(null)).toBe("");
        expect(escapeHtml(undefined)).toBe("");
    });
});