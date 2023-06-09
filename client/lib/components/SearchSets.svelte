<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "../api/service";
    import type { Exercise, ExerciseSet } from "../api/types";
    import { formatDate } from "../date";
    import { preselectExerciseSet, scrollToSetId, settings, uiDisabled } from "../store";
    import Button from "./Button.svelte";
    import Notification from "./Notification.svelte";
    import Title from "./Title.svelte";
    import UpDownArrow from "./UpDownArrow.svelte";
    import { _ } from "svelte-i18n";
    import { push } from "svelte-spa-router";

    type DisplayExerciseSet = ExerciseSet & { isPersonalBest: boolean };

    let exercises: Exercise[] = [];
    let selectedExercisePlaceholder: Exercise = null;
    let selectedExercise: Exercise = null;
    let exerciseSets: DisplayExerciseSet[] = [];
    let highlightSetId: number | null = null;

    type SortRow = "date" | "repetitions" | "weight";

    type SortState = {
        ascending: boolean;
        active: SortRow;
    };

    let sortState: SortState = {
        ascending: false,
        active: "date",
    };

    onMount(async () => {
        exercises = await api.getExercises();

        if ($preselectExerciseSet !== null) {
            selectedExercise = exercises.find(x => x.id === $preselectExerciseSet.exerciseId);
            await loadExerciseSets();

            if ($preselectExerciseSet.setId !== null) {
                highlightSetId = $preselectExerciseSet.setId;
                setTimeout(() => {
                    const element = document.querySelector(`#set-${highlightSetId}`);
                    element.scrollIntoView({
                        behavior: "auto",
                        block: "center",
                        inline: "center",
                    });
                    setTimeout(() => {
                        highlightSetId = null;
                    }, 1500); // Must be in sync with the duration in the keyframe.
                }, 0);
            }

            $preselectExerciseSet = null;
        }
    });

    async function loadExerciseSets() {
        const sets = await api.getSetsByExerciseId(selectedExercise.id);
        const displaySets = sets as DisplayExerciseSet[];

        const maxWeight = displaySets.reduce(
            (max, current) => (current.weight > max ? current.weight : max),
            0,
        );

        const maxRepsWithMaxWeight = displaySets.reduce(
            (max, current) =>
                current.repetitions > max && current.weight === maxWeight
                    ? current.repetitions
                    : max,
            0,
        );

        // Filter out body weight exercises. Otherwise all sets would be marked as personal best.
        if (maxWeight > 0) {
            displaySets.forEach(
                x =>
                    (x.isPersonalBest =
                        x.weight === maxWeight && x.repetitions === maxRepsWithMaxWeight),
            );
        }

        exerciseSets = displaySets;
        sortExerciseSets(sortState.active, false);
    }

    function sortExerciseSets(row: SortRow, toggle: boolean) {
        if (toggle) {
            sortState.ascending = !sortState.ascending;
        }

        // When sorting by a different column, start with ascending order.
        if (row !== sortState.active) {
            sortState.ascending = true;
        }

        switch (row) {
            case "date":
                exerciseSets.sort((a, b) =>
                    sortState.ascending
                        ? a.date.getTime() - b.date.getTime()
                        : b.date.getTime() - a.date.getTime(),
                );
                sortState.active = "date";
                break;
            case "repetitions":
                exerciseSets.sort((a, b) =>
                    sortState.ascending
                        ? a.repetitions - b.repetitions
                        : b.repetitions - a.repetitions,
                );
                sortState.active = "repetitions";
                break;
            case "weight":
                exerciseSets.sort((a, b) =>
                    sortState.ascending ? a.weight - b.weight : b.weight - a.weight,
                );
                sortState.active = "weight";
                break;
        }

        // Reassign to trigger re-render.
        exerciseSets = exerciseSets;
    }

    function navigateToWorkout(setId: number, workoutId: number) {
        $scrollToSetId = setId;
        push(`/workouts/${workoutId}`);
    }
</script>

<Title text={$_("search_sets")} />

<div class="block">
    <Button classes="button is-fullwidth mt-2 is-link" click={() => push("/")}>
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>{$_("back_to_workout_list")}</span>
    </Button>
</div>

<div class="field">
    <label for="exercise" class="label">{$_("exercise")}</label>

    <div class="field is-horizontal">
        <div class="field-body">
            <div class="field is-expanded">
                <div class="control is-expanded">
                    <div class="select is-fullwidth">
                        <select
                            id="exercise"
                            bind:value={selectedExercise}
                            disabled={$uiDisabled}
                            on:change={loadExerciseSets}>
                            <option value={selectedExercisePlaceholder} disabled selected
                                >{$_("select_exercise")}</option>
                            {#each exercises as exercise}
                                <option value={exercise}>{exercise.name}</option>
                            {/each}
                        </select>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>

{#if exerciseSets.length > 0}
    <table class="table is-fullwidth is-striped is-hoverable is-bordered mb-3">
        <thead>
            <tr>
                <th class="has-background-white" on:click={() => sortExerciseSets("date", true)}>
                    {$_("date")}
                    {#if sortState.active === "date"}
                        <UpDownArrow up={sortState.ascending} />
                    {/if}
                </th>
                <th
                    class="has-background-white"
                    on:click={() => sortExerciseSets("repetitions", true)}>
                    <abbr title={$_("number_repetitions")}>{$_("abbr_number_repetitions")}</abbr>
                    {#if sortState.active === "repetitions"}
                        <UpDownArrow up={sortState.ascending} />
                    {/if}
                </th>
                <th class="has-background-white" on:click={() => sortExerciseSets("weight", true)}>
                    <abbr title={$_(`weight_in_${$settings.unit}`)}
                        >{$_(`abbr_weight_in_${$settings.unit}`)}</abbr>
                    {#if sortState.active === "weight"}
                        <UpDownArrow up={sortState.ascending} />
                    {/if}
                </th>
            </tr>
        </thead>
        <tbody>
            {#each exerciseSets as set}
                <tr
                    id="set-{set.id}"
                    class="{set.isPersonalBest ? 'personal-best' : ''} {set.id === highlightSetId
                        ? 'highlight-exercise-set'
                        : ''}"
                    on:click={() => navigateToWorkout(set.id, set.workoutId)}>
                    <td>{formatDate(set.date)}</td>
                    <td>{set.repetitions}</td>
                    <td>{set.weight}</td>
                </tr>
            {/each}
        </tbody>
    </table>
{:else if selectedExercise === null}
    <Notification text={$_("notification_please_select_an_exercise")} />
{:else}
    <Notification text={$_("notification_no_sets_with_exercise_exist")} />
{/if}

<style>
    thead th {
        position: sticky;
        top: 0;
        /* Prevents the background from hiding the border. */
        background-clip: padding-box;
    }

    th:hover {
        cursor: pointer;
    }

    tr:hover td {
        cursor: pointer;
        /* Value of `has-background-link-light`. */
        background-color: hsl(219, 70%, 96%);
    }

    .table.is-striped tbody tr:not(.is-selected).personal-best {
        background-color: #90f7b3;
    }

    tr.personal-best:hover td {
        background-color: #5ff791;
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
