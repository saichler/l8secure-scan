# Layer 8 Ecosystem — Project Reference for AI Context

This document is a concise reference for AI assistants planning implementation projects on the Layer 8 Ecosystem. It describes what each ecosystem project provides so the AI knows where to look for functionality without exploring every repository.

All projects are siblings under the same parent directory. From any project, `../projectname` resolves to another project.

---

## Core Framework Stack (dependency order, bottom to top)

These are the foundational libraries that every implementation project depends on.

### l8types — Interfaces & Protobuf Types (foundation, zero l8* deps)
Defines all core Go interfaces and protobuf types. Key interfaces: `IResources` (dependency injection), `IVNic` (virtual network interface with unicast/multicast), `IServices` (service registry/router), `IServiceHandler`/`IServiceCallback` (CRUD handlers with Before/After hooks), `ISecurityProvider` (AAA contract), `IRegistry` (dynamic type registry), `IIntrospector`/`IDecorators` (runtime type introspection), `IQuery`/`IElements` (query/data containers), `IWebServer`/`IWebService` (HTTP endpoints), `IDistributedCache`, `ITransactionConfig`. Protobuf types cover API (`L8Query`, `L8MetaData`, `AuthToken`), events/alarms, health, notifications, reflection metadata, service registry, system config, web config, and shared business types (`AuditInfo`, `Money`, `Address`, `ContactInfo`, `DateRange`).

### l8reflect — Type Introspection & Deep Operations
Runtime type introspection into `L8Node` trees, dot-notation property path access for nested struct/map/slice fields, deep cloning with circular reference detection, deep equality comparison, and differential update tracking with change recording.

### l8ql — Query Language Engine
SQL-like query language (L8QL) parser and interpreter. Parses query strings (`SELECT`, `FROM`, `WHERE` with 8 comparators, `SORT-BY`, `LIMIT`, `PAGE`, `MATCH-CASE`, `MAPREDUCE`, `GROUP-BY`, `HAVING`, aggregate functions) into `L8Query` protobuf messages, then executes them against live Go objects using reflection.

### l8srlz — Binary Serialization
High-performance binary serialization/deserialization for Go objects (primitives, slices, maps, structs, protobuf). Provides `Elements` — the standard transport envelope implementing `IElements` for batching objects with query, metadata, notification, and replica support.

### l8utils — Shared Utilities
Thread-safe in-memory cache with CRUD/TTL/WebSocket tracking, async queue-backed logger, centralized `IResources` factory, dynamic type registry, blocking/priority queues, thread-safe maps, bounded worker pools with fan-out/fan-in, distributed state change notifications, RESTful web service helpers, async request/response coordination, event routing, batched element flushing, and network interface detection.

### l8bus — Virtual Network Layer
Overlay networking providing VNic (client-side virtual network interface with service API, reconnection, circuit breaker) and VNet (server-side switch with service discovery via multicast modes: Proximity, Local, Leader, RoundRobin, All). Includes UDP/DNS peer discovery, health monitoring, and connection health scoring.

### l8services — Distributed Service Framework
Service lifecycle management, leader election, replication, distributed caching (`DCache`), 2-phase commit transactions, and Map-Reduce fan-out. `BaseService` provides CRUD with in-memory caching, Before/After hooks, transaction config, and web exposure. Built-in infrastructure services: `FileStore` (file upload/download with SHA-256 and AES encryption at rest), `CsvExport` (server-side paginated CSV generation), and `DataImport` (CSV/JSON/XML import with AI-assisted column mapping).

### l8orm — Object-Relational Mapping
Database CRUD as a distributed service. Auto-creates/migrates PostgreSQL tables from protobuf introspection. Query caching (30s TTL), optional in-memory write-through cache, time-series database support. Generates SQL from L8Query AST. Bidirectional object-to-relational conversion.

### l8web — HTTP/WebSocket Bridge
HTTPS server with TLS and bearer token auth that routes REST requests to VNic with leader/local/proximity routing. WebSocket manager for authenticated connections with notification push. REST client with GZIP and auto-retry. GraphQL client. SNI-based TLS reverse proxy. Webhook handling with VCS signature verification. Built-in endpoints: `/auth`, `/registry`, `/permissions`, `/captcha`, `/register`, `/tfa*`, `/ws`.

