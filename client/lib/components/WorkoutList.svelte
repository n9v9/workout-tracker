<script lang="ts">
    import Title from "./Title.svelte";
    import { navigate } from "svelte-routing";
    import Notification from "./Notification.svelte";
    import Modal from "./Modal.svelte";
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
        await api.deleteWorkout(selectedWorkout.id);
        showDeleteModal = false;
        await loadWorkoutList();
        showDeleteModal = false;
    }

    async function createWorkout() {
        var id = await api.createWorkout();
        navigate(`/workouts/${id}`);
    }

    async function loadWorkoutList() {
        workouts = await api.getWorkoutList();
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
            <i class="bi bi-plus-lg" />
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
        confirm={{
            text: "Löschen",
            click: deleteWorkout,
            canClick: true,
        }}
        cancel={{
            text: "Abbrechen",
            click: () => (showDeleteModal = false),
        }}>
        {`Workout vom ${formatDate(selectedWorkout.started)} wirklich löschen?`}
    </Modal>
{/if}

<style>
    .workout:not(:last-child) {
        margin-bottom: 0.125rem;
    }
</style>
