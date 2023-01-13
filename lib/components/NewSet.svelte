<script lang="ts">
    import { Link, navigate } from "svelte-routing";
    import Title from "./Title.svelte";

    export let workoutId: string;

    type Exercise = {
        id: number;
        name: string;
    };

    let exercises: Exercise[] = [
        {
            id: 1,
            name: "Dehnen",
        },
        {
            id: 2,
            name: "Deadlift",
        },
        {
            id: 3,
            name: "Beinpresse",
        },
        {
            id: 4,
            name: "Handstand",
        },
        {
            id: 5,
            name: "Squats",
        },
    ];

    let canSave = false;
    let inputExerciseId: number;
    let inputRepetitions: string;
    let inputWeight: string;

    function checkCanSave() {
        canSave =
            inputRepetitions !== "" &&
            inputWeight !== "" &&
            parseInt(inputRepetitions) > 0 &&
            parseInt(inputWeight) >= 0;
    }

    function save() {
        console.warn(`Implement: save set`, inputExerciseId, inputRepetitions, inputWeight);
    }
</script>

<Title text="Neuer Satz" />

<div class="field">
    <label for="exercise" class="label">Übung</label>
    <div class="select is-fullwidth">
        <select id="exercise" bind:value={inputExerciseId}>
            {#each exercises as exercise}
                <option value={exercise.id}>{exercise.name}</option>
            {/each}
        </select>
    </div>
</div>

<div class="field">
    <label for="repetitions" class="label">Anzahl Wiederholungen</label>
    <div class="control">
        <input
            type="number"
            id="repetitions"
            class="input"
            enterkeyhint="next"
            bind:value={inputRepetitions}
            on:keyup={checkCanSave} />
    </div>
</div>

<div class="field">
    <label for="weight" class="label">Gewicht in KG</label>
    <div class="control">
        <input
            type="number"
            id="weight"
            class="input"
            enterkeyhint="done"
            bind:value={inputWeight}
            on:keyup={checkCanSave} />
    </div>
</div>

<div class="field is-grouped is-grouped-right">
    <button
        class="same-width column is-1 is-fullwidth button is-primary is-light"
        disabled={!canSave}
        on:click={save}>Speichern</button>

    <!-- Use `navigate` instead of `Link` because with `Link` the color would stay blue. -->
    <button
        class="same-width column is-1 is-fullwidth ml-2 button is-light"
        on:click={() => navigate("/workouts/{workoutId}")}>Abbrechen</button>
</div>

<style>
    .same-width {
        min-width: 109px;
    }
</style>
