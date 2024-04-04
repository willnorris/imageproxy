import http from 'k6/http';
import {check, sleep} from 'k6';
import {getRandomUrl} from "./helpers";


export const options = {
    scenarios: {
        'imageproxy_images_randomized': {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [{duration: '10s', target: 200}, {duration: '60s', target: 600}, {duration: '10s', target: 300},],
            gracefulRampDown: '5s',
        }
    }, thresholds: {
        http_req_duration: ['p(95)<200'], // 95% of requests should be below 200ms
        http_req_failed: ['rate<0.01'], // http errors should be less than 1%

    },
};

const params = {
    tags: {name: 'imageproxy'},
}

export default function () {
    const url =getRandomUrl(true);
    const res = http.get(url, { ...params });

    check(res, {'is status 200': r => {
            // if (r.status === 200) return true;
            // console.log(r.status, r.url);
            // return false;
            return r.status === 200;
        }});

    sleep(1);
}
