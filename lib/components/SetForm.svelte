<script lang="ts">
    import { onMount } from "svelte";
    import { navigate } from "svelte-routing";
    import { api } from "../api/service";
    import type { Exercise } from "../api/types";
    import { isLoading } from "../store";
    import Modal from "./Modal.svelte";
    import Title from "./Title.svelte";

    export let workoutId: number;
    export let setId: number | null = null;

    let exercises: Exercise[] = [];
    let exerciseId: number;
    let repetitions: string;
    let weight: string;
    let canSave = false;
    let showDeleteModal = false;

    if (setId !== null) {
        console.warn(`read and then set data for existing set with id ${setId}`);
    }

    onMount(async () => {
        $isLoading = true;
        try {
            if (setId !== null) {
                const result = await Promise.all([api.getExercises(), api.getSetById(setId)]);
                const set = result[1];

                exercises = result[0];
                exerciseId = set.exerciseId;
                repetitions = set.repetitions.toString();
                weight = set.weight.toString();
            } else {
                exercises = await api.getExercises();
                exerciseId = exercises[0].id;
                repetitions = "0";
                weight = "0";
            }

            checkCanSave();
        } finally {
            $isLoading = false;
        }
    });

    function checkCanSave() {
        canSave =
            repetitions !== "" &&
            weight !== "" &&
            parseInt(repetitions) > 0 &&
            parseInt(weight) >= 0;
    }

    async function save() {
        $isLoading = true;
        try {
            await api.saveSet({
                id: setId,
                exerciseId: exerciseId,
                repetitions: parseInt(repetitions),
                weight: parseInt(weight),
            });
        } finally {
            $isLoading = false;
        }
    }

    function deleteSet() {
        console.warn(`Implement: delete set with id`, setId);
    }
</script>

<Title text={setId === null ? "Neuer Satz" : "Satz Bearbeiten"} />

<div class="field">
    <label for="exercise" class="label">Übung</label>
    <div class="select is-fullwidth">
        <select id="exercise" bind:value={exerciseId}>
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
            bind:value={repetitions}
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
            bind:value={weight}
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
