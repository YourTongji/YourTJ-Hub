# Product context and engineering evidence in Issue/PR review

## Status

Accepted
Class: process

## Context and Problem Statement

Contributors and agents need a consistent path from an observed problem to a verified change.
Feature forms, PR descriptions and repository skills must agree on both the product rationale and
engineering evidence. Target users and acceptance headings alone do not establish that an implementation
solves the relevant problem; tests alone do not establish that the selected behavior serves those users.

## Decision Drivers

- Preserve the PR's summary, behavior change, verification, docs/contract impact and known gaps.
- Add current-state evidence, relevant external research, affected user perspectives and acceptance results.
- Keep requirements and their evidence traceable without copying long Issue bodies into PRs.
- Keep small fixes and maintenance proportional, and avoid adding a duplicate governance source.

## Considered Options

- Extend the existing Issue/PR process with proportional product context and acceptance evidence.
- Require a separate full product specification and approval cycle for every change.
- Keep product questions solely in Issue forms and review PRs only for engineering quality.

## Decision Outcome

Use the first option. The existing Issue/PR guide owns the process; templates collect its inputs and
skills apply it. All five existing engineering PR sections remain. Product sections add background and
research, affected users/stories, acceptance criteria/results and supplementary material.

Feature and interaction work needs relevant external evidence and an applicability conclusion. An
unambiguous defect or mechanical edit can use a short account with explained exclusions. Meaningful
stories and criteria retain identifiers across Issue and PR; completion claims point to actual evidence.
Review distinguishes verified defects, missing requirements/evidence and optional improvements.

Issue/PR descriptions retain task-specific records. Maintained docs own supported behavior; MADRs own
durable rationale. Existing explicit decisions suffice where they cover the scope. This process does not
grant publication authority or replace the separate human release review.

## Pros and Cons of the Options

- Extending the current process keeps one authority and connects product and engineering evidence. It
  requires reviewers to judge relevance and completeness; format validation cannot automate that judgment.
- A separate mandatory specification makes large changes uniform but duplicates Issue/PR content,
  adds an approval cycle for routine edits and encourages empty boilerplate.
- Engineering-only PR review is shorter but can approve a reliable implementation of an unexamined
  user assumption and cannot show which acceptance outcomes were verified.

## Links

- [Issue/PR process](../development/pull-requests.md)
- [PR template](../../.github/pull_request_template.md)
- [Development skill](../../.agents/skills/yourtj-development/SKILL.md)
- [Review skill](../../.agents/skills/yourtj-code-review/SKILL.md)
- [Atlassian: user stories](https://www.atlassian.com/agile/project-management/user-stories)
- [Google: what to look for in a code review](https://google.github.io/eng-practices/review/reviewer/looking-for.html)
