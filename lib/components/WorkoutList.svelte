<script lang="ts">
    import Title from "./Title.svelte";
    import { Link, navigate } from "svelte-routing";
    import Notification from "./Notification.svelte";
    import Modal from "./Modal.svelte";
    import { isLoading } from "../store";
    import type { Workout } from "../api/types";
    import { api } from "../api/service";
    import { onMount } from "svelte";

    let workouts: Workout[] = [];
    let showDeleteModal = false;
    let selectedWorkout: Workout;

    onMount(loadWorkoutList);

    function confirmDeletion(workout: Workout) {
        selectedWorkout = workout;
        showDeleteModal = true;
    }

    async function deleteWorkout() {
        $isLoading = true;
        try {
            await api.deleteWorkout(selectedWorkout.id);
            showDeleteModal = false;
            await loadWorkoutList();
        } finally {
            $isLoading = false;
            showDeleteModal = false;
        }
    }

    async function createWorkout() {
        $isLoading = true;
        try {
            var workout = await api.createWorkout();
            navigate(`/workouts/${workout.id}`);
        } finally {
            $isLoading = false;
        }
    }

    async function loadWorkoutList() {
        $isLoading = true;
        try {
            workouts = await api.getWorkoutList();
        } finally {
            $isLoading = false;
        }
    }
</script>

<Title text={"Workouts"} />

<div class="block">
    <button class="button is-fullwidth is-primary" on:click={createWorkout}>
        <span class="icon">
            <i class="bi bi-plus" />
        </span>
        <span>Neues Workout</span>
    </button>
</div>

<div class="block">
    <p class="is-size-5 mb-2">Bisherige Workouts</p>

    {#each workouts as workout}
        <div class="workout buttons has-addons">
            <Link
                to="/workouts/{workout.id}"
                class="button is-expanded is-justify-content-flex-start">
                {workout.id}
            </Link>
            <button class="button" on:click={() => confirmDeletion(workout)}>
                <span class="icon has-text-danger">
                    <i class="bi bi-trash3" />
                </span>
            </button>
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
