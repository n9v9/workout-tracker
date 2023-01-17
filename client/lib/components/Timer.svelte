<script lang="ts">
    import { onDestroy, onMount } from "svelte";

    export let text: string;
    export let reference: Date;

    let elapsedSeconds = Math.floor((Date.now() - reference.getTime()) / 1000);
    let timerId: NodeJS.Timer;
    let minutes: string;
    let seconds: string;

    $: {
        minutes = Math.floor(elapsedSeconds / 60)
            .toString()
            .padStart(2, "0");
        seconds = (elapsedSeconds % 60).toString().padStart(2, "0");
    }

    onMount(() => {
        timerId = setInterval(() => {
            elapsedSeconds += 1;
        }, 1000);
    });

    onDestroy(() => clearInterval(timerId));
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
