<script lang="ts">
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
            <button class="button" on:click={cancel}>Abbrechen</button>
            <button class="button" on:click={confirm}>Löschen</button>
        </footer>
    </div>
</div>

<style>
    button {
        min-width: 113px;
    }
</style>
