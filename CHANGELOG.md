# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-02-06

### Added
- Initial release
- CPU metrics collection (per-core and aggregate)
- Memory metrics collection (with swap support)
- Disk I/O metrics collection
- Network metrics collection
- SQLite storage with automatic retention management
- Statistical analysis (P50, P90, P95, P99)
- Workload pattern detection (steady-state, bursty, scheduled)
- Instance sizing recommendations for AWS and OTC
- Systemd service support
- Docker container support

### Security
- Run as non-root user (instvisor)
- Systemd hardening options
- Read-only filesystem mounts in Docker
