import { testHello } from "./hello.js";
import { testHealth } from "./health.js";

export const options = {
  stages: [
    { duration: "3s", target: 10 },   // Ramp-up: 0 → 10 VUs over 3s
    { duration: "5s", target: 10 },   // Steady state: 10 VUs for 5s
    { duration: "2s", target: 0 },    // Ramp-down: 10 → 0 VUs over 2s
  ],
  thresholds: {
    checks: ["rate>=0.99"],  // At least 99% checks pass
    http_req_duration: ["p(95)<200", "p(99)<500"],
    http_req_failed: ["rate<0.01"],
    // Per-endpoint thresholds (enables k6 to track tagged metrics)
    "http_req_duration{name:GetHello}": ["p(95)<200"],
    "http_req_duration{name:GetHealthStartup}": ["p(95)<200"],
    "http_req_duration{name:GetHealthLiveness}": ["p(95)<200"],
    "http_req_duration{name:GetHealthReady}": ["p(95)<200"],
  },
};

export default function () {
  testHello();
  testHealth();
}

// Simple summary export for grafana/run-k6-action
export function handleSummary(data) {
  return {
    'summary.json': JSON.stringify(data),
  };
}
