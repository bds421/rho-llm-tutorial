# rho/llm Tutorial Suite

Welcome to the **rho/llm Tutorial Suite**! This directory contains a comprehensive guide to mastering the `rho/llm` library—a production-grade Go wrapper for Large Language Models (LLMs) featuring built-in streaming, multi-key failover, and agentic workflows.

## Overview

The `rho/llm` library provides a unified interface for interacting with various LLM providers, including:
- **Cloud:** Anthropic (Claude), Google (Gemini), OpenAI, Groq, Mistral, xAI.
- **Local:** Ollama, vLLM, LM Studio.

This suite contains **21 tutorials** ranging from basic text completion to advanced stress-testing of concurrent multi-key auth pools.

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
| [16-18](./16_pool_deep_dive) | Internals | AuthPool mechanisms, Named Error Constructors, Content Model (Multimodal, Image/Vision). |
| [19](./19_stress_tests) | Validation | Concurrent Stress Tests, Race-condition validation, Performance Benchmarks. |
| [20](./20_capability_test) | Capability Testing | Multi-model regression matrix, YAML-driven test cases (L1 factual → L5 epistemic logic/clock trisection), multi-language (EN/DE/ES), `-config` and `-short` flags, report generation. |
| [21](./21_cloud_ctl_tool_use) | Tool Use Benchmark | Agentic tool-use loop with mock responses (no external dependencies), YAML-driven multi-model test matrix, parallel-by-provider execution, markdown report generation. |

## Getting Started

### Prerequisites
- Go 1.26.1+
- API Keys for Gemini, Anthropic, or OpenAI (optional if using [Ollama](https://ollama.com))

### Environment Setup
Create a `.env` file in the `llm` directory (or export the variables):
```bash
GEMINI_API_KEY=your_key_here
ANTHROPIC_API_KEY=your_key_here
```

### Running a Tutorial
Each tutorial is a standalone Go program. Change to the tutorial's directory and run it:
```bash
cd 01_basic
go run main.go
```

## Documentation & Reports

- **[Consolidated QA Report](./REPORT.md)**: Includes the 100% API coverage cross-reference, tutorial execution logs, and tracked bug reports.
- **[Stress Test Details](./19_stress_tests)**: Deep dive into the 49+ tests that ensure library stability.
- **[Capability Test Reports](./20_capability_test/reports)**: Multi-model regression results across reasoning and formatting tasks (generated locally, not checked in).

## Changelog

See [CHANGELOG.md](./CHANGELOG.md) for version history.

## Stability
All components (especially `AuthPool` and `PooledClient`) are verified with Go's `-race` detector and benchmarked for zero-allocation performance in hot paths.
