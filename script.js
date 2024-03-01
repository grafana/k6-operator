import ws from 'k6/ws';
import { URL } from 'https://jslib.k6.io/url/1.0.0/index.js';
import { check } from "k6";
import { Counter } from 'k6/metrics';

const connections = new Counter('websocket_connections');

export const options = {
    stages: [
        { duration: '30s', target: 1000 },
        { duration: '2m30s', target: 10000 },
        { duration: '60m0s', target: 100000 },
      ],
//   vus: 100000,
//   duration: '3600s',
};

export default function () {
  const urlone = 'ws://api.dev.sariska.io/api/v1/messaging/websocket?token=eyJhbGciOiJSUzI1NiIsImtpZCI6IjM1YzVjMzMwYzgzMDlmNWE1MDNkMGE1Yzc0YmZmOGRhNzI2OGEzYWRiNTM0Y2I5YTYyYjljYzZiYmZjZGUwYTMiLCJ0eXAiOiJKV1QifQ.eyJjb250ZXh0Ijp7InVzZXIiOnsiaWQiOiJ4eGNxdWVyOCIsIm5hbWUiOiJnYXVyYXZqaSJ9LCJncm91cCI6IjkifSwic3ViIjoiYXZvbjVqY3RucG4yZDk4cDRkZW10ZiIsInJvb20iOiIqIiwiaWF0IjoxNzA5MjcxMjU5LCJuYmYiOjE3MDkyNzEyNTksImlzcyI6InNhcmlza2EiLCJhdWQiOiJtZWRpYV9tZXNzYWdpbmdfY28tYnJvd3NpbmciLCJleHAiOjE3MDkzNTc2NTl9.iQFxkUM8D_FqyLn3hqmVtW0_2608mVGt_4OmIo0bGNrq8xz88OxIhjxxm8ZOm6qXY8KmUjwzQ82HFX3Fj3jbQPErUN0R3HnGipvH7_qJwYEvYKkRVPy4AKDHN_t7mtlLsAui1tJ9XR1dLqhko5gzDA5oSD1EaEw12LOzv2S21ZjzgSjJoF-Is4OUtQGzQg41RIG5XqAysZEBlt-nqiRyiTTpamWJKnzO5kMGzEsgyvcc5oEDAaCF79MAdEJx4qei9FQAni3EVbcKbo841D0bR8pGRT8_Vtr0lFJ2oJvCsbM2CBwLds3hOeiTn92zSuXMIrZKwOYswL7li71BvwS0TA';
  const params = {};

  const broadcastCallback = (message) => {
    console.log('Received broadcast message:', message);
  };

  const channelName = `rtc:iamgoi`;
  const channel1 = createChannel(urlone, channelName, params, broadcastCallback);

  const content1 = 'Hi Gaurav, one lakh msg';
  const createdBy1 = 'gauravji';

  const res1 = joinChannel(channel1, createdBy1, content1);

  if (res1 && res1.status === 101) {
    connections.add(1);
  }

  check(res1, { "status is 101": (r) => r && r.status === 101 });

}

function createChannel(url, topic, params, broadcastCallback) {
  const channel = new Channel(url, topic, params, broadcastCallback);
  return channel;
}

function joinChannel(channel, createdBy, content) {
  return channel.join({}, () => {
    console.log(`Joined channel ${channel.topic}`);
    const event = "new_message";
    const payload = {
      created_by_name: createdBy,
      x: 'uu',
      y: {
        x: 'ghhg',
      },
      content: content,
    };

    channel.send(event, payload, (response) => {
      console.log('Message sent. Received response:', response);
      channel.leave();
    });
  });
}

class Channel {
  constructor(url, topic, params, broadcastCallback) {
    this.url = new URL(url);
    this.topic = topic;
    this.params = params;
    this.broadcastCallback = broadcastCallback;
    this.callbacks = {};
    this.sent_messages = {};
    this.messageRef = 5;
    this.joinRef = 5;
    this.url.searchParams.append('vsn', '2.0.0');
  }

  join(payload, callback) {
    return ws.connect(
      this.url.toString(),
      this.params,
      function (socket) {
        this.socket = socket;
        socket.on("open", () => this._send("phx_join", payload, callback));
        socket.on("message", (response) => {
          const message = this._parseMessage(response);
          if (message.ref != null) {
            this.callbacks[message.ref.toString()](message);
          } else {
            this.broadcastCallback(message);
          }
        });

        socket.on('ping', () => console.log('PING!'));
        socket.on('pong', () => console.log('PONG!'));
        socket.on('close', () => console.log('disconnected'));

      }.bind(this)
    );
  }

  leave() {
    this.socket.close();
  }

  setInterval(callback, interval) {
    return this.socket.setInterval(callback, interval);
  }

  setTimeout(callback, period) {
    return this.socket.setTimeout(callback, period);
  }

  send(event, payload, callback = () => {}) {
    this._send(event, payload, callback);
  }

  _send(event, payload, callback) {
    let message = JSON.stringify([
      this.joinRef.toString(),
      this.messageRef.toString(),
      this.topic,
      event,
      payload,
    ]);

    this.socket.send(message);
    this.callbacks[this.messageRef.toString()] = callback;
    this.messageRef += 1;
  }

  _parseMessage(message) {
    let [joinRef, msgRef, topic, event, payload] = JSON.parse(message);
    return {
      joinRef: joinRef,
      ref: msgRef,
      topic: topic,
      event: event,
      payload: payload,
    };
  }
}
