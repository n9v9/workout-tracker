<script lang="ts">
    import { onMount } from "svelte";
    import { navigate } from "svelte-routing";
    import { api } from "../api/service";
    import Title from "../components/Title.svelte";
    import Notification from "./Notification.svelte";
    import type { ExerciseSet } from "../api/types";
    import Button from "./Button.svelte";
    import Timer from "./Timer.svelte";

    export let id: number;

    let sets: ExerciseSet[] = [];
    let latest: ExerciseSet | null = null;
    let firstExerciseOfLatestSet: ExerciseSet | null = null;

    onMount(async () => {
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
    });

    function editSet(set: ExerciseSet) {
        navigate(`/workouts/${id}/sets/${set.id}`);
    }
</script>

<Title text="Workout" />

<div class="block">
    <Button
        classes="button is-fullwidth is-primary"
        click={() => navigate(`/workouts/${id}/sets/add`)}>
        <span class="icon">
            <i class="bi bi-plus-lg" />
        </span>
        <span>Neuer Satz</span>
    </Button>
    <Button classes="button is-fullwidth mt-2 is-link" click={() => navigate("/")}>
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>Zur Workout Übersicht</span>
    </Button>
</div>

{#if latest !== null && firstExerciseOfLatestSet !== null}
    <div class="block level is-mobile">
        <div class="level-item">
            <Timer text="Letzter Satz" reference={latest.date} />
        </div>
        <div class="level-item">
            <Timer text="Aktuelle Übung" reference={firstExerciseOfLatestSet.date} />
        </div>
    </div>
{/if}

<div class="block">
    <p class="is-size-5 mb-2">Durchgeführte Sätze</p>

    {#if sets.length > 0}
        <table class="table is-fullwidth is-striped is-hoverable is-bordered mb-3">
            <thead>
                <tr>
                    <th class="has-background-white">Übung</th>
                    <th class="has-background-white">
                        <abbr title="Anzahl Wiederholungen">Reps</abbr>
                    </th>
                    <th class="has-background-white">
                        <abbr title="Gewicht in KG">KG</abbr>
                    </th>
                </tr>
            </thead>
            <tbody>
                {#each sets as set}
                    <tr on:click={() => editSet(set)}>
                        <td>
                            {set.exerciseName}
                            {#if set.note}
                                <sup>
                                    <i class="bi bi-chat-left-text has-text-link" />
                                </sup>
                            {/if}
                        </td>
                        <td>{set.repetitions}</td>
                        <td>{set.weight}</td>
                    </tr>
                {/each}
            </tbody>
        </table>
    {:else}
        <Notification text="Es wurden noch keine Sätze eingetragen." />
    {/if}
</div>

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
</style>
