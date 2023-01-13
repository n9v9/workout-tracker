<script lang="ts">
    import Title from "./Title.svelte";

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
</script>

<Title text={"Workouts"} />

<div class="block">
    <button class="button is-fullwidth is-primary">Neues Workout</button>
</div>

<div class="block">
    <p class="is-size-5 mb-2">Bisherige Workouts</p>

    {#each workouts as workout}
        <div class="workout buttons has-addons">
            <a href="#" class="button is-expanded is-justify-content-flex-start">{workout.id}</a>
            <button class="button" on:click={() => confirmDeletion(workout)}>
                <span class="icon has-text-danger">
                    <i class="bi bi-trash3" />
                </span>
            </button>
        </div>
    {:else}
        <div class="notification">
            <p class="has-text-centered">Es wurden noch keine Workouts eingetragen.</p>
        </div>
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
