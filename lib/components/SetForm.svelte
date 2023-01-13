<script lang="ts">
    import { navigate } from "svelte-routing";
    import Modal from "./Modal.svelte";
    import Title from "./Title.svelte";

    export let workoutId: string;
    export let setId: string | null = null;

    if (setId !== null) {
        console.warn(`read and then set data for existing set with id ${setId}`);
    }

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
    let showDeleteModal = false;

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

    function deleteSet() {
        console.warn(`Implement: delete set with id`, setId);
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

<div class="btn-group">
    {#if setId}
        <button class="button is-danger is-light" on:click={() => (showDeleteModal = true)}
            >Löschen</button>
    {/if}

    <button class="button is-light" on:click={() => navigate(`/workouts/${workoutId}`)}
        >Abbrechen</button>

    <button class="button is-primary is-light" disabled={!canSave} on:click={save}
        >Speichern</button>
</div>

{#if showDeleteModal}
    <Modal
        title="Satz Löschen"
        text="Satz wirklich löschen?"
        confirm={deleteSet}
        cancel={() => (showDeleteModal = false)} />
{/if}

<style>
    .btn-group {
        display: flex;
        flex-direction: row;
        justify-content: flex-end;
    }

    .btn-group button {
        min-width: 109px;
    }

    .btn-group button:not(:last-child) {
        margin-right: 0.75rem;
    }

    @media only screen and (max-width: 768px) {
        .btn-group {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            column-gap: 0.75rem;
        }

        .btn-group button:not(:last-child) {
            margin-right: 0;
        }
    }
</style>
