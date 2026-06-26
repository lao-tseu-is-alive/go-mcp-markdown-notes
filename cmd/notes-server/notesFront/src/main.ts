// Browser entry point. Wires DOMContentLoaded to the application bootstrap.
import { bootstrap } from "./app.ts";

window.addEventListener("DOMContentLoaded", () => {
    void bootstrap();
});