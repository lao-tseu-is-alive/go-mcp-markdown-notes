import { describe, expect, test } from "bun:test";
import { parseCommaSeparatedTags } from "./tags.ts";

describe("parseCommaSeparatedTags", () => {
    test("splits trims and drops blanks", () => {
        expect(parseCommaSeparatedTags(" go , , mcp ,notes ")).toEqual(["go", "mcp", "notes"]);
    });

    test("returns empty array for blank input", () => {
        expect(parseCommaSeparatedTags("   ")).toEqual([]);
    });
});