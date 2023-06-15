<script lang="ts">
    import { push } from "svelte-spa-router";
    import Button from "./Button.svelte";
    import Title from "./Title.svelte";
    import { _ } from "svelte-i18n";
    import { onMount } from "svelte";
    import type { Exercise } from "../api/types";
    import { api } from "../api/service";
    import Modal from "./Modal.svelte";

    type ListEntry = {
        exercise: Exercise;
        visible: boolean;
    };

    let entries: ListEntry[] = [];
    let exerciseExists = false;
    let name = "";
    let search = "";
    let visible = 0;

    let changeExerciseNameInput: HTMLInputElement;
    let changedExerciseNamePlaceholder = "";
    let changedExerciseName = "";
    let showChangeExerciseModal = false;
    let exerciseNameExists = false;
    let showDeleteModal = false;
    let showCannotDeleteModal = false;
    let selectedExercise: Exercise;
    let selectedExerciseInSetsCount = 0;

    $: canClickCreate = name.trim().length > 0 && !exerciseExists;
    $: canSaveNewExerciseName = !exerciseNameExists && changedExerciseName.trim().length > 0;

    onMount(async () => {
        await update();
    });

    function handleSearch() {
        const s = search.toLowerCase();
        let newVisible = 0;
        entries.forEach(x => {
            x.visible = x.exercise.name.toLowerCase().includes(s);
            if (x.visible) {
                newVisible++;
            }
        });
        visible = newVisible;
        entries = entries;
    }

    function handleKeyDown(e: KeyboardEvent) {
        if (exerciseExists) {
            exerciseExists = false;
            return;
        }
        if (e.key === "Enter") {
            tryCreateExercise();
        }
    }

    async function update() {
        const exercises = await api.getExercises();
        exercises.sort((a, b) => a.name.localeCompare(b.name));

        visible = 0;
        entries = exercises.map(x => {
            visible++;
            return {
                exercise: x,
                visible: true,
            };
        });

        exerciseExists = false;
        name = "";
        showDeleteModal = false;
        showCannotDeleteModal = false;
        selectedExercise = undefined;
        selectedExerciseInSetsCount = 0;
        search = "";
        changedExerciseNamePlaceholder = "";
        exerciseNameExists = false;
        changedExerciseName = "";
        showChangeExerciseModal = false;
    }

    async function tryCreateExercise() {
        const trimmed = name.trim();

        exerciseExists = await api.existsExercise(trimmed);
        if (exerciseExists) {
            return;
        }

        await api.createExercise(trimmed);
        await update();
    }

    async function tryChangeExercise() {
        const trimmed = changedExerciseName.trim();

        exerciseNameExists = await api.existsExercise(trimmed);
        if (exerciseNameExists) {
            return;
        }

        await api.updateExercise(selectedExercise.id, trimmed);
        await update();
    }

    async function openDeleteExerciseModal(exercise: Exercise) {
        selectedExercise = exercise;

        const result = await api.getExerciseCountInSets(exercise.id);

        if (result.count > 0) {
            selectedExerciseInSetsCount = result.count;
            showCannotDeleteModal = true;
            return;
        }

        showDeleteModal = true;
    }

    async function deleteExercise() {
        await api.deleteExercise(selectedExercise.id);
        await update();
    }
</script>

<Title text={$_("exercises")} />

<div class="block">
    <Button classes="button is-fullwidth mt-2 is-link" click={() => push("/")}>
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>{$_("back_to_workout_list")}</span>
    </Button>
</div>

