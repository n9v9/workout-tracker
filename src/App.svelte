<script lang="ts">
    import WorkoutList from "../lib/components/WorkoutList.svelte";
    import { Router, Route } from "svelte-routing";
    import WorkoutInfo from "../lib/components/WorkoutInfo.svelte";
    import NewSet from "../lib/components/NewSet.svelte";
</script>

<main class="container px-3 pt-3">
    <Router>
        <Route path="/" component={WorkoutList} />
        <Route path="/workouts/:id" let:params>
            <WorkoutInfo id={params.id} />
        </Route>
        <Route path="/workouts/:id/sets/add" let:params>
            <NewSet workoutId={params.id} />
        </Route>
    </Router>
</main>

<style></style>
