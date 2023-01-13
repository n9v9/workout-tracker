<script lang="ts">
    import Title from "./Title.svelte";
    import { navigate } from "svelte-routing";
    import Notification from "./Notification.svelte";
    import Modal from "./Modal.svelte";
    import { isLoading, uiDisabled } from "../store";
    import type { Workout } from "../api/types";
    import { api } from "../api/service";
    import { onMount } from "svelte";
    import Button from "./Button.svelte";

    let workouts: Workout[] = [];
    let showDeleteModal = false;
    let selectedWorkout: Workout;

    onMount(loadWorkoutList);

    function confirmDeletion(workout: Workout) {
        selectedWorkout = workout;
        showDeleteModal = true;
    }

    async function deleteWorkout() {
        $uiDisabled = true;
        $isLoading = true;
        try {
            await api.deleteWorkout(selectedWorkout.id);
            showDeleteModal = false;
            await loadWorkoutList();
        } finally {
            $uiDisabled = false;
            $isLoading = false;
            showDeleteModal = false;
        }
    }

    async function createWorkout() {
        $uiDisabled = true;
        $isLoading = true;
        try {
            var workout = await api.createWorkout();
            navigate(`/workouts/${workout.id}`);
        } finally {
            $uiDisabled = false;
            $isLoading = false;
        }
    }

    async function loadWorkoutList() {
        $uiDisabled = true;
        $isLoading = true;
        try {
            workouts = await api.getWorkoutList();
        } finally {
            $uiDisabled = false;
            $isLoading = false;
        }
    }
</script>

<Title text={"Workouts"} />

<div class="block">
    <Button classes="button is-fullwidth is-primary" click={createWorkout}>
        <span class="icon">
            <i class="bi bi-plus" />
        </span>
        <span>Neues Workout</span>
    </Button>
</div>

<div class="block">
    <p class="is-size-5 mb-2">Bisherige Workouts</p>

    {#each workouts as workout}
        <div class="workout buttons has-addons">
            <Button
                classes="button is-expanded is-justify-content-flex-start"
                click={() => navigate(`/workouts/${workout.id}`)}>
                {workout.id}
            </Button>
            <Button classes="button" click={() => confirmDeletion(workout)}>
                <span class="icon has-text-danger">
                    <i class="bi bi-trash3" />
                </span>
            </Button>
        </div>
    {:else}
        <Notification text="Es wurden noch keine Workouts eingetragen." />
    {/each}
</div>

{#if showDeleteModal}
    <Modal
        title="Workout Löschen"
        text={`Workout vom ${selectedWorkout.startDateEpochUtc} wirklich löschen?`}
        confirm={() => deleteWorkout()}
        cancel={() => (showDeleteModal = false)} />
{/if}

<style>
    .workout:not(:last-child) {
        margin-bottom: 0.125rem;
    }
</style>