### l8secure — Security & AAA
Full `ISecurityProvider` implementation: authentication (password hashing, TFA/TOTP+QR, JWT), authorization (deny-before-allow role-based with pre-computed O(1) permission index), row-level data scoping via L8Query deny rules with `${userId}` and `${associateIds}` placeholders, field-level blanking, AES encryption, CAPTCHA, password policy enforcement, user registration. Activates CRUD services for users, roles, tokens, credentials, and portals (service area 73). Security configs are per-project JSON files compiled into Go plugins.

### l8common — Top-Level Convenience Layer
One-call service activation (`ActivateService`) wiring ORM, web, replication, and transactions. CRUD helpers (`GetEntity`, `PostEntity`, `PutEntity`, `EntityExists`). Validation framework with static validators (`ValidateRequired`, `ValidateEnum`, `ValidateMoney`) and a fluent builder. `NewServiceCallback` factory. Status transition machine. Money arithmetic. Infrastructure bootstrapping (`CreateResources`, `CreateVnic`, `CreateWebServer`, `OpenDBConnection`, `WaitForSignal`). Type registration. User provisioning helpers. Mock data upload client.

---

## Observation / Data Collection Stack

These projects implement the targets-collect-parse-cache pipeline for operational data collection projects (the "probler pattern").

### l8pollaris — Polling Configuration & Target Management
Defines what data to collect and manages target lifecycle. `PollarisCenter` service (service area 0) for poll configurations with hierarchical key lookup (name/vendor/series/family/software/hardware/version). `Targets` service (service area 91) for persisting targets in PostgreSQL with IP uniqueness validation and round-robin distribution to collectors. Projects must implement `TargetLinks` to route targets to their specific collector, parser, cache, and persist services.

### l8collector — Multi-Protocol Data Collection
Polls targets using SNMP, SSH, K8s client-go, REST, and GraphQL, then forwards changed results to parsers. `ProtocolCollector` interface with `Init`, `Protocol`, `Exec`, `Connect`, `Disconnect`, `Online`. Uses FNV-1a hashing for change detection. K8s collector features shared informer cache with admission webhook support.

### l8parser — Rule-Based Data Parsing
Receives raw collected data (CJob), applies rule-based transformations to produce structured protobuf inventory objects, and forwards them to inventory cache via PATCH/DELETE. `ParsingRule` interface with `Name()`, `ParamNames()`, `Parse()`. 16+ built-in rules. Includes vendor-specific boot configs for Cisco, Arista, Juniper, HPE, Dell, Fortinet, Huawei, Nokia, NVIDIA, Palo Alto, plus comprehensive K8s resource parsing.

### l8inventory — In-Memory Distributed Inventory Cache
Sits between parsers and ORM persistence, caching parsed data and optionally forwarding mutations to a linked persist service. `Activate()` registers an inventory cache for a given protobuf type and links ID. `InventoryCenter` provides CRUD on cached data.

---

## Cross-Cutting Services

These are shared services that most implementation projects include.

### l8events — Event Recording & Alarm State Machine
ORM-backed event recording (service area 76, immutable POST/PATCH/GET). Alarm state machine enforcing transitions between ACTIVE/ACKNOWLEDGED/CLEARED/SUPPRESSED. Event-to-category dispatching with 16+ built-in category parsers. Archiving with pluggable store. Maintenance window evaluation.

### l8alarms — Full Alarm Management Application
Complete alarm lifecycle application (service area 10, prefix `/alm/`) with 11 ORM-backed services: Alarm, Event, AlmDef (definitions), CorrRule (correlation engine), NotifPol (notification policies), EscPolicy (escalation scheduling), MaintWin (maintenance windows), AlmFilter, ArcAlarm/ArcEvent (archive), AlmOverlay (topology enrichment). Has its own proto files, UI, mock data, and tests.

### l8notify — Multi-Channel Notification Delivery
Standalone library (zero l8* dependencies) for notification dispatch. Supports email/SMTP, webhook with HMAC signing, Slack, PagerDuty, and extensible custom senders. Template rendering with `{{key}}` placeholders. Per-key cooldown and hourly rate limiting. Time-delayed multi-step escalation with `StepHandler` callback.

