import { describe, expect, test } from "bun:test";
import { renderMarkdown } from "./markdown.ts";

describe("renderMarkdown", () => {
    test("returns placeholder for empty input", () => {
        expect(renderMarkdown("")).toBe("<em>No content</em>");
    });

    test("renders headings and inline formatting", () => {
        const html = renderMarkdown("# Title\n\n**bold** and `code`");
        expect(html).toContain("<h1");
        expect(html).toContain("<strong>bold</strong>");
        expect(html).toContain("<code");
    });

    test("escapes raw HTML in source", () => {
        const html = renderMarkdown("<img src=x onerror=alert(1)>");
        expect(html).not.toContain("<img");
        expect(html).toContain("&lt;img");
    });
});