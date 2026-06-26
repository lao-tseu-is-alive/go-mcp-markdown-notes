/** Ephemeral bottom-right notification toasts (auto-dismiss after 4s). */
import { dom } from "../dom.ts";

let toastTimeout: ReturnType<typeof setTimeout> | undefined;

export function showToast(message: string, type: "success" | "error" | "info" = "info") {
    if (toastTimeout) clearTimeout(toastTimeout);
    dom.toastMessage.textContent = message;
    dom.toastNotification.className = `toast toast-${type} show`;

    toastTimeout = setTimeout(() => {
        dom.toastNotification.classList.remove("show");
    }, 4000);
}