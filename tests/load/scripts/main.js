// Bundle entry point — re-exports all load test scenarios.
// esbuild bundles this into tests/load/dist/bundle.js for the k6-test-runner.

export { default, options } from './feedback_load.js';
