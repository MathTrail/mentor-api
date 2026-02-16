import { testHello } from "./hello.js";
import { testHealth } from "./health.js";

export const options = {
  stages: [
    { duration: "30s", target: 50 },  // Ramp-up: 0 → 50 VUs over 30s
    { duration: "2m", target: 100 },  // Ramp-up: 50 → 100 VUs over 2m
    { duration: "3m", target: 100 },  // Steady state: 100 VUs for 3m
    { duration: "30s", target: 0 },   // Ramp-down: 100 → 0 VUs over 30s
  ],
  thresholds: {
    // Check success rate
    checks: ["rate>=0.99"],  // At least 99% checks pass
    // HTTP response time thresholds
    http_req_duration: [
      "p(95)<200",           // p95 response time < 200ms
      "p(99)<500",           // p99 response time < 500ms
    ],
    http_req_failed: ["rate<0.01"],  // Less than 1% requests fail
  },
};

export default function () {
  // Run all test scenarios in each iteration
  testHello();
  testHealth();
}
