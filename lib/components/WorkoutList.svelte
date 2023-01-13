<script lang="ts">
    import Title from "./Title.svelte";
    import { Link } from "svelte-routing";
    import Notification from "./Notification.svelte";

    type Workout = {
        id: number;
        startDateEpochUtc: number;
    };

    let workouts: Workout[] = [
        {
            id: 1,
            startDateEpochUtc: 2,
        },
        {
            id: 3,
            startDateEpochUtc: 4,
        },
        {
            id: 5,
            startDateEpochUtc: 6,
        },
    ];

    let showDeleteModal = false;
    let selectedWorkout: Workout;

    function confirmDeletion(workout: Workout) {
        selectedWorkout = workout;
        showDeleteModal = true;
    }

    function deleteWorkout() {
        console.warn(`Implement: delete workout`, selectedWorkout);
    }

    function createWorkout() {
        console.warn(`Implement: create workout`);
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
    <div class="modal is-active">
        <div class="modal-background" />
        <div class="modal-card">
            <header class="modal-card-head">
                <p class="modal-card-title">Workout Löschen</p>
            </header>
            <section class="modal-card-body">
                Workout vom {selectedWorkout.startDateEpochUtc} wirklich löschen?
            </section>
            <footer class="modal-card-foot p-3 is-justify-content-flex-end">
                <button class="button" on:click={deleteWorkout}>Löschen</button>
                <button class="button" on:click={() => (showDeleteModal = false)}>Abbrechen</button>
            </footer>
        </div>
    </div>
{/if}

<style>
    .workout:not(:last-child) {
        margin-bottom: 0.125rem;
    }
</style>
