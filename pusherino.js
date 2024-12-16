import http from 'k6/http';
import { check } from 'k6';

export const options = {
    discardResponseBodies: true,

    scenarios: {
        contacts: {
            executor: 'ramping-arrival-rate',
            startRate: 300,
            timeUnit: '1m',
            preAllocatedVUs: 3000,
            maxVUs: 15000, // Maximum VUs for scaling up as needed

            stages: [
                // It should start 300 iterations per `timeUnit` for the first minute.
                { target: 350, duration: '1m' },

                // It should linearly ramp-up to starting 600 iterations per `timeUnit` over the following two minutes.
                { target: 2500, duration: '4m' },

                // It should continue starting 600 iterations per `timeUnit` for the following four minutes.
                { target: 7300, duration: '5m' },

                // It should linearly ramp-down to starting 60 iterations per `timeUnit` over the last two minute.
                { target: 0, duration: '2m' },
            ],
        },
    },
};


export default function () {
    const result = http.get('https://api.clubplatform.de/api/clubs/2/launch_screen?auth_token=eXupJhEwXygQFs-hKyv7');
    check(result, {
        'http response status code is 200': result.status === 200,
    });
}