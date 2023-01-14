import { svelte } from "@sveltejs/vite-plugin-svelte";
import postcss from "rollup-plugin-postcss";

export default {
    plugins: [svelte(), postcss()],
};
