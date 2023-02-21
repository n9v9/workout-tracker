<script lang="ts">
    import { onMount } from "svelte";
    import { navigate } from "svelte-routing";
    import { api } from "../api/service";
    import type { Exercise, ExerciseSet } from "../api/types";
    import { formatDate } from "../date";
    import { uiDisabled } from "../store";
    import Button from "./Button.svelte";
    import Notification from "./Notification.svelte";
    import Title from "./Title.svelte";
    import UpDownArrow from "./UpDownArrow.svelte";

    type DisplayExerciseSet = ExerciseSet & { isPersonalBest: boolean };

    let exercises: Exercise[] = [];
    let selectedExercisePlaceholder: Exercise = null;
    let selectedExercise: Exercise = null;
    let exerciseSets: DisplayExerciseSet[] = [];

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
    });

    function loadExerciseSets() {
        api.getSetsByExerciseId(selectedExercise.id).then(sets => {
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
        });
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
</script>

<Title text="Satz suchen" />

<div class="block">
    <Button classes="button is-fullwidth mt-2 is-link" click={() => navigate("/")}>
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>Zur Workout Übersicht</span>
    </Button>
</div>

<div class="field">
    <label for="exercise" class="label">Übung</label>

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
                                >Übung auswählen</option>
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
                    Datum
                    {#if sortState.active === "date"}
                        <UpDownArrow up={sortState.ascending} />
                    {/if}
                </th>
                <th
                    class="has-background-white"
                    on:click={() => sortExerciseSets("repetitions", true)}>
                    <abbr title="Anzahl Wiederholungen">Reps</abbr>
                    {#if sortState.active === "repetitions"}
                        <UpDownArrow up={sortState.ascending} />
                    {/if}
                </th>
                <th class="has-background-white" on:click={() => sortExerciseSets("weight", true)}>
                    <abbr title="Gewicht in KG">KG</abbr>
                    {#if sortState.active === "weight"}
                        <UpDownArrow up={sortState.ascending} />
                    {/if}
                </th>
            </tr>
        </thead>
        <tbody>
            {#each exerciseSets as set}
                <tr
                    class={set.isPersonalBest ? "personal-best" : ""}
                    on:click={() => navigate(`/workouts/${set.workoutId}`)}>
                    <td>{formatDate(set.date)}</td>
                    <td>{set.repetitions}</td>
                    <td>{set.weight}</td>
                </tr>
            {/each}
        </tbody>
    </table>
{:else if selectedExercise === null}
    <Notification text="Bitte eine Übung auswählen." />
{:else}
    <Notification text="Es existieren noch keine Sätze mit dieser Übung." />
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
    .personal-best {
        background-color: #90f7b3 !important;
    }
    .personal-best:hover {
        background-color: #90f7b3 !important;
    }
</style>
