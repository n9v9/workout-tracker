<script lang="ts">
    import { onMount } from "svelte";
    import { push } from "svelte-spa-router";
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

<Title text={$_("statistics")} />

<div class="block">
    <Button classes="button is-fullwidth mt-2 is-link" click={() => push("/")}>
        <span class="icon">
            <i class="bi bi-box-arrow-in-left" />
        </span>
        <span>{$_("back_to_workout_list")}</span>
    </Button>
</div>

{#if $isLoading}
    <LoadingBanner />
{:else}
    <div class="card mb-2">
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
                    <div class="columns is-mobile">
                        <div class="column">{$_("workouts")}</div>
                        <div class="column">{totalWorkouts}</div>
                    </div>
                    <div class="columns is-mobile">
                        <div class="column">{$_("workout_time")}</div>
                        <div class="column">{totalDuration}</div>
                    </div>
                    <div class="columns is-mobile">
                        <div class="column">{$_("avg_workout_time")}</div>
                        <div class="column">{avgDuration}</div>
                    </div>
                    <div class="columns is-mobile">
                        <div class="column">{$_("sets")}</div>
                        <div class="column">{totalSets}</div>
                    </div>
                    <div class="columns is-mobile">
                        <div class="column">{$_("repetitions")}</div>
                        <div class="column">{totalReps}</div>
                    </div>
                    <div class="columns is-mobile">
                        <div class="column">{$_("avg_repetitions_per_set")}</div>
                        <div class="column">{avgRepsPerSet}</div>
                    </div>
                </div>
            </div>
        {/if}
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
    .card-header:hover {
        cursor: pointer;
    }
</style>
