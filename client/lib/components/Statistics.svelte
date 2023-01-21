<script lang="ts">
    import { onMount } from "svelte";
    import { navigate } from "svelte-routing";
    import { api } from "../api/service";
    import { isLoading, uiDisabled } from "../store";
    import Button from "./Button.svelte";
    import Title from "./Title.svelte";

    let totalWorkouts: number;
    let totalDuration: string;
    let avgDuration: string;

    onMount(async () => {
        $isLoading = true;
        $uiDisabled = true;
        try {
            const statistics = await api.getStatistics();
            totalWorkouts = statistics.totalWorkouts;
            totalDuration = getHourAndMinutes(statistics.totalDurationSeconds);
            avgDuration = getHourAndMinutes(statistics.avgDurationSeconds);
        } finally {
            $isLoading = false;
            $uiDisabled = false;
        }
    });

    function getHourAndMinutes(seconds: number): string {
        const totalMinutes = Math.floor(seconds / 60);
        const hours = Math.floor(totalMinutes / 60);
        const minutes = totalMinutes % 60;

        return `${hours.toString().padStart(2, "0")}h ${minutes.toString().padStart(2, "0")}min`;
    }
</script>

<Title text="Statistiken" />

<div class="block">
    <Button classes="button is-fullwidth mt-2" click={() => navigate("/")}>
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>Zur Workout Übersicht</span>
    </Button>
</div>

<div class="block">
    <h2 class="title is-4">Zeiten</h2>
</div>

<div class="block">
    <div class="container">
        <div class="has-text-centered">
            <p class="is-size-6 heading">Anzahl Workouts</p>
            <p class="title">{totalWorkouts}</p>
        </div>
        <div class="has-text-centered">
            <p class="is-size-6 heading">Gesamt Workout Zeit</p>
            <p class="title">{totalDuration}</p>
        </div>
        <div class="has-text-centered">
            <p class="is-size-6 heading">Ø Workout Zeit</p>
            <p class="title">{avgDuration}</p>
        </div>
    </div>
</div>

<div class="block">
    <h2 class="title is-4">Übungen</h2>
</div>

<style>
    .container {
        display: grid;
        grid-template-columns: repeat(auto-fit, 250px);
        justify-content: center;
    }

    .container div {
        margin-bottom: 1.25rem;
    }
</style>
