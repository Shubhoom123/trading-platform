// k6 WebSocket load test for the gateway's live-fill fan-out.
//
// Opens many concurrent WebSocket connections to /ws and counts frames received,
// demonstrating the gateway holds up under concurrent streaming clients.
//
// Usage:
//   TOKEN=$(curl -s localhost:8080/api/auth/login -d '{"email":"...","password":"..."}' \
//            -H 'content-type: application/json' | jq -r .accessToken)
//   k6 run -e TOKEN=$TOKEN -e GATEWAY=ws://localhost:8090 gateway-go/loadtest/ws_load.js

import ws from 'k6/ws';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const framesReceived = new Counter('ws_frames_received');

export const options = {
  scenarios: {
    concurrent_streams: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 500 },  // ramp to 500 concurrent clients
        { duration: '30s', target: 500 },  // hold
        { duration: '10s', target: 0 },    // ramp down
      ],
    },
  },
};

const GATEWAY = __ENV.GATEWAY || 'ws://localhost:8090';
const TOKEN = __ENV.TOKEN || '';
const SYMBOL = __ENV.SYMBOL || 'AAPL';

export default function () {
  const url = `${GATEWAY}/ws?symbol=${SYMBOL}&token=${TOKEN}`;

  const res = ws.connect(url, {}, function (socket) {
    socket.on('message', function () {
      framesReceived.add(1);
    });
    // Each VU keeps its stream open for a few seconds, then closes cleanly.
    socket.setTimeout(function () {
      socket.close();
    }, 5000);
  });

  check(res, { 'ws handshake status is 101': (r) => r && r.status === 101 });
}
