<script lang="ts">
    import { onDestroy, onMount } from "svelte";

    export let text: string;
    export let reference: Date;

    let elapsedSeconds: number = 0;
    let timerId: NodeJS.Timer;
    let minutes: string;
    let seconds: string;

    $: {
        minutes = Math.floor(elapsedSeconds / 60)
            .toString()
            .padStart(2, "0");
        seconds = (elapsedSeconds % 60).toString().padStart(2, "0");
    }

    calculateDifference();

    onMount(() => {
        timerId = setInterval(calculateDifference, 1000);
    });

    onDestroy(() => clearInterval(timerId));

    function calculateDifference() {
        // Instead of just adding 1 to elapsedSecond, every second,
        // we calculate the delta of the reference time and the current time.
        // Need to do it this way, because on e.g. Firefox on Android `setInterval`
        // is only executed when the tab is active.
        elapsedSeconds = Math.floor((Date.now() - reference.getTime()) / 1000);
    }
</script>

<!-- Only show the timer up until one hour. -->
{#if elapsedSeconds < 3600}
    <div class="level">
        <div class="level-item has-text-centered">
            <div>
                <p class="is-size-6 heading">{text}</p>
                <p class="title">{minutes}:{seconds}</p>
            </div>
        </div>
    </div>
{/if}
