<script lang="ts">
    import Title from "./Title.svelte";
    import { navigate } from "svelte-routing";
    import Notification from "./Notification.svelte";
    import Modal from "./Modal.svelte";
    import type { Workout } from "../api/types";
    import { api } from "../api/service";
    import { onMount } from "svelte";
    import Button from "./Button.svelte";
    import { formatDate } from "../date";
    import { _ } from "svelte-i18n";

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
</script>

<Title text={$_("workouts")} />

<div class="block">
    <Button classes="button is-fullwidth is-primary" click={createWorkout}>
        <span class="icon">
            <i class="bi bi-plus-lg" />
        </span>
        <span>{$_("new_workout")}</span>
    </Button>
    <Button classes="button is-fullwidth is-info mt-2" click={() => navigate("/sets")}>
        <span class="icon">
            <i class="bi bi-search" />
        </span>
        <span>{$_("search_sets")}</span>
    </Button>
    <Button classes="button is-fullwidth is-info mt-2" click={() => navigate("/statistics")}>
        <span class="icon">
            <i class="bi bi-graph-up-arrow" />
        </span>
        <span>{$_("statistics")}</span>
    </Button>
</div>

<div class="block">
    <p class="is-size-5 mb-2">{$_("previous_workouts")} ({workouts.length})</p>

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
        }}
        cancel={{
            text: $_("cancel"),
            click: () => (showDeleteModal = false),
        }}>
        {$_("delete_workout_confirmation", {
            values: { date: formatDate(selectedWorkout.started) },
        })}
    </Modal>
{/if}

<style>
    .workout:not(:last-child) {
        margin-bottom: 0.125rem;
    }
</style>
