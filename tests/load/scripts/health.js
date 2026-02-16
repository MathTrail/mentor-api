import http from "k6/http";
import { check } from "k6";

const baseUrl = __ENV.BASE_URL || "http://mentor-api.mathtrail.svc.cluster.local:8080";

function testEndpoint(path, expectedStatus) {
  const res = http.get(`${baseUrl}${path}`);
  const ok = check(res, {
    [`[${path}] status is 200`]: (r) => r.status === 200,
    [`[${path}] status field matches`]: (r) => {
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

export function testHealth() {
  testEndpoint("/health/startup", "started");
  testEndpoint("/health/liveness", "ok");
  testEndpoint("/health/ready", "ready");
}
