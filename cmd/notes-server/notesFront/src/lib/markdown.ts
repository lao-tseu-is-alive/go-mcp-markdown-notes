/**
 * Lightweight Markdown-to-HTML renderer for note preview cards.
 * Not a full CommonMark implementation — covers headings, emphasis, code, and line breaks.
 * Source HTML is escaped before pattern substitution.
 */
export function renderMarkdown(md: string): string {
    if (!md) return "<em>No content</em>";
    let html = md
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");

    html = html.replace(/```([\s\S]*?)```/gm, (_, code) => `<pre style="font-family: var(--font-mono),monospace; background: rgba(0,0,0,0.5); padding: 0.5rem; border-radius: 4px; overflow-x: auto; margin: 0.5rem 0; font-size: 0.8rem; border: 1px solid rgba(255,255,255,0.05); color: #a5f3fc;">${code.trim()}</pre>`);
    html = html.replace(/^### (.*$)/gim, '<h3 style="margin: 0.75rem 0 0.25rem 0; font-size: 1rem; color: #e9d5ff;">$1</h3>');
    html = html.replace(/^## (.*$)/gim, '<h2 style="margin: 1rem 0 0.5rem 0; font-size: 1.15rem; color: #c084fc;">$1</h2>');
    html = html.replace(/^# (.*$)/gim, '<h1 style="margin: 1.25rem 0 0.75rem 0; font-size: 1.35rem; color: #d8b4fe; border-bottom: 1px solid rgba(255,255,255,0.05); padding-bottom: 0.25rem;">$1</h1>');
    html = html.replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>");
    html = html.replace(/\*(.*?)\*/g, "<em>$1</em>");
    html = html.replace(/`(.*?)`/g, '<code style="font-family: var(--font-mono),monospace; background: rgba(255,255,255,0.1); padding: 0.15rem 0.3rem; border-radius: 4px; font-size: 0.85rem; color: #f472b6;">$1</code>');
    html = html.replace(/\n/g, "<br>");

    return html;
}