import { addMessages, init, getLocaleFromNavigator } from "svelte-i18n";
import de from "./de.json";
import en from "./en.json";

export function initialize() {
    addMessages("de", de);
    addMessages("en", en);

    init({
        fallbackLocale: "de",
        // initialLocale: getLocaleFromNavigator(),
        initialLocale: "en",
    });
}
