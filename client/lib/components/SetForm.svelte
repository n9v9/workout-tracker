<script lang="ts">
    import { onMount } from "svelte";
    import { push } from "svelte-spa-router";
    import { api } from "../api/service";
    import type { Exercise, ExerciseSet } from "../api/types";
    import { preselectExerciseSet, settings, uiDisabled } from "../store";
    import Button from "./Button.svelte";
    import Modal from "./Modal.svelte";
    import Title from "./Title.svelte";
    import { _ } from "svelte-i18n";
    import MultilineInput from "./MultilineInput.svelte";

    export let params: { id: string; setId: string | undefined };
    let workoutId = parseInt(params.id);
    let setId: number | null = null;
    if (params.setId) {
        setId = parseInt(params.setId);
    }

    let exercises: Exercise[] = [];

    let inputExerciseId: number;
    let inputRepetitions: string;
    let inputWeight: string;
    let inputNote: string;
    let updateNote: (text: string) => void;

    let inputWeightElement: HTMLInputElement;

    let canSave = false;
    let showDeleteModal = false;
    let showAddExerciseModal = false;
    let exerciseInSetsCount = 0;

    onMount(load);

    function resetVariables() {
        exercises = [];

        inputExerciseId = undefined;
        inputRepetitions = undefined;
        inputWeight = undefined;
        inputNote = "";

        canSave = false;
        showDeleteModal = false;
        showAddExerciseModal = false;
        exerciseInSetsCount = 0;
    }

    async function load() {
        resetVariables();

        const result = await Promise.all([
            api.getExercises(),
            setId !== null ? api.getSetByIds(setId) : api.suggestNewSet(workoutId, null),
        ]);
        const set = result[1] as ExerciseSet;

        exercises = result[0] as Exercise[];
        inputExerciseId = set.exerciseId;
        inputRepetitions = set.repetitions.toString();
        inputWeight = set.weight.toString();
        inputNote = set.note || "";
        updateNote(inputNote);

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
            note: inputNote,
        });

        goBack();
    }

    async function deleteSet() {
        await api.deleteSetById(setId);
        goBack();
    }

    function goBack() {
        push(`/workouts/${workoutId}`);
    }

    function selectText(e: FocusEvent) {
        const input = e.target as HTMLInputElement;
        input.select();
    }

    function navigateToHistory() {
        $preselectExerciseSet = {
            exerciseId: inputExerciseId,
            setId: setId,
        };
        push("/sets");
    }

    async function loadNewSuggestion() {
        const { exerciseId, repetitions, weight } = await api.suggestNewSet(
            workoutId,
            inputExerciseId,
        );
        inputExerciseId = exerciseId;
        inputRepetitions = repetitions.toString();
        inputWeight = weight.toString();
    }
</script>

<Title text={setId === null ? $_("new_set") : $_("edit_set")} />

<div class="field">
    <label for="exercise" class="label">{$_("exercise")}</label>

    <div class="field is-horizontal">
        <div class="field-body">
            <div class="field is-expanded">
                <div class="control is-expanded">
                    <div class="select is-fullwidth">
                        <select
                            id="exercise"
                            bind:value={inputExerciseId}
                            on:change={loadNewSuggestion}
                            disabled={$uiDisabled}>
                            {#each exercises as exercise}
                                <option value={exercise.id}>{exercise.name}</option>
                            {/each}
                        </select>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>

<div class="field">
    <Button classes="button is-link is-light is-fullwidth" click={navigateToHistory}>
        <span class="icon-text">
            <span class="icon">
                <i class="bi bi-graph-up" />
            </span>
            <span>{$_("history")}</span>
        </span>
    </Button>
</div>

<div class="field">
    <label for="repetitions" class="label">{$_("number_repetitions")}</label>
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
    <label for="weight" class="label">{$_(`weight_in_${$settings.unit}`)}</label>
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
    <label for="input-note" class="label">Note</label>
    <div class="control" id="input-note">
        <MultilineInput on:change={x => (inputNote = x.detail.text)} bind:setText={updateNote} />
    </div>
</div>

<div class="btn-group">
    <!-- This div is always displayed so that the other two divs are aligned to the right. -->
    <div>
        {#if setId}
            <Button
                classes="button is-danger is-light is-fullwidth"
                click={() => (showDeleteModal = true)}>{$_("delete")}</Button>
        {/if}
    </div>

    <div>
        <Button classes="button is-light is-fullwidth" click={() => goBack()}
            >{$_("cancel")}</Button>
    </div>

    <div>
        <Button classes="button is-primary is-light is-fullwidth" click={save} disabled={!canSave}
            >{$_("save")}</Button>
    </div>
</div>

{#if showDeleteModal}
    <Modal
        title={$_("delete_set")}
        confirm={{
            text: $_("delete"),
            click: deleteSet,
            canClick: true,
            isDestructive: true,
        }}
        cancel={{
            text: $_("cancel"),
            click: () => (showDeleteModal = false),
        }}>
        {$_("delete_set_confirmation")}
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
