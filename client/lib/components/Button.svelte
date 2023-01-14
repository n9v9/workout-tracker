<script lang="ts">
    import { uiDisabled } from "../store";

    export let classes: string;
    export let click: () => void;
    export let disabled = false;
</script>

<button on:click={click} class={classes} disabled={disabled || $uiDisabled}>
    <slot />
</button>
