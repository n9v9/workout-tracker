import { getLocaleFromNavigator } from "svelte-i18n";
import { writable, type Writable } from "svelte/store";
import { changeLanguage } from "./i18n";

export const isLoading = writable(false);
export const uiDisabled = writable(false);
export const apiErrorMessage = writable("");

export const scrollToSetId = writable<number | null>(null);

export type PreselectedExerciseSet = {
    setId: number | null;
    exerciseId: number;
};

export const preselectExerciseSet = writable<PreselectedExerciseSet | null>(null);

export type Unit = "kg" | "lbs";

export type Language = "en" | "de";

export type Settings = {
    unit: Unit;
    language: Language;
};

export const settings = persistentStore<Settings>(
    "settings",
    {
        language: getLocaleFromNavigator() === "de" ? "de" : "en",
        unit: "kg",
    },
    settings => {
        changeLanguage(settings.language);
    },
);

function persistentStore<T>(key: string, initial: T, afterWrite?: (value: T) => void): Writable<T> {
    if (localStorage.getItem(key) === null) {
        localStorage.setItem(key, JSON.stringify(initial));
    }

    const saved = JSON.parse(localStorage.getItem(key)) as T;
    const { subscribe, set, update } = writable(saved);

    return {
        subscribe,
        set: (value: T) => {
            localStorage.setItem(key, JSON.stringify(value));
            if (afterWrite) {
                afterWrite(value);
            }
            return set(value);
        },
        update,
    };
}
