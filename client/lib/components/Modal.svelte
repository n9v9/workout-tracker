<script lang="ts">
    import Button from "./Button.svelte";

    export let title: string;
    export let text: string;
    export let confirm: () => void;
    export let cancel: () => void;
</script>

<div class="modal is-active">
    <div class="modal-background" />
    <div class="modal-card">
        <header class="modal-card-head">
            <p class="modal-card-title">{title}</p>
        </header>
        <section class="modal-card-body">{text}</section>
        <footer class="modal-card-foot p-3 is-justify-content-flex-end">
            <div class="same-width mr-2">
                <Button classes="button is-fullwidth" click={confirm}>Löschen</Button>
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
</style>
