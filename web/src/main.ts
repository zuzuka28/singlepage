import { mount } from 'svelte';
import App from './ui/App.svelte';
import './ui/styles.css';

mount(App, { target: document.getElementById('app')! });
