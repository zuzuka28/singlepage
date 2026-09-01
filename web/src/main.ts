import { mount } from 'svelte';
import App from './ui/App.svelte';
import { createApplicationRuntime } from './runtime';
import './ui/styles.css';

const runtime = await createApplicationRuntime();
await runtime.restoreLocator();
mount(App, { target: document.getElementById('app')!, props: { runtime } });
