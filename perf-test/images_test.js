import http from "k6/http";
import { check, sleep } from "k6";
import { getRandomUrl } from "./helpers";

export const options = {
  scenarios: {
    imageproxy_images: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 20 },
        { duration: "60s", target: 60 },
        { duration: "10s", target: 30 },
      ],
      gracefulRampDown: "5s",
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<200"], // 95% of requests should be below 200ms
    http_req_failed: ["rate<0.01"], // http errors should be less than 1%
  },
};

const params = {
  tags: { name: "imageproxy" },
};

export default function () {
  const url = getRandomUrl();
  const res = http.get(url, {
    ...params,
    headers: {
      //   "if-modified-since": "Tue, 20 Feb 2024 04:48:19 GMT",
    },
  });

  check(res, {
    "is status 200": (r) => {
      if (r.status === 200) return true;
      console.log(r.status, r.url);
      return false;
      //   return r.status === 200;
    },
  });

  sleep(1);
}
