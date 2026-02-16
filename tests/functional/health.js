import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    checks: ["rate==1.0"],
  },
};

const baseUrl = __ENV.BASE_URL || "http://mentor-api.mathtrail.svc.cluster.local:8080";

function testEndpoint(path, expectedStatus) {
  const res = http.get(`${baseUrl}${path}`);
  const ok = check(res, {
    "status is 200": (r) => r.status === 200,
    "status field matches": (r) => {
      try {
        const body = r.json();
        return body && body.status === expectedStatus;
      } catch (err) {
        return false;
      }
    },
  });

  if (!ok) {
    console.error(`[${path}] Response body: ${res.body}`);
  }

  return ok;
}

export default function () {
  testEndpoint("/health/startup", "started");
  testEndpoint("/health/liveness", "ok");
  testEndpoint("/health/ready", "ready");
}
