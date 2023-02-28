<script lang="ts">
    import { onMount, createEventDispatcher } from "svelte";
    import { _ } from "svelte-i18n";

    export const setText = (text: string) => {
        note.innerText = text;
        dispatchText();
    };

    let note: HTMLSpanElement;

    const dispatcher = createEventDispatcher();

    onMount(() => {
        note.setAttribute("data-content", $_("placeholder_note"));
    });

    function handleChange() {
        dispatchText();
    }

    function handleCopy(e: ClipboardEvent) {
        // Prevent copying of the span HTML, as it is `contenteditable`.
        e.clipboardData.setData("text/plain", note.innerText.trim());
        e.preventDefault();
    }

    function dispatchText() {
        dispatcher("change", { text: note.innerText.trim() });
    }
</script>

<span
    id="note"
    bind:this={note}
    on:copy={handleCopy}
    on:input={handleChange}
    class="textarea"
    contenteditable="true"
    role="textbox" />

<style>
    #note {
        display: block;
        padding: calc(0.5em - 1px) calc(0.75em - 1px);
        min-height: 0;
        height: auto;
        line-height: 1.5;
    }

    #note:hover {
        /*
        Keeps the correct cursor even when hovering over the
        `contenteditable` placeholder part.
        */
        cursor: text;
    }

    #note[contenteditable]:empty::before {
        /* Set in TS above to allow for I18N. */
        content: attr(data-content);
        color: gray;
    }
</style>
