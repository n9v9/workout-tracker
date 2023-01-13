<script lang="ts">
    import { Link } from "svelte-routing";
    import Title from "../components/Title.svelte";
    import Notification from "./Notification.svelte";

    export let id: string;

    type Set = {
        id: number;
        exercise: string;
        repetitions: number;
        weight: number;
    };

    let sets: Set[] = [
        {
            id: 1,
            exercise: "Dehnen",
            repetitions: 1,
            weight: 0,
        },
        {
            id: 2,
            exercise: "Deadlifts",
            repetitions: 6,
            weight: 90,
        },
        {
            id: 3,
            exercise: "Squats",
            repetitions: 8,
            weight: 100,
        },
        {
            id: 4,
            exercise: "Arme",
            repetitions: 12,
            weight: 54,
        },
        {
            id: 1,
            exercise: "Dehnen",
            repetitions: 1,
            weight: 0,
        },
        {
            id: 2,
            exercise: "Deadlifts",
            repetitions: 6,
            weight: 90,
        },
        {
            id: 3,
            exercise: "Squats",
            repetitions: 8,
            weight: 100,
        },
        {
            id: 4,
            exercise: "Arme",
            repetitions: 12,
            weight: 54,
        },
    ];

    function createSet() {
        console.warn(`Implement: create set`);
    }
</script>

<Title text="Workout" />

<div class="block">
    <Link to="/workouts/{id}/sets/add" class="button is-fullwidth is-primary" on:click={createSet}>
        <span class="icon">
            <i class="bi bi-plus" />
        </span>
        <span>Neuer Satz</span>
    </Link>
    <Link to="/" class="button is-fullwidth mt-2">
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>Zur Workout Übersicht</span>
    </Link>
</div>

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
                    <tr>
                        <td>{set.exercise}</td>
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
