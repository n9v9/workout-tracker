<script lang="ts">
    export let up: boolean;
</script>

<span class="icon">
    {#if up}
        <i class="bi bi-arrow-up" />
    {:else}
        <i class="bi bi-arrow-down" />
    {/if}
</span>
