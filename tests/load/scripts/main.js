import { testHello } from "./hello.js";
import { testHealth } from "./health.js";

export const options = {
  stages: [
    { duration: "3s", target: 10 },   // Ramp-up: 0 → 10 VUs over 3s
    { duration: "5s", target: 10 },   // Steady state: 10 VUs for 5s
    { duration: "2s", target: 0 },    // Ramp-down: 10 → 0 VUs over 2s
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