<div class="block">
    <div class="field">
        <label for="create-edit-exercise-input" class="label">{$_("create")}</label>
        <div class="field has-addons">
            <div class="control is-expanded">
                <input
                    type="text"
                    class="input"
                    id="create-edit-exercise-input"
                    placeholder={$_("exercise_name_placeholder")}
                    on:keydown={handleKeyDown}
                    bind:value={name} />
            </div>
            <div class="control">
                <button
                    class="button is-primary"
                    disabled={!canClickCreate}
                    on:click={tryCreateExercise}>{$_("create")}</button>
            </div>
        </div>
        {#if exerciseExists}
            <p class="help is-danger">{$_("exercise_exists")}</p>
        {/if}
    </div>
</div>

<div class="block">
    <label for="search-exercise-input" class="label">{$_("overview")}</label>
    <div class="field has-addons">
        <p class="control has-icons-left is-expanded">
            <input
                class="input"
                type="text"
                id="search-exercise-input"
                placeholder={$_("search_for_exercise")}
                on:keyup={handleSearch}
                bind:value={search} />
            <span class="icon is-small is-left">
                <i class="bi bi-search" />
            </span>
        </p>
        <p id="visible-count" class="control">
            <span class="button is-static is-fullwidth">{visible}/{entries.length}</span>
        </p>
    </div>

    {#each entries as exercise}
        {#if exercise.visible}
            <div class="exercise buttons has-addons">
                <button
                    class="button exercise-name-button"
                    on:click={() => {
                        console.log("TODO");
                    }}>
                    <span class="exercise-name">{exercise.exercise.name}</span>
                </button>
                <Button
                    classes="button"
                    click={() => {
                        selectedExercise = exercise.exercise;
                        changedExerciseName = exercise.exercise.name;
                        showChangeExerciseModal = true;
                        exerciseNameExists = false;

                        setTimeout(() => {
                            changeExerciseNameInput.focus();
                            changeExerciseNameInput.select();
                        }, 0);
                    }}>
                    <span class="icon">
                        <i class="bi bi-pencil" />
                    </span>
                </Button>
                <Button
                    classes="button delete-exercise"
                    click={() => openDeleteExerciseModal(exercise.exercise)}>
                    <span class="icon has-text-danger">
                        <i class="bi bi-trash3" />
                    </span>
                </Button>
            </div>
        {/if}
    {/each}
</div>

{#if showChangeExerciseModal}
    <Modal
        title={$_("edit_exercise")}
        confirm={{
            text: $_("save"),
            click: tryChangeExercise,
            canClick: canSaveNewExerciseName,
            isDestructive: false,
        }}
        cancel={{
            text: $_("cancel"),
            click: () => (showChangeExerciseModal = false),
        }}>
        <div class="field">
            <label for="changed-exercise-name" class="label">{$_("new_exercise_name")}</label>
            <div class="field">
                <div class="control">
                    <input
                        id="changed-exercise-name"
                        class="input"
                        type="text"
                        bind:this={changeExerciseNameInput}
                        bind:value={changedExerciseName}
                        on:keyup={e => {
                            if (e.key === "Enter") {
                                tryChangeExercise();
                                return;
                            }
                            if (e.key === "Escape") {
                                showChangeExerciseModal = false;
                                return;
                            }

                            if (exerciseNameExists) {
                                exerciseNameExists = false;
                            }
                        }}
                        enterkeyhint="send" />
                </div>
                <p class="{!exerciseNameExists ? 'is-hidden' : ''} help is-danger"
                    >{$_("exercise_exists")}</p>
            </div>
        </div>
    </Modal>
{:else if showDeleteModal}
    <Modal
        title={$_("delete_exercise")}
        confirm={{
            text: $_("delete"),
            click: deleteExercise,
            canClick: true,
            isDestructive: true,
        }}
        cancel={{
            text: $_("cancel"),
            click: () => (showDeleteModal = false),
        }}>
        {$_("delete_exercise_confirmation", {
            values: { name: selectedExercise.name },
        })}
    </Modal>
{:else if showCannotDeleteModal}
    <Modal
        title={$_("delete_exercise")}
        cancel={{
            text: $_("ok"),
            click: () => (showCannotDeleteModal = false),
        }}>
        <p
            >{$_("delete_exercise_confirmation_sets_exist", {
                values: {
                    name: entries.find(x => x.exercise.id === selectedExercise.id).exercise.name,
                    count: selectedExerciseInSetsCount,
                },
            })}
        </p>
    </Modal>
{/if}

<style>
    #visible-count {
        /* Same width as the edit and delete buttons together. */
        width: 78px;
    }

    .exercise {
        width: 100%;
        display: flex;
        white-space: nowrap;
    }

    .exercise:not(:last-child) {
        margin-bottom: 0.125rem;
    }

    .exercise-name-button {
        flex: 1;
        justify-content: start;
        overflow: hidden;
    }

    .exercise-name {
        display: block;
        text-overflow: ellipsis;
        white-space: nowrap;
        overflow: hidden;
    }
</style>