### l8logfusion — Distributed Log Collection
Log-agent tails log files on each node (with /proc scanning, rotation detection, crash-resume via byte offset). Log-server receives batches, persists to disk by source IP/filename, and serves via REST API. Service: `logs` (service area 87). Every project's `log-vnet` and `log-agent` binaries use this.

### l8topology — Topology Visualization
Discovers network nodes/links from inventory services and computes positions using four layout algorithms (hierarchical, circular, radial, force-directed) plus geographic layout. Projects implement `ITopoDiscovery` interface to feed inventory data into the renderer. Ships a WebGL-based browser UI.

### l8agent — AI Agent (LLM Chat)
LLM-powered chat system backed by the Claude API with ORM persistence. CRUD services for conversations, messages, and prompts. Introspector-based schema generation for tool definitions. Field classification for data privacy masking. Tool call execution against vnic services. Ships JS/CSS for desktop and mobile chat UI.

### l8opensim — API/Device Simulator
Creates virtual devices with TUN interfaces responding to SNMP/SSH/REST queries with configurable resource profiles. Used exclusively for testing the observation pipeline (l8collector/l8parser). Not an importable library — standalone binaries only.

---

## UI Library

### l8ui — Shared UI Component Library
Zero-dependency vanilla JavaScript/CSS library (no npm, no bundler). Loaded via script/link tags in strict dependency order. Added to projects as a git submodule.

**Core:** Config, utils, renderers, factories (enum, column, form, reference, SVG, section, module config), notification, popup (stacking modals), forms (fields, data collection, pickers, modal facade), data source (shared fetch/pagination).

**Data Views:** Table (paginated, sortable, filterable), chart (bar/line/pie SVG), kanban, calendar, timeline, gantt, tree grid, wizard.

**Inputs:** Date picker, reference picker (entity search), input formatters (currency, phone, SSN, percentage, etc.), file upload.

**Module System:** Service registry, module CRUD, module navigation, toggle tree, module filter, module factory (one-call bootstrap), view factory (pluggable view types with switcher), dashboard KPI widgets.

**Built-in System Module:** Health monitoring, security (users/roles/credentials CRUD), module management with dependency graph, log file viewer with tree browser, data import (CSV/JSON/XML with AI-assisted column mapping).

**Mobile (`m/`):** Full mobile equivalents — card-based navigation, mobile forms/table/popup/confirm/datepicker/reference picker, mobile view factory with chart/kanban/calendar/timeline/gantt/tree/wizard.

**Auth Pages:** Login (with TFA support), registration (with CAPTCHA).

**AI Chat:** Desktop and mobile chat interface for l8agent.

**Theming:** All components use `--layer8d-*` CSS custom properties. Dark mode handled centrally via `[data-theme="dark"]`.

---

## Canonical Implementation Projects

These are complete implementation projects that serve as references for new projects.

### l8erp — Enterprise Resource Planning (canonical for ERP-style projects)
Full ERP with 12 modules (HCM, FIN, SCM, Sales, MFG, CRM, PRJ, BI, DOC, ECOM, COMP, SYS). Canonical reference for project structure, service patterns, UI patterns, mock data, deployment (Docker/K8s), run-local.sh, login.json, and all four K8s deployment modes.

### probler — Network/Infrastructure Observation (canonical for observation projects)
Full observation pipeline: targets, collectors (SNMP/SSH/K8s/REST), parsers, inventory cache, ORM persistence, topology, alarms. Canonical reference for the l8pollaris pattern (targets-collect-parse-cache), binary layout, Dockerfiles, and K8s manifests for observation projects.

---

## How to Use This Document

When planning a new implementation project:

1. **Classify the project**: ERP-style (CRUD entities, forms, persistence) or observation (targets, collection, parsing, caching). Use l8erp or probler as the canonical reference accordingly.

2. **Identify needed services**: Every project uses l8types through l8common (the core stack). Most also need l8secure, l8logfusion, l8events, and l8ui. Observation projects additionally need l8pollaris, l8collector, l8parser, and l8inventory.

3. **Check for existing functionality** before building: l8services has FileStore, CsvExport, DataImport built in. l8events has event recording. l8alarms has full alarm management. l8notify has notification delivery. l8topology has visualization. l8agent has AI chat.

4. **Examine the relevant canonical project** (l8erp or probler) for the specific pattern you need — service structure, UI wiring, mock data, deployment artifacts, K8s manifests, security config.
