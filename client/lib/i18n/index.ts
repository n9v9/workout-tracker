import { addMessages, init } from "svelte-i18n";
import { get } from "svelte/store";
import { settings, type Language } from "../store";
import de from "./de.json";
import en from "./en.json";

export function initialize() {
    addMessages("de", de);
    addMessages("en", en);

    init({
        fallbackLocale: "de",
        initialLocale: get(settings).language,
    });
}

export function changeLanguage(language: Language) {
    init({
        fallbackLocale: "de",
        initialLocale: language,
    });
}
