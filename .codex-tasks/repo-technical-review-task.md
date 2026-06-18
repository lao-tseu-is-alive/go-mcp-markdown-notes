---
name: repo-technical-review
description: Analyze a software repository, especially a cloud-native Go repository, and produce a structured technical review report with strengths, weaknesses, risks, quick wins, and scores.
---

# Repository Technical Review Skill

Use this skill when the user asks to analyze, review, audit, benchmark, or evaluate a software repository.

The default focus is cloud-native Go development, including:

- Go code quality
- package/module organization
- API boundaries
- Kubernetes/cloud-native readiness
- observability
- configuration
- error handling
- testing
- documentation
- security posture
- developer experience
- production readiness

## Core behavior

When invoked, analyze the current repository objectively.

Base the analysis on actual repository content.

Mention concrete files, packages, commands, configuration files, or code areas when relevant.

Clearly separate:

- observed facts;
- assumptions;
- recommendations.

Do not invent missing features.

Do not browse the web unless explicitly asked.

Do not make source code changes unless explicitly asked.

Create only one Markdown report file unless the user asks otherwise.

## Output file

Create the report as:

`report_YYYYMMDD_MODEL.md`

Where:

- `YYYYMMDD` is today's date;
- `MODEL` is the model name if known or provided;
- if the model name is unknown, use `unknown-model`.

Examples:

- `report_20260617_sonnet.md`
- `report_20260617_glm52.md`
- `report_20260617_unknown-model.md`

## Report structure

Use this exact structure:

# Repository Technical Review — YYYY-MM-DD

## 1. Executive Summary

Give a short synthesis of the repository in 5 to 10 bullet points.

## 2. Repository Purpose

Explain what the project appears to do, who it is for, and what problem it solves.

## 3. Main Technical Scope

Describe the main technologies, frameworks, patterns, commands, APIs, services, or runtime assumptions used by the project.

## 4. Architecture and Design

Analyze:

- package/module organization;
- separation of concerns;
- API boundaries;
- configuration approach;
- error handling approach;
- logging and observability;
- testability;
- extensibility.

## 5. Code Clarity and Maintainability

Assess:

- naming;
- readability;
- complexity;
- duplication;
- idiomatic Go usage;
- dependency management;
- consistency;
- risky abstractions.

## 6. Documentation Quality

Assess:

- README quality;
- inline comments;
- examples;
- operational instructions;
- onboarding quality.

## 7. Strengths

List the most important positive aspects of the repository.

Be specific.

## 8. Weaknesses and Risks

List the main weaknesses, risks, missing pieces, or unclear design choices.

Classify each one as:

- Critical
- Important
- Moderate
- Minor

## 9. Obvious Incomplete or Broken Areas

Identify anything that appears unfinished, inconsistent, non-functional, misleading, or likely to fail.

## 10. Quick Wins

Give a prioritized list of improvements that could be done quickly.

For each quick win, include:

- expected benefit;
- estimated effort: XS, S, M, L;
- risk level: Low, Medium, High.

## 11. Suggested Next Steps

Propose a pragmatic improvement roadmap:

- next 1 hour;
- next half-day;
- next 1–2 days;
- next 1–2 weeks.

## 12. Evaluation Scores

Give scores from 1 to 10 with a short justification for each:

- Purpose clarity
- Architecture
- Go idiomatic quality
- Code maintainability
- Error handling
- Logging/observability
- Testing
- Documentation
- Security posture
- Production readiness
- Developer experience

## 13. Final Verdict

Give a direct final judgment:

- Is the project useful?
- Is it technically promising?
- What is the most important thing to fix first?
- What should be preserved?

## Quality bar

The report must be:

- concrete;
- synthetic;
- evidence-based;
- useful for a senior engineer;
- free of generic filler.

A good report cites real repository elements.

A bad report gives generic advice without showing it inspected the project.
