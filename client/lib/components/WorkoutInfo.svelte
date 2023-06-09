<script lang="ts">
    import { onMount } from "svelte";
    import { push } from "svelte-spa-router";
    import { api } from "../api/service";
    import Title from "../components/Title.svelte";
    import Notification from "./Notification.svelte";
    import type { ExerciseSet } from "../api/types";
    import Button from "./Button.svelte";
    import Timer from "./Timer.svelte";
    import { scrollToSetId, settings } from "../store";
    import { _ } from "svelte-i18n";
    import MultilineInput from "./MultilineInput.svelte";
    import Modal from "./Modal.svelte";

    export let params: { id: string };

    let id = parseInt(params.id);
    let sets: ExerciseSet[] = [];
    let latest: ExerciseSet | null = null;
    let firstExerciseOfLatestSet: ExerciseSet | null = null;
    let showNoteSavedModal = false;
    let inputNote = "";
    let updateNote: (text: string) => void;

    onMount(async () => {
        const workout = await api.getWorkout(id);
        updateNote(workout.note);

        sets = await api.getSetsByWorkoutId(id);

        if (sets.length > 0) {
            sets.sort((a, b) => {
                if (a.date < b.date) return 1;
                if (a.date > b.date) return -1;
                return 0;
            });
            latest = sets[0];

            let ptr: ExerciseSet;

            for (let i = 0; i < sets.length; i++) {
                ptr = sets[i];
                if (latest.exerciseId !== sets[i].exerciseId) {
                    firstExerciseOfLatestSet = sets[i - 1];
                    break;
                }
            }

            if (firstExerciseOfLatestSet === null) {
                firstExerciseOfLatestSet = ptr;
            }
        }

        if ($scrollToSetId !== null) {
            setTimeout(() => {
                const element = document.querySelector(`#set-${$scrollToSetId}`);
                element.scrollIntoView({
                    behavior: "auto",
                    block: "center",
                    inline: "center",
                });
                setTimeout(() => {
                    $scrollToSetId = null;
                }, 1500); // Must be in sync with the duration in the keyframe.
            }, 0);
        }
    });

    function editSet(set: ExerciseSet) {
        push(`/workouts/${id}/sets/${set.id}`);
    }

    async function saveNote() {
        await api.updateWorkoutMetaData(id, inputNote);
        updateNote(inputNote);
        showNoteSavedModal = true;
    }
</script>

<Title text="Workout" />

<div class="block">
    <Button classes="button is-fullwidth is-primary" click={() => push(`/workouts/${id}/sets/add`)}>
        <span class="icon">
            <i class="bi bi-plus-lg" />
        </span>
        <span>{$_("new_set")}</span>
    </Button>
    <Button classes="button is-fullwidth mt-2 is-link" click={() => push("/")}>
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>{$_("back_to_workout_list")}</span>
    </Button>
</div>

{#if latest !== null && firstExerciseOfLatestSet !== null}
    <div class="block level is-mobile">
        <div class="level-item">
            <Timer text={$_("last_set")} reference={latest.date} />
        </div>
        <div class="level-item">
            <Timer text={$_("current_exercise")} reference={firstExerciseOfLatestSet.date} />
        </div>
    </div>
{/if}

<div class="block">
    <p class="is-size-5 mb-2">{$_("note")}</p>

    <div class="field">
        <div class="control">
            <MultilineInput
                on:change={e => (inputNote = e.detail.text)}
                bind:setText={updateNote} />
        </div>
    </div>

    <div class="field">
        <div class="control">
            <Button click={saveNote} classes="button is-fullwidth is-light is-primary"
                >{$_("save")}</Button>
        </div>
    </div>
</div>

<div class="block">
    <p class="is-size-5 mb-2">{$_("completed_sets")} ({sets.length})</p>

    {#if sets.length > 0}
        <table class="table is-fullwidth is-striped is-hoverable is-bordered mb-3">
            <thead>
                <tr>
                    <th class="has-background-white">{$_("exercise")}</th>
                    <th class="has-background-white">
                        <abbr title={$_("number_repetitions")}
                            >{$_("abbr_number_repetitions")}</abbr>
                    </th>
                    <th class="has-background-white">
                        <abbr title={$_(`weight_in_${$settings.unit}`)}
                            >{$_(`abbr_weight_in_${$settings.unit}`)}</abbr>
                    </th>
                </tr>
            </thead>
            <tbody>
                {#each sets as set}
                    <tr
                        id="set-{set.id}"
                        class={set.id === $scrollToSetId ? "highlight-exercise-set" : ""}
                        on:click={() => editSet(set)}>
                        <td>
                            {set.exerciseName}
                            {#if set.note}
                                <i class="bi bi-chat-left-text has-text-link" />
                            {/if}
                        </td>
                        <td>{set.repetitions}</td>
                        <td>{set.weight}</td>
                    </tr>
                {/each}
            </tbody>
        </table>
    {:else}
        <Notification text={$_("notification_no_sets_exist")} />
    {/if}
</div>

{#if showNoteSavedModal}
    <Modal
        cancel={{
            text: $_("ok"),
            click: () => (showNoteSavedModal = false),
        }}
        title={$_("save_note")}
        on:close={() => (showNoteSavedModal = false)}>
        <p>{$_("save_note_success")}</p>
    </Modal>
{/if}

<style>
    thead th {
        position: sticky;
        top: 0;
        /* Prevents the background from hiding the border. */
        background-clip: padding-box;
    }

    tr:hover td {
        cursor: pointer;
        /* Value of `has-background-link-light`. */
        background-color: hsl(219, 70%, 96%);
    }

    .highlight-exercise-set {
        animation: highlight 0.75s 2 ease-out;
    }

    /*
    Have to use `-global` to prevent the keyframes from being removed.
    https://stackoverflow.com/a/74491304
    */
    @keyframes -global-highlight {
        50% {
            background-color: hsl(204, 86%, 53%);
        }
    }
</style>
