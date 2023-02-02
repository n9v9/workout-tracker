<script lang="ts">
    import { onMount } from "svelte";
    import { navigate } from "svelte-routing";
    import { api } from "../api/service";
    import type { Exercise, ExerciseSet } from "../api/types";
    import { uiDisabled } from "../store";
    import Button from "./Button.svelte";
    import Modal from "./Modal.svelte";
    import Title from "./Title.svelte";

    export let workoutId: number;
    export let setId: number | null = null;

    let exercises: Exercise[] = [];

    let inputExerciseId: number;
    let inputRepetitions: string;
    let inputWeight: string;
    let inputNote = "";

    let inputWeightElement: HTMLInputElement;
    // Used to enter a new exercise name and to update an existing name.
    let inputExerciseNameElement: HTMLInputElement;
    let inputExerciseName = "";

    let canSave = false;
    let showDeleteModal = false;
    let showAddExerciseModal = false;
    let showChangeExerciseModal = false;
    let showDeleteExerciseModal = false;
    let showCannotDeleteExerciseModal = false;
    let exerciseInSetsCount = 0;

    let canSaveOrUpdateExercise = false;
    let exerciseNameExists = false;
    let existingExercises: string[] = [];

    onMount(load);

    function resetVariables() {
        exercises = [];

        inputExerciseId = undefined;
        inputRepetitions = undefined;
        inputWeight = undefined;
        inputNote = "";

        inputExerciseName = "";

        canSave = false;
        showDeleteModal = false;
        showAddExerciseModal = false;
        showDeleteExerciseModal = false;
        showCannotDeleteExerciseModal = false;
        exerciseInSetsCount = 0;

        canSaveOrUpdateExercise = false;
        exerciseNameExists = false;
        existingExercises = [];
    }

    async function load() {
        resetVariables();

        const result = await Promise.all([
            api.getExercises(),
            setId !== null ? api.getSetByIds(setId) : api.getNewSetRecommendation(workoutId),
        ]);
        const set = result[1] as ExerciseSet;

        exercises = result[0] as Exercise[];
        inputExerciseId = set.exerciseId;
        inputRepetitions = set.repetitions.toString();
        inputWeight = set.weight.toString();
        inputNote = set.note || "";

        checkCanSave();
    }

    function checkCanSave() {
        canSave =
            inputRepetitions !== "" &&
            inputWeight !== "" &&
            parseInt(inputRepetitions) > 0 &&
            parseInt(inputWeight) >= 0;
    }

    async function save() {
        await api.createOrUpdateSet(workoutId, setId, {
            exerciseId: inputExerciseId,
            repetitions: parseInt(inputRepetitions),
            weight: parseInt(inputWeight),
            note: inputNote.trim(),
        });

        goBack();
    }

    async function deleteSet() {
        await api.deleteSetById(setId);
        goBack();
    }

    function goBack() {
        navigate(`/workouts/${workoutId}`);
    }

    function selectText(e: FocusEvent) {
        const input = e.target as HTMLInputElement;
        input.select();
    }

    async function exerciseExists(name: string): Promise<boolean> {
        name = name.trim().toLowerCase();

        if ((await api.existsExercise(name)).exists) {
            existingExercises.push(name);
            canSaveOrUpdateExercise = false;
            exerciseNameExists = true;
            return true;
        }

        return false;
    }

    async function createExercise() {
        if (await exerciseExists(inputExerciseName)) {
            return;
        }

        const { id } = await api.createExercise(inputExerciseName);
        showAddExerciseModal = false;

        // Reload this component, then set the exercise ID to the ID
        // of the newly created exercise.
        await load();
        inputExerciseId = id;
    }

    async function updateExercise() {
        if (await exerciseExists(inputExerciseName)) {
            return;
        }

        const { id } = await api.updateExercise(inputExerciseId, inputExerciseName);
        showChangeExerciseModal = false;

        // Reload this component, then set the exercise ID to the ID
        // of the newly created exercise.
        await load();
        inputExerciseId = id;
    }

    async function updateOrCreateExerciseKeyUp(event: KeyboardEvent, action: () => Promise<void>) {
        const lowerName = inputExerciseName.trim().toLowerCase();

        if (existingExercises.includes(lowerName)) {
            exerciseNameExists = true;
            canSaveOrUpdateExercise = false;
            return;
        }

        exerciseNameExists = false;
        canSaveOrUpdateExercise = lowerName !== "";

        if (event.key === "Enter" && canSaveOrUpdateExercise) {
            await action();
        }
    }

    async function openDeleteExerciseModal() {
        const result = await api.getExerciseCountInSets(inputExerciseId);

        if (result.count > 0) {
            showCannotDeleteExerciseModal = true;
            exerciseInSetsCount = result.count;
            return;
        }

        showDeleteExerciseModal = true;
    }

    async function deleteExercise() {
        await api.deleteExercise(inputExerciseId);
        await load();
    }
</script>

<Title text={setId === null ? "Neuer Satz" : "Satz Bearbeiten"} />

