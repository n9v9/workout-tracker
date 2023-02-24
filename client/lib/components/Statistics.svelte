<script lang="ts">
    import { onMount } from "svelte";
    import { navigate } from "svelte-routing";
    import { api } from "../api/service";
    import { isLoading } from "../store";
    import Button from "./Button.svelte";
    import LoadingBanner from "./LoadingBanner.svelte";
    import Title from "./Title.svelte";
    import { _ } from "svelte-i18n";

    let showQuickView = false;
    let showDetailView = false;

    let totalWorkouts: number;
    let totalDuration: string;
    let avgDuration: string;

    let totalSets: number;
    let totalReps: number;
    let avgRepsPerSet: number;

    onMount(async () => {
        const statistics = await api.getStatistics();

        totalWorkouts = statistics.totalWorkouts;
        totalDuration = getHourAndMinutes(statistics.totalDurationSeconds);
        avgDuration = getHourAndMinutes(statistics.avgDurationSeconds);

        totalSets = statistics.totalSets;
        totalReps = statistics.totalReps;
        avgRepsPerSet = statistics.avgRepsPerSet;
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
    <Button classes="button is-fullwidth mt-2 is-link" click={() => navigate("/")}>
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>{$_("back_to_workout_list")}</span>
    </Button>
</div>

{#if $isLoading}
    <LoadingBanner />
{:else}
    <div class="mb-2">
        <div class="card">
            <header
                class="card-header"
                on:click={() => (showQuickView = !showQuickView)}
                on:keypress={() => (showQuickView = !showQuickView)}>
                <p class="card-header-title">{$_("quick_overview")}</p>
                <button class="card-header-icon">
                    <span class="icon">
                        <i class="bi bi-chevron-{showQuickView ? 'down' : 'right'}" />
                    </span>
                </button>
            </header>
            {#if showQuickView}
                <div class="card-content">
                    <div class="content">
                        <div class="container">
                            <div class="has-text-centered">
                                <p class="is-size-6 heading">{$_("workouts")}</p>
                                <p class="title">{totalWorkouts}</p>
                            </div>
                            <div class="has-text-centered">
                                <p class="is-size-6 heading">{$_("workout_time")}</p>
                                <p class="title">{totalDuration}</p>
                            </div>
                            <div class="has-text-centered">
                                <p class="is-size-6 heading">{$_("avg_workout_time")}</p>
                                <p class="title">{avgDuration}</p>
                            </div>
                            <div class="has-text-centered">
                                <p class="is-size-6 heading">{$_("sets")}</p>
                                <p class="title">{totalSets}</p>
                            </div>
                            <div class="has-text-centered">
                                <p class="is-size-6 heading">{$_("repetitions")}</p>
                                <p class="title">{totalReps}</p>
                            </div>
                            <div class="has-text-centered">
                                <p class="is-size-6 heading">{$_("avg_repetitions_per_set")}</p>
                                <p class="title">{avgRepsPerSet}</p>
                            </div>
                        </div>
                    </div>
                </div>
            {/if}
        </div>
    </div>

    <div class="card">
        <header
            class="card-header"
            on:click={() => (showDetailView = !showDetailView)}
            on:keypress={() => (showDetailView = !showDetailView)}>
            <p class="card-header-title">🚧 {$_("detail_view")}</p>
            <button class="card-header-icon">
                <span class="icon">
                    <i class="bi bi-chevron-{showDetailView ? 'down' : 'right'}" />
                </span>
            </button>
        </header>
        {#if showDetailView}
            <div class="card-content" />
        {/if}
    </div>
{/if}

<style>
    .container {
        display: grid;
        grid-template-columns: repeat(auto-fit, 350px);
        justify-content: center;
    }

    .container div {
        margin-bottom: 1.25rem;
    }

    .card-header:hover {
        cursor: pointer;
    }
</style>
