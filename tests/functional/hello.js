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

export default function () {
  const res = http.get(`${baseUrl}/hello`);
  const ok = check(res, {
    "status is 200": (r) => r.status === 200,
    "message and version match": (r) => {
      try {
        const body = r.json();
        return (
          body &&
          body.message === "Hello from LOCAL development via Telepresence2!" &&
          body.version === "local-dev"
        );
      } catch (err) {
        return false;
      }
    },
  });

  if (!ok) {
    console.error(`Response body: ${res.body}`);
  }
}
