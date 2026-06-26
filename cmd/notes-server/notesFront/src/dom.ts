/**
 * Typed references to static DOM nodes declared in index.html.
 * Resolved once at module load (script runs at end of <body>).
 */

function requireElement<T extends HTMLElement>(id: string): T {
    const element = document.getElementById(id);
    if (!element) {
        throw new Error(`missing required element #${id}`);
    }
    return element as T;
}

export const dom = {
    workspaceTabButtons: document.querySelectorAll<HTMLElement>("[data-workspace-tab]"),
    workspacePanels: document.querySelectorAll<HTMLElement>("[data-workspace-panel]"),
    ssoPanel: requireElement<HTMLElement>("auth-panel-sso"),
    devPanel: requireElement<HTMLElement>("auth-panel-dev"),
    devTokenInput: requireElement<HTMLInputElement>("dev-token-input"),
    btnLoginDev: requireElement<HTMLButtonElement>("btn-login-dev"),
    btnLogoutDev: requireElement<HTMLButtonElement>("btn-logout-dev"),
    btnSignIn: requireElement<HTMLButtonElement>("btn-sign-in"),
    btnSignOut: requireElement<HTMLButtonElement>("btn-sign-out"),
    ssoHint: requireElement<HTMLElement>("sso-hint"),
    manageTokensLink: requireElement<HTMLAnchorElement>("lnk-manage-tokens"),
    authStatusBadge: requireElement<HTMLElement>("auth-status"),
    authStatusText: requireElement<HTMLElement>("auth-status-text"),
    userProfileCard: requireElement<HTMLElement>("user-profile-card"),
    profileName: requireElement<HTMLElement>("profile-name"),
    profileEmail: requireElement<HTMLElement>("profile-email"),
    profileId: requireElement<HTMLElement>("profile-id"),
    profileScopes: requireElement<HTMLElement>("profile-scopes"),
    createNoteForm: requireElement<HTMLFormElement>("create-note-form"),
    searchNoteForm: requireElement<HTMLFormElement>("search-note-form"),
    updateNoteForm: requireElement<HTMLFormElement>("update-note-form"),
    btnSubmitList: requireElement<HTMLButtonElement>("btn-submit-list"),
    listLimitInput: requireElement<HTMLInputElement>("list-limit"),
    notesContainer: requireElement<HTMLElement>("notes-container"),
    notesCountLabel: requireElement<HTMLElement>("notes-count"),
    debugOutput: requireElement<HTMLElement>("debug-output"),
    btnClearConsole: requireElement<HTMLElement>("btn-clear-console"),
    toastNotification: requireElement<HTMLElement>("toast-notification"),
    toastMessage: requireElement<HTMLElement>("toast-message"),
    createNotePanel: requireElement<HTMLElement>("op-panel-create"),
    updateNotePanel: requireElement<HTMLElement>("op-panel-update"),
    btnCancelUpdate: requireElement<HTMLButtonElement>("btn-cancel-update"),
    notesPagination: requireElement<HTMLElement>("notes-pagination"),
    btnNotesPrev: requireElement<HTMLButtonElement>("btn-notes-prev"),
    btnNotesNext: requireElement<HTMLButtonElement>("btn-notes-next"),
    notesPageInfo: requireElement<HTMLElement>("notes-page-info"),
};