import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import yaml from "js-yaml";

const paths = new URL("../paths/", import.meta.url);
const documents = new Map();
function operation(file, key, method) {
  if (!documents.has(file)) {
    documents.set(file, yaml.load(readFileSync(new URL(file, paths), "utf8")));
  }
  return documents.get(file)[key][method];
}

// These routes use RateLimit in route4api.go. Read-only passkey listing does not;
// reauthentication does, sharing password.change with the credential mutations.
const rateLimited = [
  ["auth-passkeys.yaml", "loginBegin", "post", "login"],
  ["auth-passkeys.yaml", "loginFinish", "post", "login"],
  ["auth-passkeys.yaml", "reauth", "post", "password.change"],
  ["auth-passkeys.yaml", "registrationBegin", "post", "password.change"],
  ["auth-passkeys.yaml", "registrationFinish", "post", "password.change"],
  ["auth-passkeys.yaml", "rename", "post", "password.change"],
  ["auth-passkeys.yaml", "revoke", "post", "password.change"],
  ["agent-events.yaml", "agentListEvents", "get", "interact"],
  ["agent-events.yaml", "agentGetEvent", "get", "interact"],
  ["agent-events.yaml", "agentAckEvents", "post", "interact"],
  ["agent-events.yaml", "forumMentionTargets", "get", "interact"],
  ["moderation-appeals.yaml", "contentAppeals", "post", "interact"],
  ["moderation-appeals.yaml", "moderationAppealAction", "post", "interact"],
];

for (const [file, key, method, action] of rateLimited) {
  test(`${file} ${key} documents the middleware rate-limit response`, () => {
    const response = operation(file, key, method).responses["429"];
    assert.ok(response, "RateLimit must have a 429 response");
    assert.deepEqual(response.headers["Retry-After"], {
      required: true,
      schema: { type: "integer", minimum: 1 },
    });
    const content = response.content["application/json"];
    assert.equal(content.schema.$ref, "../components/schemas.yaml#/RateLimitedFailure");
    const fixture = content.examples.rateLimited.externalValue;
    assert.match(fixture, /-rate-limited\.json$/);
    const sample = JSON.parse(readFileSync(new URL(fixture, new URL(file, paths)), "utf8"));
    assert.equal(sample.code, 1);
    assert.equal(sample.result, null);
    assert.equal(sample.messageCode, "common.rateLimited");
    assert.equal(sample.params.action, action);
    assert.ok(Number.isInteger(sample.params.retryAfterSeconds));
    assert.ok(sample.params.retryAfterSeconds >= 1);
  });
}

for (const key of ["registrationBegin", "registrationFinish", "rename", "revoke"]) {
  test(`passkey ${key} documents authentication and write-gate failures`, () => {
    const responses = operation("auth-passkeys.yaml", key, "post").responses;
    for (const status of ["401", "403"]) {
      assert.ok(responses[status], `missing ${status}`);
      assert.equal(responses[status].content["application/json"].schema.$ref,
        "../components/schemas.yaml#/ApiFailure");
    }
    const examples = responses["403"].content["application/json"].examples;
    for (const [name, code] of [
      ["frozenAccount", "permission.userFrozen"],
      ["pendingActivation", "permission.emailRequired"],
      ["csrfRejected", "auth.csrf.rejected"],
    ]) {
      const fixture = examples[name].externalValue;
      const sample = JSON.parse(readFileSync(new URL(fixture, new URL("auth-passkeys.yaml", paths)), "utf8"));
      assert.equal(sample.messageCode, code);
    }
  });
}

test("read-only passkey listing does not invent rate-limit or write-gate failures", () => {
  const responses = operation("auth-passkeys.yaml", "credentials", "get").responses;
  assert.ok(responses["401"]);
  assert.equal(responses["403"], undefined);
  assert.equal(responses["429"], undefined);
});
