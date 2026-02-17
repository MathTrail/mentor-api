import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp up to 10 users
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '30s', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    errors: ['rate<0.1'],              // Error rate should be below 10%
  },
};

const feedbackMessages = {
  en: {
    hard: [
      'This is too hard',
      'Very difficult problem',
      'I can\'t solve this',
      'Too challenging for me',
      'This is confusing',
    ],
    easy: [
      'This is too easy',
      'Very simple task',
      'Boring, need harder problems',
      'Trivial exercise',
      'Super easy',
    ],
    neutral: [
      'I completed the task',
      'Thank you',
      'Interesting problem',
      'Good exercise',
      'Nice task',
    ],
  },
};

function randomElement(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

function generateFeedback() {
  const language = 'en';
  const sentiments = ['hard', 'easy', 'neutral'];
  const sentiment = randomElement(sentiments);
  const message = randomElement(feedbackMessages[language][sentiment]);

  return {
    student_id: uuidv4(),
    task_id: `task-${Math.floor(Math.random() * 1000)}`,
    message: message,
    language: language,
  };
}

export default function () {
  // Test health endpoint
  const healthRes = http.get(`${BASE_URL}/health/ready`);
  check(healthRes, {
    'health check status is 200': (r) => r.status === 200,
  }) || errorRate.add(1);

  sleep(1);

  // Test feedback submission
  const payload = JSON.stringify(generateFeedback());
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const feedbackRes = http.post(`${BASE_URL}/api/v1/feedback`, payload, params);

  const success = check(feedbackRes, {
    'feedback status is 200': (r) => r.status === 200,
    'response has student_id': (r) => JSON.parse(r.body).student_id !== undefined,
    'response has difficulty_adjustment': (r) => JSON.parse(r.body).difficulty_adjustment !== undefined,
    'response has sentiment': (r) => JSON.parse(r.body).sentiment !== undefined,
  });

  if (!success) {
    errorRate.add(1);
    console.error(`Request failed: ${feedbackRes.status} - ${feedbackRes.body}`);
  }

  sleep(1);
}
