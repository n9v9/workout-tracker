<script lang="ts">
    import Title from "./Title.svelte";
    import { push } from "svelte-spa-router";
    import Notification from "./Notification.svelte";
    import Modal from "./Modal.svelte";
    import type { Workout } from "../api/types";
    import { api } from "../api/service";
    import { onMount } from "svelte";
    import Button from "./Button.svelte";
    import { formatDate } from "../date";
    import { _ } from "svelte-i18n";
    import { settings } from "../store";

    let workouts: Workout[] = [];
    let showDeleteModal = false;
    let selectedWorkout: Workout;
    let showSettingsModal = false;
    let showWorkoutCreateModal = false;
    let language = $settings.language;
    let unit = $settings.unit;

    onMount(loadWorkoutList);

    function confirmDeletion(workout: Workout) {
        selectedWorkout = workout;
        showDeleteModal = true;
    }

    async function deleteWorkout() {
        await api.deleteWorkout(selectedWorkout.id);
        showDeleteModal = false;
        await loadWorkoutList();
        showDeleteModal = false;
    }

    function handleCreateWorkoutClick() {
        if (workouts.length >= 0) {
            const latest = workouts[0];

            // If the latest workout was started less than 6 hours ago, ask for confirmation.
            // This prevents accidentally creating a new workouts.
            if (new Date().getTime() - latest.started.getTime() < 1000 * 60 * 60 * 6) {
                showWorkoutCreateModal = true;
                return;
            }
        }

        createWorkout();
    }

    async function createWorkout() {
        var id = await api.createWorkout();
        push(`/workouts/${id}`);
    }

    async function loadWorkoutList() {
        workouts = await api.getWorkoutList();
    }

    function saveSettings() {
        $settings = {
            language: language,
            unit: unit,
        };
        // Force re-rendering of workouts to reload the date format according to the set language.
        workouts = workouts;
        showSettingsModal = false;
    }
</script>

<Title text={$_("workouts")} />

<div class="block">
    <Button classes="button is-fullwidth is-primary" click={handleCreateWorkoutClick}>
        <span class="icon">
            <i class="bi bi-plus-lg" />
        </span>
        <span>{$_("new_workout")}</span>
    </Button>
    <Button classes="button is-fullwidth is-info mt-2" click={() => push("/exercises")}>
        <span class="icon">
            <i class="bi bi-list-ul" />
        </span>
        <span>{$_("exercises")}</span>
    </Button>
    <Button classes="button is-fullwidth is-info mt-2" click={() => push("/sets")}>
        <span class="icon">
            <i class="bi bi-search" />
        </span>
        <span>{$_("search_sets")}</span>
    </Button>
    <Button classes="button is-fullwidth is-info mt-2" click={() => push("/statistics")}>
        <span class="icon">
            <i class="bi bi-graph-up-arrow" />
        </span>
        <span>{$_("statistics")}</span>
    </Button>
    <Button
        classes="button is-fullwidth has-background-grey-dark has-text-white-ter mt-2"
        click={() => {
            showSettingsModal = true;
        }}>
        <span class="icon">
            <i class="bi bi-gear" />
        </span>
        <span>{$_("settings")}</span>
    </Button>
</div>

<div class="block">
    <p class="is-size-5 mb-2">{$_("previous_workouts")} ({workouts.length})</p>

    {#each workouts as workout}
        <div class="workout buttons has-addons">
            <Button
                classes="button is-expanded is-justify-content-flex-start"
                click={() => push(`/workouts/${workout.id}`)}>
                {formatDate(workout.started)}
                {#if workout.note}
                    <i class="ml-1 bi bi-chat-left-text has-text-link" />
                {/if}
            </Button>
            <Button classes="button" click={() => confirmDeletion(workout)}>
                <span class="icon has-text-danger">
                    <i class="bi bi-trash3" />
                </span>
            </Button>
        </div>
    {:else}
        <Notification text={$_("notification_no_workouts_exist")} />
    {/each}
</div>

{#if showDeleteModal}
    <Modal
        title={$_("delete_workout")}
        confirm={{
            text: $_("delete"),
            click: deleteWorkout,
            canClick: true,
            isDestructive: true,
        }}
        cancel={{
            text: $_("cancel"),
            click: () => (showDeleteModal = false),
        }}>
        {$_("delete_workout_confirmation", {
            values: { date: formatDate(selectedWorkout.started) },
        })}
    </Modal>
{:else if showWorkoutCreateModal}
    <Modal
        title={$_("create_workout")}
        confirm={{
            text: $_("create"),
            click: createWorkout,
            canClick: true,
            isDestructive: false,
        }}
        cancel={{
            text: $_("cancel"),
            click: () => (showWorkoutCreateModal = false),
        }}>
        {$_("workout_with_near_date_exists")}
    </Modal>
{:else if showSettingsModal}
    <Modal
        title={$_("settings")}
        confirm={{ text: $_("save"), click: saveSettings, canClick: true, isDestructive: false }}
        cancel={{
            text: $_("cancel"),
            click: () => {
                showSettingsModal = false;
            },
        }}>
        <div class="field">
            <label for="language-control" class="label">{$_("language")}</label>
            <div id="language-control" class="control">
                <label for="language-en" class="radio">
                    <input
                        type="radio"
                        name="language"
                        id="language-en"
                        bind:group={language}
                        value="en" />
                    {$_("language_en")}
                </label>
                <label for="language-de" class="radio">
                    <input
                        type="radio"
                        name="language"
                        id="language-de"
                        bind:group={language}
                        value="de" />
                    {$_("language_de")}
                </label>
            </div>
        </div>
        <div class="field">
            <label for="unit-control" class="label">{$_("unit")}</label>
            <div class="control">
                <label for="unit-kg" class="radio">
                    <input type="radio" name="unit" id="unit-kg" bind:group={unit} value="kg" />
                    {$_("unit_kg")}
                </label>
                <label for="unit-lbs" class="radio">
                    <input type="radio" name="unit" id="unit-lbs" bind:group={unit} value="lbs" />
                    {$_("unit_lbs")}
                </label>
            </div>
        </div>
    </Modal>
{/if}

<style>
    .workout:not(:last-child) {
        margin-bottom: 0.125rem;
    }
</style>