<div class="field">
    <label for="exercise" class="label">Übung</label>

    <div class="field is-horizontal">
        <div class="field-body">
            <div class="field is-expanded">
                <div class="field has-addons">
                    <div class="control is-expanded">
                        <div class="select is-fullwidth">
                            <select
                                id="exercise"
                                bind:value={inputExerciseId}
                                disabled={$uiDisabled}>
                                {#each exercises as exercise}
                                    <option value={exercise.id}>{exercise.name}</option>
                                {/each}
                            </select>
                        </div>
                    </div>
                    <p class="control">
                        <Button
                            classes="button"
                            click={() => {
                                showAddExerciseModal = true;
                                // XXX: Without `setTimeout`, the element would still be undefined
                                //      because it is only rendered when `showAddExerciseModal` is
                                //      true. So we just queue it here to make it work.
                                setTimeout(() => inputExerciseNameElement.focus(), 0);
                            }}>
                            <span class="icon">
                                <i class="bi bi-plus-lg" />
                            </span>
                        </Button>
                    </p>
                    <p class="control">
                        <Button
                            classes="button"
                            click={() => {
                                showChangeExerciseModal = true;
                                setTimeout(() => inputExerciseNameElement.focus(), 0);
                            }}>
                            <span class="icon">
                                <i class="bi bi-pencil" />
                            </span>
                        </Button>
                    </p>
                    <p class="control">
                        <Button classes="button" click={openDeleteExerciseModal}>
                            <span class="icon has-text-danger">
                                <i class="bi bi-trash3" />
                            </span>
                        </Button>
                    </p>
                </div>
            </div>
        </div>
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
            on:focus={selectText}
            on:keyup={e => {
                if (e.key == "Enter") {
                    inputWeightElement.focus();
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
            enterkeyhint="send"
            bind:value={inputWeight}
            bind:this={inputWeightElement}
            on:focus={selectText}
            on:keyup={e => {
                if (e.key === "Enter") {
                    inputWeightElement.blur();
                    if (canSave) {
                        save();
                    }
                }
                checkCanSave();
            }}
            disabled={$uiDisabled} />
    </div>
</div>

<div class="field">
    <label for="note" class="label">Notiz</label>
    <div class="control">
        <span
            id="note"
            class="textarea"
            contenteditable="true"
            role="textbox"
            bind:innerHTML={inputNote}>{inputNote}</span>
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
        confirm={{
            text: "Löschen",
            click: deleteSet,
            canClick: true,
        }}
        cancel={{
            text: "Abbrechen",
            click: () => (showDeleteModal = false),
        }}>
        Satz wirklich löschen?
    </Modal>
{:else if showAddExerciseModal}
    <Modal
        title="Übung erstellen"
        confirm={{
            text: "Speichern",
            click: createExercise,
            canClick: canSaveOrUpdateExercise,
        }}
        cancel={{
            text: "Abbrechen",
            click: () => {
                showAddExerciseModal = false;
                inputExerciseName = "";
                exerciseNameExists = false;
            },
        }}>
        <div class="field">
            <label for="new-exercise-name" class="label">Name der Übung</label>
            <div class="field">
                <div class="control">
                    <input
                        id="new-exercise-name"
                        class="input"
                        type="text"
                        bind:this={inputExerciseNameElement}
                        bind:value={inputExerciseName}
                        on:keyup={e => updateOrCreateExerciseKeyUp(e, createExercise)}
                        placeholder="z. B. Squats"
                        enterkeyhint="send" />
                </div>
                <p class="{!exerciseNameExists ? 'is-hidden' : ''} help is-danger"
                    >Diese Übung existiert bereits.</p>
            </div>
        </div>
    </Modal>
{:else if showChangeExerciseModal}
    <Modal
        title="Übung bearbeiten"
        confirm={{ text: "Speichern", click: updateExercise, canClick: canSaveOrUpdateExercise }}
        cancel={{
            text: "Abbrechen",
            click: () => {
                showChangeExerciseModal = false;
                inputExerciseName = "";
                exerciseNameExists = false;
            },
        }}>
        <div class="field">
            <label for="changed-exercise-name" class="label">Neuer Name</label>
            <div class="field">
                <div class="control">
                    <input
                        id="changed-exercise-name"
                        class="input"
                        type="text"
                        bind:this={inputExerciseNameElement}
                        bind:value={inputExerciseName}
                        on:keyup={e => updateOrCreateExerciseKeyUp(e, updateExercise)}
                        placeholder="z. B. Squats"
                        enterkeyhint="send" />
                </div>
                <p class="{!exerciseNameExists ? 'is-hidden' : ''} help is-danger"
                    >Diese Übung existiert bereits.</p>
            </div>
        </div>
    </Modal>
{:else if showDeleteExerciseModal}
    <Modal
        title="Übung Löschen"
        confirm={{
            text: "Löschen",
            click: deleteExercise,
            canClick: true,
        }}
        cancel={{
            text: "Abbrechen",
            click: () => (showDeleteExerciseModal = false),
        }}>
        <p
            >Soll die Übung "{exercises.find(x => x.id === inputExerciseId).name}" wirklich gelöscht
            werden?</p>
    </Modal>
{:else if showCannotDeleteExerciseModal}
    <Modal
        title="Übung Löschen"
        cancel={{
            text: "OK",
            click: () => (showCannotDeleteExerciseModal = false),
        }}>
        <p
            >Die Übung "{exercises.find(x => x.id === inputExerciseId).name}" kann nicht gelöscht
            werden, da sie in {exerciseInSetsCount} Sätzen enthalten ist.</p>
    </Modal>
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

    #note {
        display: block;
        padding: calc(0.75em - 1px);
        min-height: 0;
        height: auto;
        line-height: 1.5;
    }

    #note[contenteditable]:empty::before {
        content: "Optionale Notiz ...";
        color: gray;
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
