<script lang="ts">
    import { isLoading } from "../store";
    import Button from "./Button.svelte";

    export let title: string;
    export let confirm: () => void;
    export let canConfirm = true;
    export let confirmText: string;
    export let cancel: () => void;
</script>

<div class="modal is-active">
    <div class="modal-background" />
    <div class="modal-card">
        <header class="modal-card-head is-flex-wrap-wrap p-0">
            <p id="title" class="modal-card-title">{title}</p>
            <progress id="progress" class="progress mb-0 {!$isLoading ? 'is-invisible' : ''}" />
        </header>
        <section class="modal-card-body">
            <slot />
        </section>
        <footer class="modal-card-foot p-3 is-justify-content-flex-end">
            <div class="same-width mr-2">
                <Button classes="button is-fullwidth" click={confirm} disabled={!canConfirm}
                    >{confirmText}</Button>
            </div>
            <div class="same-width">
                <Button classes="button is-fullwidth" click={cancel}>Abbrechen</Button>
            </div>
        </footer>
    </div>
</div>

<style>
    .same-width {
        min-width: 113px;
    }

    #title {
        padding: 20px 20px 10px 20px;
    }

    #progress {
        border-radius: 0;
        height: 0.25rem;
    }
</style>
