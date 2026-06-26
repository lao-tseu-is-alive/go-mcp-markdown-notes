// Shared wire-format and UI types for the notes Connect client.

/** Server auth mode exposed by GET /config. */
export type AuthMode = "jwt" | "dev";

/** User profile returned by the auth service token mint endpoint. */
export interface TokenUser {
    user_id: number;
    email: string;
    name: string;
    avatar_url?: string;
    is_admin: boolean;
}

/** Response from POST {authBaseUrl}/auth/token (JWT mode). */
export interface TokenResponse {
    token: string;
    expires_in_seconds: number;
    user: TokenUser;
}

/** Display metadata for a note lifecycle status badge. */
export interface NoteStatusInfo {
    key: string;
    label: string;
}

/**
 * Subset of the Connect Note message used by the SPA.
 * Field names follow the JSON/proto camelCase wire encoding.
 */
export interface NoteRecord {
    id?: string;
    title?: string;
    bodyMarkdown?: string;
    category?: string;
    tags?: string[];
    status?: string | number;
    updatedAt?: string;
}

/** Payload from GET /config on the notes-server. */
export interface AppConfig {
    authMode?: string;
    authBaseUrl?: string;
}