/** Workspace tab switching (Authentication / Create / List / Debug). */
import { dom } from "../dom.ts";

export function initTabNavigation() {
    dom.workspaceTabButtons.forEach((btn) => {
        btn.addEventListener("click", () => {
            const workspaceTab = btn.getAttribute("data-workspace-tab");
            if (workspaceTab) activateWorkspaceTab(workspaceTab);
        });
    });
}

/** Activates one workspace tab and its matching panel; updates aria-selected. */
export function activateWorkspaceTab(workspaceTab: string) {
    dom.workspaceTabButtons.forEach((btn) => {
        const active = btn.getAttribute("data-workspace-tab") === workspaceTab;
        btn.classList.toggle("active", active);
        btn.setAttribute("aria-selected", String(active));
    });

    dom.workspacePanels.forEach((panel) => {
        panel.classList.toggle("active", panel.getAttribute("data-workspace-panel") === workspaceTab);
    });
}