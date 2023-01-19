<script>
    import { navigate } from "svelte-routing";
    import Button from "./Button.svelte";
    import Title from "./Title.svelte";
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
            <p class="title">18</p>
        </div>
        <div class="has-text-centered">
            <p class="is-size-6 heading">Gesamt Workout Zeit</p>
            <p class="title">86 h</p>
        </div>
        <div class="has-text-centered">
            <p class="is-size-6 heading">Ø Workout Zeit</p>
            <p class="title">2 h 37 min</p>
        </div>
        <div class="has-text-centered">
            <p class="is-size-6 heading">Ø Workout Uhrzeit</p>
            <p class="title">19:42 Uhr</p>
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
