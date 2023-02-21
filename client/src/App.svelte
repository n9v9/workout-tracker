<script lang="ts">
    import WorkoutList from "../lib/components/WorkoutList.svelte";
    import { Router, Route } from "svelte-routing";
    import WorkoutInfo from "../lib/components/WorkoutInfo.svelte";
    import SetForm from "../lib/components/SetForm.svelte";
    import { isLoading, apiErrorMessage } from "../lib/store";
    import Statistics from "../lib/components/Statistics.svelte";
    import SearchSets from "../lib/components/SearchSets.svelte";
</script>

<div class="app">
    <progress class="progress is-small mb-0 {!$isLoading ? 'is-invisible' : ''}" />

    <main class="container px-3 pt-3">
        {#if $apiErrorMessage !== ""}
            <div class="notification is-danger is-light">
                {$apiErrorMessage}
            </div>
        {/if}

        <Router>
            <Route path="/" component={WorkoutList} />
            <Route path="/workouts/:id" let:params>
                <WorkoutInfo id={parseInt(params.id)} />
            </Route>
            <Route path="/workouts/:id/sets/add" let:params>
                <SetForm workoutId={parseInt(params.id)} />
            </Route>
            <Route path="/workouts/:id/sets/:setId" let:params>
                <SetForm workoutId={parseInt(params.id)} setId={parseInt(params.setId)} />
            </Route>
            <Route path="/statistics">
                <Statistics />
            </Route>
            <Route path="/sets">
                <SearchSets />
            </Route>
        </Router>
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
