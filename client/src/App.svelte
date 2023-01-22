<script lang="ts">
    import WorkoutList from "../lib/components/WorkoutList.svelte";
    import { Router, Route } from "svelte-routing";
    import WorkoutInfo from "../lib/components/WorkoutInfo.svelte";
    import SetForm from "../lib/components/SetForm.svelte";
    import { isLoading, apiErrorMessage } from "../lib/store";
    import Statistics from "../lib/components/Statistics.svelte";
</script>

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
        <Route path="/statistiken">
            <Statistics />
        </Route>
    </Router>
</main>

<style>
    progress {
        border-radius: 0;
        height: 0.5rem !important;
    }
</style>
