<script lang="ts">
    import WorkoutList from "../lib/components/WorkoutList.svelte";
    import WorkoutInfo from "../lib/components/WorkoutInfo.svelte";
    import SetForm from "../lib/components/SetForm.svelte";
    import { isLoading, apiErrorMessage } from "../lib/store";
    import Statistics from "../lib/components/Statistics.svelte";
    import SearchSets from "../lib/components/SearchSets.svelte";
    import Router from "svelte-spa-router";
    import ExerciseList from "../lib/components/ExerciseList.svelte";

    let routes = {
        "/": WorkoutList,
        "/exercises": ExerciseList,
        "/workouts/:id": WorkoutInfo,
        "/workouts/:id/sets/add": SetForm,
        "/workouts/:id/sets/:setId?": SetForm,
        "/statistics": Statistics,
        "/sets": SearchSets,
    };
</script>

<div class="app">
    <progress class="progress is-small mb-0 {!$isLoading ? 'is-invisible' : ''}" />

    <main class="container px-3 pt-3">
        {#if $apiErrorMessage !== ""}
            <div class="notification is-danger is-light">
                {$apiErrorMessage}
            </div>
        {/if}

        <Router {routes} />
    </main>
</div>

<style>
    .app {
        width: 100%;
        height: 100vh;
    }
    progress {
        border-radius: 0;
        height: 0.5rem !important;
    }
    main {
        margin-bottom: 1em;
    }
</style>
