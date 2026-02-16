import http from "k6/http";
import { check } from "k6";

const baseUrl = __ENV.BASE_URL || "http://mentor-api.mathtrail.svc.cluster.local:8080";

function testEndpoint(path, expectedStatus, tagName) {
  const res = http.get(`${baseUrl}${path}`, {
    tags: { name: tagName },
  });

  check(res, {
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
}

export function testHealth() {
  testEndpoint("/health/startup", "started", "GetHealthStartup");
  testEndpoint("/health/liveness", "ok", "GetHealthLiveness");
  testEndpoint("/health/ready", "ready", "GetHealthReady");
}
