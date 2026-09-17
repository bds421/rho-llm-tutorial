# rho/llm Tutorial Suite

Tutorial suite release: **v0.3.6**.

Welcome to the **rho/llm Tutorial Suite**! This directory contains a comprehensive guide to mastering the `rho/llm` library—a production-grade Go wrapper for Large Language Models (LLMs) featuring built-in streaming, multi-key failover, and agentic workflows.

## Overview

The `rho/llm` library provides a unified interface for interacting with various LLM providers, including:
- **Cloud:** Anthropic (Claude), Google (Gemini), OpenAI, Groq, Mistral, xAI.
- **Local:** Ollama, vLLM, LM Studio.

This suite contains **23 tutorials and test suites**, plus a root demo (24 Go modules), using **rho-llm v0.7.5**.

The library source is at [github.com/bds421/rho-llm](https://github.com/bds421/rho-llm).

## Repository Structure

The tutorials are organized by complexity and feature set:

| # | Topic | Key Concepts |
|---|-------|--------------|
| [01-02](./01_basic) | Core Basics | Complete & Stream APIs, Roles, Tokens. |
| [03-04](./03_tool_use) | Agency & Logic | **Tool Use** (Function Calling) & **Extended Thinking** (Reasoning). |
| [05-07](./05_error_handling) | Production Readiness | Error Classification, Backoff, Cost Estimation, Logging Middleware. |
| [08-10](./08_auth_pool_failover) | Advanced Flows | Multi-key Failover, System Prompts, Multi-turn Chat, Streaming Tools. |
| [11-13](./11_stream_abort_and_errors) | Reliability | Abort Control, Request Overrides, Deep Registry Inspection. |
| [14-15](./14_provider_helpers) | Ecosystem | Provider Presets, No-Auth detection, Multi-provider comparisons, Thinking/Reasoning content, live integration tests. |
| [16-18](./16_pool_deep_dive) | Internals | AuthPool mechanisms, Named Error Constructors, Content Model (Multimodal, Image/Vision), live vision integration tests. |
| [19](./19_stress_tests) | Validation | Concurrent Stress Tests, Race-condition validation, Performance Benchmarks. |
| [20](./20_capability_test) | Capability Testing | Multi-model regression matrix, YAML-driven test cases (L1 factual → L5 epistemic logic/clock trisection), multi-language (EN/DE/ES), `-config` and `-short` flags, report generation. |
| [21](./21_cloud_ctl_tool_use) | Tool Use Benchmark | Live model tool-use loop with mocked tool responses, YAML-driven multi-model test matrix, parallel-by-provider execution, markdown report generation. |
| [22](./22_cloud_ctl_http_tool_use) | HTTP Tool Use Benchmark | Live model tool-use loop against a running cloud-ctl HTTP server. |
| [23](./23_food_nutrition_vision) | Food-photo Nutrition Evaluation | Vision, forced structured tool results, reference error metrics, Nutrition5k and SNAPMe evaluation guidance; offline unit tests. |

## Getting Started

### Prerequisites
- Go 1.26.8+
- API Keys for Gemini, Anthropic, or OpenAI (optional if using [Ollama](https://ollama.com))

### Environment Setup
Create a `.env` file in the repository root (or export the variables):
```bash
GEMINI_API_KEY=your_key_here
ANTHROPIC_API_KEY=your_key_here
```

### Running a Tutorial
Tutorials 01–18 are standalone Go programs. Export the environment variables, change to the tutorial's directory, and run it:
```bash
cd 01_basic
go run main.go
```

### Offline Verification

`make build-all` compiles all 24 modules, including test binaries, without running them. `make vet-all` checks every module. `make test-all` also runs tutorial 19's mock-based stress tests and tutorial 23's offline tests with the race detector; it does not run live provider benchmarks. Tutorial 23's CLI explicitly sends the selected image to a provider when run; its tests do not.

Tutorials 20–22 are live benchmarks. Tutorial 20's `-short` flag only reduces the language matrix; it still calls models. Tutorial 21 mocks tools, not model responses. Tutorial 22 also requires a running cloud-ctl HTTP server. Run these suites explicitly when their services and credentials are configured.

## Documentation & Reports

- **[Historical QA Report](./REPORT.md)**: Includes an earlier API coverage cross-reference, tutorial execution logs, and tracked bug reports; it is not a coverage audit of v0.7.5.
- **[Stress Test Details](./19_stress_tests)**: Deep dive into the 49+ tests that ensure library stability.
- **[Capability Test Reports](./20_capability_test/reports)**: Multi-model regression results across reasoning and formatting tasks (generated locally, not checked in).

## Changelog

See [CHANGELOG.md](./CHANGELOG.md) for version history.

## Stability
Tutorial 19 exercises `AuthPool` and `PooledClient` with mocks and Go's `-race` detector. Offline checks verify compilation and mock behavior; live model behavior requires the opt-in integration tests and benchmarks.
