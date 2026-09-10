#!/usr/bin/env node
// Regression guard for issue #141: the generated Agent list/search TypeScript
// types must reference the concrete payload schemas instead of empty records.
// Run after `pnpm run generate:ts`.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "..");
const output = join(root, "apps", "gooseforum", "resource", "packages", "client", "src", "gen", "openapi.ts");

const source = readFileSync(output, "utf8");

// Each previously-empty Agent item/pagination must now be a named schema ref.
const requiredRefs = [
  'components["schemas"]["PostPayload"]',
  'components["schemas"]["ReplyTargetPayload"]',
  'components["schemas"]["TopicPayload"]',
  'components["schemas"]["UserSearchPayload"]',
  'components["schemas"]["CategorySearchPayload"]',
  'components["schemas"]["CourseSearchPayload"]',
  'components["schemas"]["PaginationPayload"]',
];

// Exact generated lines that appeared when the Agent list/search schemas were
// empty objects. Any one remaining means the empty-record regression is back.
const forbiddenLines = [
  "posts: Record<string, never>[];",
  "replyTargets: Record<string, never>[];",
  "topics: Record<string, never>[];",
  "users: Record<string, never>[];",
  "categories: Record<string, never>[];",
  "pagination: Record<string, never>;",
];

const missing = requiredRefs.filter((ref) => !source.includes(ref));
if (missing.length > 0) {
  console.error(
    `[check-ts-gen] generated types are missing expected schema refs:\n${missing.map((ref) => `  - ${ref}`).join("\n")}\n` +
      `Run \`pnpm run generate:ts\` after editing packages/api-contract, then commit the regenerated openapi.ts.`
  );
  process.exit(1);
}

const leftover = forbiddenLines.filter((line) => source.includes(line));
if (leftover.length > 0) {
  console.error(
    `[check-ts-gen] Agent list/search types still contain empty records:\n${leftover.map((line) => `  - ${line}`).join("\n")}\n` +
      `Fix packages/api-contract/components/schemas.yaml so the Agent responses reference concrete payload schemas, then regenerate.`
  );
  process.exit(1);
}

console.log("[check-ts-gen] OK: Agent list/search types reference concrete payload schemas.");
