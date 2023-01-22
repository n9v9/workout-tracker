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
            var id = await api.createWorkout();
            navigate(`/workouts/${id}`);
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

    function formatDate(date: Date): string {
        return (
            date.toLocaleString("de", {
                hour: "2-digit",
                minute: "2-digit",
                day: "2-digit",
                month: "2-digit",
                year: "numeric",
            }) + " Uhr"
        );
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
    <Button classes="button is-fullwidth is-info mt-2" click={() => navigate("/statistiken")}>
        <span class="icon">
            <i class="bi bi-graph-up-arrow" />
        </span>
        <span>Statistiken</span>
    </Button>
</div>

<div class="block">
    <p class="is-size-5 mb-2">Bisherige Workouts ({workouts.length})</p>

    {#each workouts as workout}
        <div class="workout buttons has-addons">
            <Button
                classes="button is-expanded is-justify-content-flex-start"
                click={() => navigate(`/workouts/${workout.id}`)}>
                {formatDate(workout.started)}
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
        confirmText="Löschen"
        confirm={() => deleteWorkout()}
        cancel={() => (showDeleteModal = false)}>
        {`Workout vom ${formatDate(selectedWorkout.started)} wirklich löschen?`}
    </Modal>
{/if}

<style>
    .workout:not(:last-child) {
        margin-bottom: 0.125rem;
    }
</style>
