import "./app.css";
import App from "./App.svelte";

import { initialize } from "../lib/i18n";

initialize();

const app = new App({
    target: document.getElementById("app"),
});

export default app;
