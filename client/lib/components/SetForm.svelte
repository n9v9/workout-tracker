<script lang="ts">
    import { onMount } from "svelte";
    import { navigate } from "svelte-routing";
    import { api } from "../api/service";
    import type { Exercise } from "../api/types";
    import { isLoading, uiDisabled } from "../store";
    import Button from "./Button.svelte";
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

    let inputWeight: HTMLInputElement;

    onMount(async () => {
        $uiDisabled = true;
        $isLoading = true;
        try {
            const result = await Promise.all([
                api.getExercises(),
                setId !== null
                    ? api.getSetByIds(workoutId, setId)
                    : api.getNewSetRecommendation(workoutId),
            ]);
            const set = result[1];

            exercises = result[0];
            exerciseId = set.exerciseId;
            repetitions = set.repetitions.toString();
            weight = set.weight.toString();

            checkCanSave();
        } finally {
            $uiDisabled = false;
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
        $uiDisabled = true;
        $isLoading = true;
        try {
            await api.createOrUpdateSet(workoutId, {
                setId: setId,
                exerciseId: exerciseId,
                repetitions: parseInt(repetitions),
                weight: parseInt(weight),
            });
            goBack();
        } finally {
            $uiDisabled = false;
            $isLoading = false;
        }
    }

    async function deleteSet() {
        $uiDisabled = true;
        $isLoading = true;
        try {
            await api.deleteSetById(workoutId, setId);
            goBack();
        } finally {
            $uiDisabled = false;
            $isLoading = false;
        }
    }

    function goBack() {
        navigate(`/workouts/${workoutId}`);
    }
</script>

<Title text={setId === null ? "Neuer Satz" : "Satz Bearbeiten"} />

<div class="field">
    <label for="exercise" class="label">Übung</label>
    <div class="select is-fullwidth">
        <select id="exercise" bind:value={exerciseId} disabled={$uiDisabled}>
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
            on:keyup={e => {
                if (e.key == "Enter") {
                    inputWeight.focus();
                } else {
                    checkCanSave();
                }
            }}
            disabled={$uiDisabled} />
    </div>
</div>

<div class="field">
    <label for="weight" class="label">Gewicht in KG</label>
    <div class="control">
        <input
            type="number"
            id="weight"
            class="input"
            enterkeyhint="go"
            bind:value={weight}
            bind:this={inputWeight}
            on:keyup={e => {
                if (e.key === "Enter") {
                    inputWeight.blur();
                    if (canSave) {
                        save();
                    }
                }
                {
                    checkCanSave();
                }
            }}
            disabled={$uiDisabled} />
    </div>
</div>

<div class="btn-group">
    <!-- This div is always displayed so that the other two divs are aligned to the right. -->
    <div>
        {#if setId}
            <Button
                classes="button is-danger is-light is-fullwidth"
                click={() => (showDeleteModal = true)}>Löschen</Button>
        {/if}
    </div>

    <div>
        <Button classes="button is-light is-fullwidth" click={() => goBack()}>Abbrechen</Button>
    </div>

    <div>
        <Button classes="button is-primary is-light is-fullwidth" click={save} disabled={!canSave}
            >Speichern</Button>
    </div>
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

    .btn-group div {
        min-width: 109px;
    }

    .btn-group div:not(:last-child) {
        margin-right: 0.75rem;
    }

    @media only screen and (max-width: 768px) {
        .btn-group {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            column-gap: 0.75rem;
        }

        .btn-group div:not(:last-child) {
            margin-right: 0;
        }
    }
</style>
