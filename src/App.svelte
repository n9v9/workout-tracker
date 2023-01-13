<script lang="ts">
    import WorkoutList from "../lib/components/WorkoutList.svelte";
    import { Router, Route } from "svelte-routing";
    import WorkoutInfo from "../lib/components/WorkoutInfo.svelte";
    import SetForm from "../lib/components/SetForm.svelte";
    import { isLoading } from "../lib/store";
</script>

<progress class="progress is-small mb-0 hidden {!$isLoading ? 'is-invisible' : ''}" />

<main class="container px-3 pt-3">
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
    </Router>
</main>

<style>
    progress {
        border-radius: 0;
        height: 0.5rem !important;
    }
</style>
