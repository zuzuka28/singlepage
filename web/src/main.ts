import { mount } from 'svelte';
import App from './app/App.svelte';
import { createApplicationRuntime } from './app/logic/runtime';
import './app/styles.css';

const runtime = await createApplicationRuntime();
await runtime.restoreLocator();
mount(App, { target: document.getElementById('app')!, props: { runtime } });
