import http from "k6/http";
import { check } from "k6";

const baseUrl = __ENV.BASE_URL || "http://mentor-api.mathtrail.svc.cluster.local:8080";

export function testHello() {
  const res = http.get(`${baseUrl}/hello`, {
    tags: { name: "GetHello" },
  });

  check(res, {
    "[hello] status is 200": (r) => r.status === 200,
    "[hello] response structure": (r) => {
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
}
