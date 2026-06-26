/**
 * Authentication state and flows for the notes SPA.
 *
 * jwt mode: short-lived JWTs are minted silently from the SSO session cookie
 *           (in memory only — never stored in localStorage).
 * dev mode: a static token is entered manually for local hacking.
 */
import type { AuthMode, TokenResponse, TokenUser } from "./types.ts";
import { dom } from "./dom.ts";
import { logToConsole } from "./ui/console-log.ts";
import { showToast } from "./ui/toast.ts";

export const authState = {
    mode: "jwt" as AuthMode,
    authBaseUrl: "",
    currentToken: "",
    currentUser: null as TokenUser | null,
    remintTimer: null as ReturnType<typeof setTimeout> | null,
};

/** Toggles SSO/dev panels, profile card, and scope display from authState. */
export function updateAuthStateUI() {
    const connected = authState.currentToken !== "";
    dom.authStatusBadge.className = connected ? "status-badge connected" : "status-badge";
    dom.authStatusText.textContent = connected ? "CONNECTED" : "NOT CONNECTED";
    dom.userProfileCard.style.display = connected ? "block" : "none";

    if (authState.mode === "dev") {
        dom.devPanel.style.display = "block";
        dom.ssoPanel.style.display = "none";
        dom.btnLoginDev.style.display = connected ? "none" : "inline-flex";
        dom.btnLogoutDev.style.display = connected ? "block" : "none";
        if (connected) {
            dom.profileName.textContent = "Local Notes User";
            dom.profileEmail.textContent = "dev@localhost";
            dom.profileId.textContent = "1";
            dom.profileScopes.textContent = "notes:read, notes:write, notes:mcp";
        }
        return;
    }

    dom.devPanel.style.display = "none";
    dom.ssoPanel.style.display = "block";
    dom.btnSignIn.style.display = connected ? "none" : "inline-flex";
    dom.btnSignOut.style.display = connected ? "inline-flex" : "none";
    dom.manageTokensLink.style.display = connected ? "inline-flex" : "none";
    dom.ssoHint.textContent = connected
        ? "You are signed in. Tokens are refreshed silently from your SSO session."
        : "Sign in with your Google or GitHub account. You will be redirected to the auth service and back.";

    if (connected && authState.currentUser) {
        dom.profileName.textContent = authState.currentUser.name || "-";
        dom.profileEmail.textContent = authState.currentUser.email || "-";
        dom.profileId.textContent = String(authState.currentUser.user_id ?? "-");
        dom.profileScopes.textContent = authState.currentUser.is_admin
            ? "notes:read, notes:write, notes:mcp, notes:admin"
            : "notes:read, notes:write, notes:mcp";
    }
}

/**
 * Silently obtains a fresh JWT from the auth service using the SSO session cookie.
 * Returns true when a token was obtained.
 */
export async function mintToken(): Promise<boolean> {
    try {
        const res = await fetch(`${authState.authBaseUrl}/auth/token`, { credentials: "include" });
        if (res.status === 401) {
            clearAuth();
            logToConsole("No SSO session: sign-in required.");
            return false;
        }
        if (!res.ok) {
            logToConsole(`Token mint failed with HTTP ${res.status}`, true);
            return false;
        }
        const data: TokenResponse = await res.json();
        authState.currentToken = data.token;
        authState.currentUser = data.user;
        scheduleRemint(data.expires_in_seconds);
        updateAuthStateUI();
        logToConsole(`Token minted for ${data.user.email}, valid ${data.expires_in_seconds}s.`);
        return true;
    } catch (e) {
        const message = e instanceof Error ? e.message : String(e);
        logToConsole(`Token mint failed: ${message} (is the auth service up at ${authState.authBaseUrl}?)`, true);
        return false;
    }
}

/** Schedules automatic re-mint at ~80% of JWT lifetime to avoid expiry during RPC calls. */
export function scheduleRemint(expiresInSeconds: number) {
    if (authState.remintTimer) clearTimeout(authState.remintTimer);
    const delayMs = Math.max(expiresInSeconds * 800, 30_000);
    authState.remintTimer = setTimeout(() => { void mintToken(); }, delayMs);
}

/** Redirects to the external auth service login page. */
export function signIn() {
    window.location.href = `${authState.authBaseUrl}/auth/login?redirect_uri=${encodeURIComponent(window.location.href)}`;
}

export async function signOut() {
    try {
        await fetch(`${authState.authBaseUrl}/auth/logout`, { method: "POST", credentials: "include" });
    } catch (e) {
        const message = e instanceof Error ? e.message : String(e);
        logToConsole(`Logout call failed: ${message}`, true);
    }
    clearAuth();
    showToast("Signed out.", "info");
}

export function clearAuth() {
    authState.currentToken = "";
    authState.currentUser = null;
    if (authState.remintTimer) clearTimeout(authState.remintTimer);
    updateAuthStateUI();
}

/** Applies a manually entered dev token (dev mode only). */
export function applyDevToken(token: string) {
    authState.currentToken = token;
    updateAuthStateUI();
}