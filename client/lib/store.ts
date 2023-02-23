import { writable } from "svelte/store";

export const isLoading = writable(false);
export const uiDisabled = writable(false);
export const apiErrorMessage = writable("");

export const scrollToSetId = writable<number | null>(null);

export type PreselectedExerciseSet = {
    setId: number | null;
    exerciseId: number;
};

export const preselectExerciseSet = writable<PreselectedExerciseSet | null>(null);
