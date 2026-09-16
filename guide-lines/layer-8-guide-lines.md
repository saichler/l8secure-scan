# Layer 8 Ecosystem — Implementation Guide Lines

Compacted rule reference for AI context. All 75 rules from `l8book/rules/` combined.
Preserves all technical details (field names, interfaces, patterns, code). Not intended for human reading.

---

## AddingModule

### Desktop Steps

**1. Config** (`l8ui/projects/projects-config.js`):
```js
(function() {
    'use strict';
    const svc = Layer8ModuleConfigFactory.service;
    const mod = Layer8ModuleConfigFactory.module;
    Layer8ModuleConfigFactory.create({
        namespace: 'Projects',
        modules: {
            'planning': mod('Planning', 'icon-emoji', [
                svc('projects', 'Projects', 'icon', '/60/Project', 'Project'),
                svc('tasks', 'Tasks', 'icon', '/60/Task', 'ProjectTask')
            ])
        },
        submodules: ['ProjectPlanning']
    });
})();
```

**2. Sub-module data** (per sub-module): enums, columns, forms files under `l8ui/projects/planning/`.

Enums pattern:
```js
window.ProjectPlanning = window.ProjectPlanning || {};
ProjectPlanning.enums = { STATUS: {0:'Unknown',1:'Draft',2:'Active',3:'Done'}, STATUS_VALUES: {...}, STATUS_CLASSES: {...} };
ProjectPlanning.render = {};
ProjectPlanning.render.status = Layer8DRenderers.createStatusRenderer(ProjectPlanning.enums.STATUS, ProjectPlanning.enums.STATUS_CLASSES);
```

Columns pattern: `ProjectPlanning.columns = { Project: [...] }; ProjectPlanning.primaryKeys = { Project: 'projectId' };`

Forms pattern: `ProjectPlanning.forms = { Project: { title: 'Project', sections: [{...}] } };`

**3. Init** (`projects-init.js`):
```js
Layer8DModuleFactory.create({
    namespace: 'Projects', defaultModule: 'planning', defaultService: 'projects',
    sectionSelector: 'planning', initializerName: 'initializeProjects',
    requiredNamespaces: ['ProjectPlanning']
});
```

**4. Section HTML** (`sections/projects.html`): Container IDs follow `{moduleKey}-{serviceKey}-table-container`. CSS classes use `l8-` prefix.

```html
<div class="section-container l8-section">
    <div class="l8-module-tabs"><button class="l8-module-tab active" data-module="planning">...</button></div>
    <div class="l8-module-content active" data-module="planning">
        <div class="l8-subnav"><a class="l8-subnav-item active" data-service="projects">Projects</a>...</div>
        <div class="l8-service-view active" data-service="projects">
            <div class="l8-table-container" id="planning-projects-table-container"></div>
        </div>
    </div>
</div>
```

**5. app.html**: Script tags in order: config, enums, columns, forms, init.

**6. sections.js**: Add section mapping + initializer.

**7. Reference registry**: `Layer8DReferenceRegistry.register({Project:{idColumn:'projectId',displayColumn:'name',displayLabel:'Project'}});`

### Mobile Steps

**1. Data files** under `m/js/projects/`: enums (prefix `MobileProjectPlanning`, use `Layer8MRenderers`), columns (add `primary:true`/`secondary:true`), forms.

**2. Registry** (`projects-index.js`): `Layer8MModuleRegistry.create('MobileProjects', {'Planning': MobileProjectPlanning});`

**3. Nav config**: Add to `layer8m-nav-config-base.js` modules array with `hasSubModules:true`. Add config block with `subModules` and `services` to category file.

**4. Nav.js**: Add `window.MobileProjects` to registry arrays in `_getServiceColumns`, `_getServiceFormDef`, `_getServiceTransformData`.

**5. m/app.html**: Script tags + sidebar link `data-section="dashboard" data-module="projects"`.

**6. Reference registry**: Create `layer8m-reference-registry-projects.js` using `Layer8RefFactory`, call `Layer8MReferenceRegistry.register()`.

### Checklist

Desktop: config, per-submodule data (enums/columns/forms), init, section HTML (correct container IDs), app.html scripts, sections.js, reference registry.

Mobile: per-submodule data (with `primary`/`secondary`), registry index, nav config (`hasSubModules`), nav.js arrays, m/app.html scripts+sidebar, reference registry.

Rules: field names must match `.pb.go`, endpoint names max 10 chars, CSS uses `l8-` prefix. Desktop: `new Layer8DTable(options)` then `table.init()`. Mobile: `new Layer8MEditTable(containerId, config)` -- no init() needed.

## AppHtmlBodyFromL8erp

Copy `<body>` from l8erp's `app.html` -- never write from scratch. L8ui CSS targets specific selectors that break if changed.

### Required DOM Structure (do NOT change):
- `div.app-container` > `header.app-header` + `nav.sidebar > ul.nav-menu > li > a.nav-link[data-section]` + `main.main-content > #content-area`
- `#layer8d-popup-root` (do NOT omit)
- `div.user-menu > span.username + button.logout-btn`

### Adapt only:
- `<title>`, `<h1>`, `<p class="header-subtitle">` text
- Sidebar `<li>` items with correct `data-section` values
- `alt` attributes

### app.js pattern:
- `DOMContentLoaded`, `.nav-link` selectors, load into `#content-area` (not `#main-content`)
- Remove/guard `Layer8DModuleFilter.load()` if no ModConfig service

### css/base-core.css is required:
Copy `css/base-core.css` and `css/responsive.css` from l8erp -- they define the entire grid layout (`.app-container`, `.sidebar`, `.main-content`, etc.). Without them, everything stacks vertically.

### Built-in SYS module section HTML:
Section HTML hosting built-in l8ui modules (e.g., `sections/system.html`) must also be copied from l8erp. Required container IDs:

| Module | Container id |
|---|---|
| Health | `health-table-container` |
| Modules | `modules-settings-container` |
| Logs | `logs-table-container` |
| Data Import | `dataimport-container` |
| Security | `security-<service>-table-container` |

Mismatched ids produce silently empty tabs.

## ArchitectureOverview

Configuration-driven module pattern: behavioral logic in shared library, modules supply only data (configs, enums, columns, forms). Desktop (`Layer8D*`): table-based + sidebar. Mobile (`Layer8M*`): card-based + drill-down nav.

### Desktop Dependency Graph
```
Layer8DConfig -> Layer8DUtils <- Layer8DRenderers
  +- Factory: EnumFactory, RefFactory, ColumnFactory, FormFactory, SvgFactory, ModuleConfigFactory
  +- Components: Table, DatePicker, InputFormatter, ReferencePicker, Notification, Popup, ReferenceRegistry
  +- Forms: FormsFields, FormsData, FormsPickers, FormsModal -> Forms facade
  +- Module: ServiceRegistry, ModuleNavigation, ModuleCRUD, ToggleTree, ModuleFilter -> ModuleFactory
  +- Views: ViewFactory, ViewSwitcher, DataSource, Chart(Bar/Line/Pie), Kanban, Timeline, Calendar, Gantt, TreeGrid, Wizard
  +- Widget, Markdown, L8AgentChat
```

### Mobile Dependency Graph
```
Layer8MConfig -> Layer8MAuth, Layer8MUtils
  +- Desktop Shared: DConfig, DUtils, DRenderers, DReferenceRegistry, all factories
  +- Components: MPopup, MConfirm, MTable, MEditTable, MForms, MDatePicker, MReferenceRegistry/Picker, MRenderers
  +- Module: MModuleRegistry, DToggleTree, DModuleFilter, NAV_CONFIG, MNavCrud, MNavData, MNav
  +- Views: MViewFactory, ViewSwitcher(shared), MChart, MKanban, MCalendar, MTimeline, MGantt, MTreeGrid, MWizard
  +- MDataSource, L8AgentChatMobile
```

### Project Structure
```
l8ui/: shared/, edit_table/, popup/, reference_picker/, datepicker/, input_formatters/,
       chart/, kanban/, calendar/, timeline/, tree_grid/, gantt/, wizard/, dashboard/,
       notification/, login/, register/, l8agent/(+m/), m/(js/,css/),
       sys/(security/,modules/,health/,logs/,dataimport/), images/, font/
```

Theming: all components use `--layer8d-*` CSS properties from `layer8d-theme.css`. Dark mode via `[data-theme="dark"]` overrides. No per-component dark mode blocks.

## AssociateIdsScopeView

`L8User.associate_ids` (field 19, repeated string) enables multi-entity scoping via `${associateIds}` placeholder in deny rules. Expands to `[id1,id2,id3]` bracket notation for L8Query `not in`.

### Populate via Security API or config JSON:
```json
"users": { "guardian1": { "associateIds": ["STU-001", "STU-002"], "roles": {"guardian": true} } }
```

### Deny rule pattern:
```json
{ "elemType": "Student", "allowed": false, "actions": {},
  "attributes": { "Student": "select * from Student where studentId not in ${associateIds}" } }
```

Resolves to: `where studentId not in [STU-001,STU-002]` -- rows NOT in list are denied.

Empty `associate_ids` resolves to `[]` -- denies all rows (secure default). Can combine with `${userId}`. IDs must not contain `,[]` characters.

## CanonicalProjectSelection

Classify project objective before choosing canonical reference:

| Objective | Canonical |
|---|---|
| ERP-style (CRUD, persistence) | `../l8erp` |
| Observation/collection (targets, live state) | `../probler` + `../l8collector` + `../l8parser` |

Probler sub-references: `l8collector` for collection stage, `l8parser` for parsing stage, `probler` for targets model and full pipeline.

L8ui component rules apply to both families. Fall back to l8erp only for genuinely generic patterns (k8s entries, login.json) when probler has no equivalent.

## CascadingHideZeroChildren

Cascading hide logic must distinguish "all children hidden by filter" from "no children by design." Check `allItems.length > 0` before hiding parent:

```javascript
var allItems = container.querySelectorAll('.child-item');
if (allItems.length > 0) {
    var visibleItems = container.querySelectorAll('.child-item:not([style*="display: none"])');
    if (visibleItems.length === 0) parentTab.style.display = 'none';
}
// allItems.length === 0 means no sub-nav by design -- leave visible
```

Applies to: `layer8d-permission-filter.js` `applyToSection`, module filter, mobile nav filtering.

## CleanupTestBinaries

Never use `go build ./path/to/main/package/` -- produces a binary in cwd for `main` packages.

Use `go build ./...` (preferred, discards all binaries) or `go build -o /dev/null ./path/to/main/package/`.

## CompoundFormFieldDataCollection

Compound fields (e.g., `money`) render multiple sub-elements with names like `fieldKey.__amount`, `fieldKey.__currencyId`. `form.elements[field.key]` returns null, silently skipping data collection.

Fix: add compound types to the guard exception:
```javascript
if (!element && field.type !== 'money' && field.type !== 'newCompoundType') return;
```
Then use `form.elements[field.key + '.__subField']` to find sub-elements.

Current compound types: `money` (renders `__currencyId` select + `__amount` input). Desktop `layer8d-forms-data.js` is vulnerable; mobile `layer8m-forms.js` is not (iterates all elements).

## DataCompletenessPipeline

Every protobuf field must flow: **Proto -> Forms/Columns -> Mock data**. Gaps produce silent empty cells.

**Stage 1 (Proto -> Forms/Columns):** Every non-system field must appear in both `*-forms.js` and `*-columns.js`. System fields excluded: primary key ID, `auditInfo`, `customFields`. Dependent groups must be complete (period->month/quarter/year).

**Stage 2 (Columns -> Mock Data):** Every column field must be populated with non-zero values. Protobuf `omitempty` omits zeros. Enum fields must use non-zero values (0=UNSPECIFIED, omitted).

**Stage 3 (Services -> Generators):** ALL services need mock generators. Commonly missed: line items, secondary entities, break tables, junction tables.

```bash
# Verify proto vs form fields
grep -A 40 "type ModelName struct" go/types/<module>/*.pb.go | grep -oP 'json:"\K[^,"]+' | sort
grep -oP "key:\s*'[^']+'" <forms-file> | sort
```

## DataImportSystem

System section module for CSV/JSON/XML imports with AI-assisted column mapping.

| File | Object | Purpose |
|------|--------|---------|
| `l8dataimport.js` | `L8DataImport` | Main controller, 3 tabs |
| `l8dataimport-templates.js` | `L8DITemplates` | Template CRUD, AI mapping |
| `l8dataimport-transfer.js` | `L8DITransfer` | Export/import templates |
| `l8dataimport-execute.js` | `L8DIExecute` | Run import |

Endpoints: `/erp/0/ImprtTmpl` (CRUD), `/erp/0/ImprtAI` (POST, AI mapping), `/erp/0/ImprtInfo` (POST, field metadata), `/erp/0/ImprtExec` (POST, execute), `/erp/0/ImprtXfer` (POST export, PUT import).

```js
L8DataImport.initialize()    // into #dataimport-container
L8DataImport.showTab(name)   // 'templates' | 'transfer' | 'import'
```

CSS: `l8di-` prefix, uses `--layer8d-*` tokens. Loading: after `l8logs.js`, before `l8sys-init.js`.

## DateFieldRenderingPipeline

Date/datetime/time rendering must handle both numeric AND string-typed values (protobuf int64 serializes as string). Never guard with `typeof value === 'number'` -- always coerce first.

```javascript
function formatDate(timestamp) {
    if (timestamp === null || timestamp === undefined) return '-';
    if (typeof timestamp === 'string') timestamp = Number(timestamp);
    if (isNaN(timestamp)) return '-';
    // ...
}
```

Three rendering paths must ALL handle `date`, `datetime`, `time`:
1. Utility functions (`layer8d-utils.js`)
2. Read-only form rendering (`layer8d-forms-fields.js` isReadOnly block)
3. Inline table cell rendering (`layer8d-forms-fields-ext.js` formatInlineTableCell)

Symptom: dates show raw numbers like `1704067200` -- completely silent.

## DemoDirectorySync

Never edit, copy to, or sync files in `/go/demo/`. It is auto-generated by `run-local.sh` and rebuilt from scratch each run. Source of truth is always the source web directory (e.g., `go/erp/ui/web/`).

## DeploymentArtifacts

Every PRD introducing a new deployable service must include: `build.sh`, `Dockerfile`, K8s YAMLs (4 modes), KIND scripts, updates to `build-all-images.sh` and `deploy.sh`/`undeploy.sh`.

### Docker image pattern:
- Multi-stage: `saichler/builder:latest` build stage, `saichler/<project>-security:latest` (or `<project>-postgres`) runtime
- `build.sh`: `docker build --no-cache --platform=linux/amd64 -t saichler/<image>:latest . && docker push`

### Base image rule:
Every project MUST have its own `-security` and `-postgres` base images. Never use `erp-security`/`erp-postgres` for non-ERP projects -- causes protobuf namespace conflict panics.

### l8erp reference images:

| Image | Directory | Base | K8s Kind |
|-------|-----------|------|----------|
| `saichler/erp` | `go/erp/main/` | `erp-postgres` | StatefulSet |
| `saichler/erp-web` | `go/erp/ui/` | `erp-security` | DaemonSet(hostNetwork) |
| `saichler/erp-vnet` | `go/erp/vnet/` | `erp-security` | DaemonSet(hostNetwork) |
| `saichler/erp-logs-vnet` | `go/logs/vnet/` | `erp-security` | DaemonSet(hostNetwork) |
| `saichler/erp-log-agent` | `go/logs/agent/` | `erp-security` | DaemonSet |

Does NOT apply when adding services to existing images, pure UI changes, or library changes.

## EnumRendererColumnCascade

All rules below cause the same cascading failure: TypeError in enums/columns IIFE -> `Module.render`/`Module.columns` undefined -> ALL tables fall back to DEFAULT_COLUMNS (`id`, `name`, `status`). Silent unless DevTools is open.

### Rule 0: Enum Factory API
`Layer8EnumFactory` has exactly 3 methods:

| Method | Returns |
|--------|---------|
| `create(defs)` | `{enum, values, classes}` |
| `simple(labels)` | `{enum}` |
| `withValues(defs)` | `{enum, values}` |

`createStatus`, `createEnum` do NOT exist -- TypeError kills the IIFE.

### Rule 1: Renderer API
`createEnumRenderer` does NOT exist. Use:
- `(v) => renderEnum(v, map)` for plain enums
- `createStatusRenderer(map, classes)` for status enums (returns function)

### Rule 2: f.select() enum reference
`f.select('field', 'Label', enums.SOME_ENUM)` -- if `SOME_ENUM` not exported, `field.options` is undefined, crashes on row click with `Cannot convert undefined or null to object`.

### Rule 3: Column factory methods
Every `col.*()` call must exist in `layer8-column-factory.js`. `col.enum()`/`col.status()` require 4 args (4th is renderer function).

Available: `col`, `basic`, `number`, `boolean`, `status`, `enum`, `date`, `money`, `period`, `id`, `custom`, `link`.

```bash
# Verify enum factory calls
grep -rn 'factory\.\w\+(' --include="*-enums.js" | grep -v 'factory\.create\b\|factory\.simple\b\|factory\.withValues\b'
```

## EventsServiceRequired

Every project must include `l8events`. Required infrastructure.

**Backend main.go:**
```go
import evtservices "github.com/saichler/l8events/go/services"
// after ActivateAllServices:
evtservices.ActivateEvents(dbcred, dbname, nic)
```

**UI main.go:**
```go
import l8events "github.com/saichler/l8types/go/types/l8events"
l8c.RegisterType(resources, &l8events.EventRecord{}, &l8events.EventRecordList{}, "EventId")
```

```bash
grep -n "ActivateEvents" go/<project>/main/*.go
grep -n "EventRecord" go/<project>/ui/*.go
ls go/vendor/github.com/saichler/l8events/
```

## FileUploadPattern

Use `FileStore` service + `Layer8FileUpload` UI for entity attachments. Entities store `storage_path` string fields only -- never binary data.

**Flow:** `Layer8FileUpload.upload(file)` -> POST `/0/FileStore` -> validates (5MB max), SHA-256, encrypts, writes to `/data/l8files/{docId}/{version}/{fileName}` -> returns `{storagePath, fileName, fileSize, mimeType, checksum}` -> form stores storagePath string.

**Proto pattern (single file):**
```protobuf
string image_storage_path = 10;
string image_file_name = 11;
int64 image_file_size = 12;
```

**Proto pattern (gallery):** Use `repeated MyEntityImage images` with `storage_path`, `file_name`, `file_size`, `caption`, `sort_order` fields.

**Form:** `f.file('imageStoragePath', 'Image')` or `f.inlineTable('images', 'Images', [{key:'storagePath',type:'file'},...])`.

**Download:** `Layer8FileUpload.download(storagePath, fileName)` -> PUT to FileStore -> decrypt -> base64 download.

No file I/O in ServiceCallbacks. Reference: l8physio `PhysioExercise`.

## FollowInstructionsVerifyUserIssue

When user reports a bug: (1) reproduce and verify the exact issue first, (2) identify root cause of THAT issue, (3) then fix. Do not assume, fix a different problem, or claim done without confirming the specific symptom is addressed.

Before claiming done: re-read original complaint, trace code path for that scenario, verify fix touches that path. If app uses copied files (go/demo/), note rebuild is needed.

## FrameworkInterfaceBoundaries

Never add methods/types/interfaces to `l8types/go/ifs/` for feature work in consuming projects. Framework interfaces change only for fundamental model changes.

Three sub-rules:
1. **Feature hooks go in implementation layer** (`l8services`, `l8web`, project code), not contract layer (`l8types/go/ifs/`).
2. **Refactor before abstracting** -- extract/reorganize existing code rather than creating new abstractions.
3. **Use existing extension points first**: `IServiceHandler` (CRUD), `IServiceCacheListener` (cache notifications), `ITransactionConfig`, `IWebService`, `ServiceCallback` (Before/After hooks).

If genuinely needed: flag to framework owner, don't add yourself.

## ImmutabilityUiAlignment

Backend immutability must be reflected in UI.

**Entity-level** (rejects PUT): table read-only mode, all detail fields display-only, hide Edit/Save buttons.

**Field-level** (specific fields protected on PUT): those fields read-only in edit forms, editable fields normal.

Checklist: backend validation, UI config (read-only mode), UI forms (display-only fields), verify UI doesn't offer controls backend will reject.

## IndexHtmlRedirect

Every web directory (`go/<project>/ui/web/`) must have `index.html` redirecting to `login.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta http-equiv="refresh" content="0; url=login.html">
    <title>PROJECT_TITLE</title>
    <script>window.location.href = 'login.html';</script>
</head>
<body><p>Redirecting to <a href="login.html">PROJECT_TITLE</a>...</p></body>
</html>
```

Create when setting up web directory alongside `login.html`, `app.html`, `login.json`.

## InlinePopupRenderingParity

Every form rendering context (popup, inline, mobile) must call the same functions in the same order as the canonical popup path.

### openViewForm pipeline:
1. `generateFormHtml(formDef, data)`
2. `setFormContext(formDef, serviceConfig)`
3. Body class `probler-popup-body`
4. `setTimeout(50ms)`
5. Inside timeout: `attachDatePickers(body)` (calls `attachInputFormatters` + `attachReferencePickers`)
6. Disable all inputs

### openEditForm adds:
7. `updateFormContext({isEdit, recordId, onSuccess})`
8. `attachInlineTableHandlers(body)`
9. Save reads from `getFormContext()`
10. Errors via `Layer8DNotification.error()`

Skipping any step causes silent rendering differences (raw IDs instead of names, unformatted dates, non-clickable child rows).

## IntrospectorNodesParams

Never call `Introspector().Nodes(true, true)` -- `onlyLeafs` + `onlyRoots` is contradictory, always returns empty list.

| Call | Meaning |
|------|---------|
| `Nodes(false, false)` | All nodes |
| `Nodes(true, false)` | Leaf nodes only |
| `Nodes(false, true)` | Root nodes only (top-level registered types) |

## JsProtobufFieldNames

Every JS field name must be verified against protobuf JSON name. Never guess.

**Workflow:** Read `.pb.go`, extract JSON name from `protobuf:` tag (`json=fieldName,proto3"`), use that exact name.

Protobuf `snake_case` -> JSON `camelCase`: `warehouse_id` -> `warehouseId`.

Common mismatches: singular/plural (`projectedInflow`/`projectedInflows`), abbreviated (`allocatedQty`/`allocatedQuantity`), missing prefix (`reason`/`reasonCode`), fabricated fields.

**Nav config `idField` (5x regression):** Must use JSON name (lowercase). Wrong casing -> `getItemId` returns `undefined` -> row clicks silently do nothing.

```javascript
// WRONG: { idField: 'Id' }    -- Go struct field name
// CORRECT: { idField: 'id' }  -- JSON tag name
```

```bash
grep -A 30 "type TypeName struct" go/types/<module>/*.pb.go | grep 'json:"'
```

Symptoms: empty columns (silent), row clicks do nothing (silent), server log `Unknown attribute`.

## K8sRules

### Directory Location
`k8s/` at project root, NOT under `go/`.

### Required YAML Entries (all 4 modes)
- Namespace with `labels: {name: <ns>}`
- Resource metadata with `labels: {app: <name>}`
- Container `env` with `NODE_IP` from `status.hostIP`
- Volume name `hdata` (not `data`), mountPath `/data`

### Four Deployment Modes Required

**Local** (`-local.yaml`): `hostPath` volumes, `DirectoryOrCreate`. DaemonSet for per-node, Deployment for scalable, StatefulSet for stateful.

**Bare-metal** (`-baremetal.yaml`): StorageClass `rancher.io/local-path`, `reclaimPolicy:Delete`. DaemonSets -> StatefulSet + `podAntiAffinity` (hostname). All use `volumeClaimTemplates`.

**GKE** (`-gke.yaml`): StorageClass `kubernetes.io/gce-pd` (pd-standard), `reclaimPolicy:Retain`. Single shared PVC (`<project>-data`, 50Gi). DaemonSets unchanged.

**KIND** (`-kind.yaml`): Same as bare-metal but remove custom StorageClass, use `storageClassName: standard`.

### KIND Scripts
`kind-start.sh`: install kind, create cluster (1 cp + 1 worker), load images, deploy in 3 phases.
`kind-stop.sh`: delete cluster, cleanup.

### Identical across modes:
Namespace, images, env vars, ports, Services, RBAC, ConfigMaps, Jobs, Webhooks.

### Differs only:
Storage provisioning, workload kind (DaemonSet vs StatefulSet+anti-affinity), volume source.

### Conversion: Local -> Bare-metal/KIND

| Local | Bare-metal/KIND |
|-------|----------------|
| DaemonSet | StatefulSet + anti-affinity + volumeClaimTemplates |
| Deployment | StatefulSet + volumeClaimTemplates |
| StatefulSet | Add volumeClaimTemplates |

### Conversion: Local -> GKE
Replace `hostPath` with `persistentVolumeClaim: claimName: <project>-data`.

```bash
# Verify
ls k8s/<project>-{local,baremetal,gke,kind}.yaml k8s/kind-{start,stop}.sh
grep -c "StorageClass\|PersistentVolumeClaim\|volumeClaimTemplates" k8s/<project>-local.yaml  # expect 0
grep "rancher.io/local-path" k8s/<project>-baremetal.yaml
grep "kubernetes.io/gce-pd" k8s/<project>-gke.yaml
grep -c "kind: StorageClass" k8s/<project>-kind.yaml  # expect 0
diff <(grep "image:" k8s/<project>-local.yaml | sort) <(grep "image:" k8s/<project>-baremetal.yaml | sort)
```

Canonical: `probler/k8s/` (all 4 modes + KIND scripts).

## L8AgentChat

AI chat with conversation management, markdown responses, inline data tables.

**Desktop:**
```js
L8AgentChat.init({ containerId: 'agent-container', chatEndpoint: '/0/AgentChat' });
L8AgentChat.sendMessage(text)
L8AgentChat.loadConversation(id)
L8AgentChat.newConversation()
```

**Mobile** (`L8AgentChatMobile`): Same API, uses `Layer8MAuth` for HTTP, `Layer8MTable` for inline tables.

```js
L8AgentChatMobile.init({ containerId: 'agent-container', chatEndpoint: '/0/AgentChat' });
```

## L8Logs

System Log Viewer in the System section. File system tree browser with paginated log content.

```js
L8Logs.initialize()   // Renders tree and loads file list
L8Logs.refresh()      // Reloads tree data
```

Uses L8Query `select * from l8file where path="*" mapreduce true` for the tree, paginated 5KB chunks for file content.

## L8PollarisBinaryDeployment

Projects using l8pollaris (collection/parsing/inventory/orm pattern) MUST mirror probler's binary layout and deployment artifacts exactly.

### Per-service directory layout
```
go/<module>/<service>/
├── main.go
├── build.sh        # docker build --no-cache --platform=linux/amd64 + docker push
└── Dockerfile      # Multi-stage: saichler/builder -> saichler/<project base> final
```

### Required separate binaries

| Service | Purpose | Probler ref |
|---------|---------|-------------|
| `collector` | Collects raw data from targets | `go/prob/collector/` |
| `parser` | Parses collected data into typed models | `go/prob/parser/` |
| `orm` | Persists parsed data via ORM | `go/prob/orm/` |
| `inv_*` | One per inventory domain | `go/prob/inv_box/`, etc. |
| `vnet` | Virtual network backbone | `go/prob/vnet/` |
| `log-vnet`, `log-agent` | Distributed logging | `go/prob/log-vnet/`, etc. |

Each is a separate process, image, and K8s resource -- never collapsed.

### Deployment artifacts per service
1. `build.sh` -- standard docker build+push
2. `Dockerfile` -- multi-stage; MUST use project's own base images (`saichler/<project>-security`), never another project's (mismatched base images cause protobuf namespace conflict panics)
3. K8s manifests -- four YAMLs: `-local.yaml`, `-baremetal.yaml`, `-gke.yaml`, `-kind.yaml`
4. KIND scripts -- `kind-start.sh`, `kind-stop.sh`
5. `k8s/deploy.sh`, `k8s/undeploy.sh` -- correct dependency order
6. `build-all-images.sh` -- calls each build.sh

### Process
Copy from `../probler/` and adapt image name, binary name, namespace, paths. Do NOT hand-write from scratch.

### Verification
```bash
for svc in collector parser orm inv_box; do
  test -f go/<module>/$svc/build.sh   || echo "MISSING: $svc/build.sh"
  test -f go/<module>/$svc/Dockerfile || echo "MISSING: $svc/Dockerfile"
done
for mode in local baremetal gke kind; do
  test -f k8s/<project>-${mode}.yaml || echo "MISSING"
done
```

Does NOT apply to pure ERP projects or services outside the collection/parsing/orm chain.

## L8QueryRules

### Rule 1: GET Requests Require L8Query
Every GET MUST include `?body=` with JSON-encoded L8Query. No bare GETs.

```javascript
// CORRECT
const query = 'select * from L8ImportTemplate';  // protobuf type name, NOT ServiceName
const body = encodeURIComponent(JSON.stringify({ text: query }));
fetch('/erp/0/ImprtTmpl?body=' + body, { method: 'GET', headers: getHeaders() })

// WRONG -- bare GET
fetch('/erp/0/ImprtTmpl', { method: 'GET', headers: getHeaders() })
```

| Component | Example | Used Where |
|-----------|---------|------------|
| ServiceName | `ImprtTmpl` | URL path |
| Protobuf type | `L8ImportTemplate` | L8Query `from` clause |

Layer8DTable/Layer8DDataSource handle this automatically. POST/PUT/DELETE use JSON body.

Error: `Cannot find pb for method GET` + 400 Bad Request.

### Rule 2: SELECT * for Detail Popups
Detail popups MUST use `select * from ...`. Specific column lists leave fields blank silently.

### Rule 3: GetEntities with Empty Filter
`common.GetEntities` with zero-value filter returns nothing. Use L8Query `select * from <ProtoType>` instead.

```bash
grep -n "fetch('/erp/" *.js | grep "GET" | grep -v "?body="
```

## L8UICopyToNewProject

Before implementing ANY UI, add l8ui as a git submodule:
1. `cp <path-to-l8ui>/setup-l8ui-submodule.sh <new-project>/go/<project>/ui/web/`
2. `cd <new-project>/go/<project>/ui/web/ && ./setup-l8ui-submodule.sh`
3. Create project-specific files (nav configs, reference registries) inside the new project
4. Update HTML paths to use `../l8ui/`

Do NOT `cp -r` l8ui from another project, use cross-project paths, or symlink.

## L8UINoProjectSpecificCode

`l8ui/` MUST contain only generic, project-agnostic components. No project names, endpoints, service areas, hardcoded prefixes (`/erp/`, `/physio/`), or conditional project-checking logic.

Project-specific code goes in the project's own directories (`erp-ui/`, project-level `js/`, `sections/`) and registers via extension points (`Layer8MReferenceRegistry.register()`, `Layer8SvgFactory.registerTemplate()`, nav config objects).

## L8UIThemeCompliance

All l8ui components MUST use `--layer8d-*` CSS custom properties from `layer8d-theme.css`.

| Purpose | Use | Never Use |
|---------|-----|-----------|
| Primary accent | `var(--layer8d-primary)` | `var(--accent-color)` |
| White/card bg | `var(--layer8d-bg-white)` | `var(--bg-primary)` |
| Light bg | `var(--layer8d-bg-light)` | `var(--bg-secondary)` |
| Input bg | `var(--layer8d-bg-input)` | `var(--bg-tertiary)` |
| Dark text | `var(--layer8d-text-dark)` | `var(--text-primary)` |
| Medium text | `var(--layer8d-text-medium)` | `var(--text-secondary)` |
| Border | `var(--layer8d-border)` | `var(--border-color)` |
| Status | `var(--layer8d-success/warning/error)` | hardcoded hex |

No `[data-theme="dark"]` blocks in component CSS -- dark mode is central in `layer8d-theme.css`.

Buttons: use `layer8d-btn layer8d-btn-primary/secondary layer8d-btn-small`. No per-view button styles.

JS colors: use `Layer8DChart.readThemeColor(varName, fallback)` and `Layer8DChart.getThemePalette()`. No hardcoded hex.

```bash
grep 'var(--accent-color\|var(--bg-primary\|var(--text-primary\|var(--border-color' <file>  # should be empty
```

## Layer8CsvExport

`window.Layer8CsvExport` (shared desktop/mobile). Server-side CSV generation via `POST /erp/0/CsvExport`.

```js
Layer8CsvExport.export({ modelName: 'Employee', serviceName: 'Employee', serviceArea: 30, filename: 'Employee' });
Layer8CsvExport.parseEndpoint(endpoint)    // '/erp/30/Employee' -> { serviceName, serviceArea }
```

Export button appears automatically in pagination bars when `endpoint` and `modelName` are set.

## Layer8ProjectLocations

All projects are siblings under the same parent. `../projectname` resolves from any project root.

| Project | Purpose |
|---------|---------|
| `l8ui` | Shared UI library. Contains `setup-l8ui-submodule.sh` and `rules/`. |
| `l8erp` | Canonical ERP reference (structure, run-local.sh, login.json, K8s, UI, mocks) |
| `probler` | Canonical for targets/collect/parse/cache pipeline |
| `l8collector` | Canonical collection stage |
| `l8parser` | Canonical parsing stage |
| `l8topology` | Topology visualization |
| `l8opensim` | API simulation |
| `l8agent` | AI agent |
| `l8logfusion` | Distributed log collection |
| `l8events` | Event processing |
| `l8alarms` | Alarm lifecycle |
| `l8notify` | Notification handling |
| `l8physio` | Physiotherapy project |
| `l8myfamily` | Android app |

See `canonical-project-selection.md` for choosing l8erp vs probler as canonical reference.

## Layer8DApiReference

### Layer8DConfig
```js
await Layer8DConfig.load()                     // Fetches login.json
Layer8DConfig.getConfig()                      // Full app config
Layer8DConfig.getDateFormat()                  // 'mm/dd/yyyy'
Layer8DConfig.getApiPrefix()                   // '/erp'
Layer8DConfig.resolveEndpoint('/30/Employee')  // '/erp/30/Employee'
```

### Layer8DUtils
```js
Layer8DUtils.escapeHtml(text)
Layer8DUtils.formatDate(timestamp)             // Unix sec -> 'MM/DD/YYYY'
Layer8DUtils.formatDateTime(timestamp)
Layer8DUtils.parseDateToTimestamp(dateString)
Layer8DUtils.formatMoney(cents, currency?)     // 150000 -> '$1,500.00'
Layer8DUtils.formatPercentage(decimal)         // 0.75 -> '75.00%'
Layer8DUtils.formatPhone(digits)
Layer8DUtils.formatSSN(digits, masked?)
Layer8DUtils.formatHours(minutes)              // 150 -> '2:30'
Layer8DUtils.getNestedValue(obj, 'a.b.c')
Layer8DUtils.debounce(fn, ms)
Layer8DUtils.matchEnumValue(input, enumMap)
```

### Layer8DRenderers
```js
Layer8DRenderers.renderEnum(value, enumMap)
Layer8DRenderers.renderBoolean(value)
Layer8DRenderers.renderDate(timestamp)
Layer8DRenderers.renderDateTime(timestamp)
Layer8DRenderers.renderMoney(cents, currency?)
Layer8DRenderers.renderPercentage(decimal)
Layer8DRenderers.renderPhone(digits)
Layer8DRenderers.renderSSN(digits, masked?)
Layer8DRenderers.renderHours(minutes)
Layer8DRenderers.renderRating(value, max?)
Layer8DRenderers.createStatusRenderer(enumMap, classMap) // Returns function
```

### Layer8DPopup
```js
Layer8DPopup.show({
    title, titleHtml, content, size: 'small'|'medium'|'large'|'xlarge',
    showFooter, saveButtonText, cancelButtonText, noPadding,
    onSave: (formData) => {}, onShow: (body) => {}  // onShow fires 50ms after render
});
Layer8DPopup.close()           // Topmost
Layer8DPopup.closeAll()
Layer8DPopup.updateContent(html)
Layer8DPopup.updateTitle(title)
Layer8DPopup.getBody()         // Body element (topmost non-stacked)
```

Tab support: `.probler-popup-tab[data-tab]` + `.probler-popup-tab-pane[data-pane]`.

### Layer8DNotification
```js
Layer8DNotification.success(msg)    // 3000ms
Layer8DNotification.error(msg, details[])  // manual close
Layer8DNotification.warning(msg)    // 5000ms
Layer8DNotification.info(msg)       // 4000ms
```

### Layer8DTable
Constructor takes single options object. **Must call `table.init()` after construction.**

```js
const table = new Layer8DTable({
    containerId, endpoint, modelName, columns, pageSize: 10,
    serverSide: true, primaryKey, sortable: true, filterable: true,
    filterDebounceMs: 1000, transformData: (item) => ({}),
    baseWhereClause, onDataLoaded, onRowClick: (item, id) => {},
    onAdd, onEdit: (id) => {}, onDelete: (id) => {},
    addButtonText, showActions: true, emptyMessage, pageSizeOptions
});
table.init();
```

Methods: `init()`, `setData(array)`, `setServerData(array, total)`, `fetchData(page, size)`, `setBaseWhereClause(str)`, `render()`, `sort(key)`, `goToPage(n)`.

Static: `Layer8DTable.tag(text, cls)`, `.tags(arr, cls)`, `.countBadge(n, singular, plural)`, `.statusTag(bool, up, down)`.

### Layer8DDatePicker
```js
Layer8DDatePicker.attach(input, { minDate, maxDate, onChange, showTodayButton, firstDayOfWeek })
Layer8DDatePicker.setDate(input, timestamp)  // 0 = 'Current'/'N/A'
Layer8DDatePicker.getDate(input)             // Unix timestamp (0=Current, null=empty)
Layer8DDatePicker.detach(input)
```

### Layer8DInputFormatter
Types: `ssn`, `phone`, `currency`, `percentage`, `routingNumber`, `ein`, `email`, `url`, `colorCode`, `rating`, `hours`

```js
Layer8DInputFormatter.attach(input, type, opts)
Layer8DInputFormatter.getValue(input)
Layer8DInputFormatter.setValue(input, value)
Layer8DInputFormatter.validate(input)  // { valid, errors[] }
Layer8DInputFormatter.attachAll(container)
Layer8DInputFormatter.collectValues(container)
Layer8DInputFormatter.format.currency(15000)  // '$150.00'
```

### Layer8DReferencePicker
```js
Layer8DReferencePicker.attach(input, {
    endpoint, modelName, idColumn, displayColumn,  // all REQUIRED
    displayFormat, selectColumns, baseWhereClause, pageSize, onChange, title
});
Layer8DReferencePicker.getValue(input) / .getItem(input) / .setValue(input, id, display, item) / .detach(input)
```

### Layer8DReferenceRegistry
```js
Layer8DReferenceRegistry.register({
    Employee: { idColumn, displayColumn, selectColumns, displayLabel, displayFormat }
});
Layer8DReferenceRegistry.get('Employee')
```

### Layer8DDataSource
Shared fetch layer for all view types. Metadata valid ONLY on page 1.

```js
const ds = new Layer8DDataSource({ endpoint, modelName, columns, pageSize, baseWhereClause, transformData });
ds.fetchData(page)  // 1-indexed
ds.buildQuery(page, pageSize)
ds.setBaseWhereClause(str) / .setFilter(key, val) / .clearFilters() / .setSort(key, dir) / .getTotalPages()
```

### Layer8DForms
Facade for `Layer8DFormsFields`, `Layer8DFormsData`, `Layer8DFormsPickers`, `Layer8DFormsModal`.

```js
Layer8DForms.openAddForm(serviceConfig, formDef, onSuccess)
Layer8DForms.openEditForm(serviceConfig, formDef, recordId, onSuccess)
Layer8DForms.openViewForm(serviceConfig, formDef, data)
Layer8DForms.confirmDelete(serviceConfig, recordId, onSuccess)
// Low-level: generateFormHtml, collectFormData, validateFormData, fetchRecord, saveRecord, deleteRecord, attachDatePickers, attachReferencePickers
```

`serviceConfig`: `{ endpoint, primaryKey, modelName }`

#### FormFactory Presets
```js
f.basicEntity()      // [code, name, description, isActive]
f.dateRange(prefix)   // [startDate, endDate]
f.address(parentKey)  // [line1, line2, city, stateProvince, postalCode, countryCode]
f.contact(parentKey)  // [value, contactType]
f.audit()             // Read-only [createdBy, createdAt, modifiedBy, modifiedAt]
f.person(middle?)     // [firstName, (middleName), lastName]
```

#### FormsFields Extended
Inline tables (`generateInlineTableHtml`), period selector, tags/multiselect, file upload.

### Layer8DToggleTree
```js
const tree = Layer8DToggleTree.create({ container, data, onToggle, dependencies });
tree.getDisabledPaths()
```

### Layer8DModuleFilter
```js
await Layer8DModuleFilter.load(bearerToken)
Layer8DModuleFilter.isEnabled('hcm.payroll')
Layer8DModuleFilter.applyToSidebar() / .applyToSection('hcm')
await Layer8DModuleFilter.save(disabledPaths, bearerToken)
```

### Layer8DViewFactory
```js
Layer8DViewFactory.register(type, factoryFn) / .create(type, options) / .has(type) / .getTypes()
Layer8DViewFactory.detectTitleField(columns, pk)
Layer8DViewFactory.createWithSwitcher(type, options, viewTypes, serviceKey, onSwitch)
```

Types: `table`, `chart`, `kanban`, `timeline`, `calendar`, `gantt`, `tree`, `wizard`. All instances: `init()`, `refresh()`, `destroy()`.

#### Layer8ViewSwitcher (shared desktop/mobile)
```js
Layer8ViewSwitcher.render(serviceKey, viewTypes, activeType)
Layer8ViewSwitcher.attach(container, callback)
```

### Layer8DChart
SVG chart (bar/line/pie). Auto-detects category/value fields.

```js
const chart = new Layer8DChart({
    containerId, columns, dataSource,
    viewConfig: { chartType: 'bar'|'line'|'area'|'pie'|'donut', categoryField, valueField,
                  aggregation: 'count'|'sum'|'avg'|'min'|'max', title, colors, pageSize: 100 },
    onItemClick, onAdd
});
```

Static: `Layer8DChart.readThemeColor(varName, fallback)`, `.getThemePalette()`.

Auto-detection priority: period columns > date+money columns > status/type/category patterns > title field fallback.

Auto-enabled chart view when columns include both `type: 'date'` and `type: 'money'`.

L8Period support: auto-converts `{periodType, periodYear, periodValue}` to labels. Date normalization: timestamps bucketed to year/quarter. Handles numeric strings.

### Layer8DKanban
```js
viewConfig: { laneField, lanes: { 1: { label, color } }, cardTitle, cardSubtitle, cardFields }
```

### Layer8DCalendar
```js
viewConfig: { dateField, titleField, viewMode: 'month'|'week' }
```

### Layer8DTimeline
```js
viewConfig: { dateField, actorField, titleField, descriptionField, colorField, pageSize }
```

### Layer8DGantt
```js
viewConfig: { startDateField, endDateField, progressField, titleField, dependencyField, defaultZoom: 'day'|'week'|'month'|'quarter'|'year' }
```

Auto-detects date fields by key patterns: start/begin/from for start, end/due/until/required/expir for end. Handles numeric string timestamps.

### Layer8DTreeGrid
```js
viewConfig: { parentIdField, idField, labelField, expandedByDefault: true, pageSize: 500 }
```
Methods: `toggleNode(id)`, `expandAll()`, `collapseAll()`.

### Layer8DWizard
```js
viewConfig: { steps: [{ key, label, fields: [] }] }
```

### Layer8DWidget
```js
Layer8DWidget.render({ label, icon, onClick }, value, { trend, trendValue, sparklineData, sparklineColor })
Layer8DWidget.renderEnhancedStatsGrid(kpis, iconMap)
```

### Layer8DModuleFactory
```js
Layer8DModuleFactory.create({
    namespace: 'HCM', defaultModule: 'core-hr', defaultService: 'employees',
    sectionSelector: 'core-hr', initializerName: 'initializeHCM',
    requiredNamespaces: ['CoreHR', 'Payroll']
});
```

#### Layer8ModuleConfigFactory
```js
const svc = Layer8ModuleConfigFactory.service;
const mod = Layer8ModuleConfigFactory.module;
Layer8ModuleConfigFactory.create({
    namespace: 'Bi',
    modules: { 'reporting': mod('Reporting', 'icon', [svc('reports', 'Reports', 'icon', '/35/BiReport', 'BiReport')]) },
    submodules: ['BiReporting']
});
```

## Layer8DTablePaginationMetadata

Server computes metadata (totalCount, key counts) only on page 1. Pages 2+ return zero/stale metadata. Reading metadata on every page overwrites correct totalItems with 0.

```javascript
// CORRECT -- only read metadata on first page
if (page === 1 && data.metadata?.keyCount?.counts) {
    totalCount = data.metadata.keyCount.counts.Total || 0;
    this.totalItems = totalCount;
} else {
    totalCount = this.totalItems;
}
```

`setServerData` must also guard: only update `this.totalItems` if `totalItems > 0`.

### Files requiring this guard

| File | Page var | First page |
|------|----------|-----------|
| `layer8d-table-data.js` | `page` | `1` |
| `layer8d-data-source.js` | `page` | `1` |
| `layer8m-data-source.js` | `page` | `1` |
| `layer8d-reference-picker-data.js` | `state.currentPage` | `0` |

Add any new paginated component to this table.

Actions that reset to page 1 (triggering fresh metadata): filter/sort/pageSize changes, `setBaseWhereClause()`, `init()`.

```bash
grep -n "metadata?.keyCount" l8ui/edit_table/layer8d-table-data.js l8ui/shared/layer8d-data-source.js l8ui/m/js/layer8m-data-source.js l8ui/reference_picker/layer8d-reference-picker-data.js
# Every match MUST be preceded by a page === 1 (or page === 0) check
```

## Layer8MApiReference

### Layer8MConfig
```js
await Layer8MConfig.load()
Layer8MConfig.getConfig()            // raw { login, app }; use config.app.healthPath
Layer8MConfig.resolveEndpoint(path)
Layer8MConfig.registerModules(obj) / .registerReferences(obj) / .getReferenceConfig(model)
```

### Layer8MAuth
```js
Layer8MAuth.requireAuth() / .getUsername() / .logout()
await Layer8MAuth.get(url) / .post(url, data) / .put(url, data) / .delete(url, data?)
```

### Layer8MUtils
Same as desktop utils plus `showSuccess(msg)`, `showError(msg)` toasts.

### Layer8MPopup
```js
Layer8MPopup.show({
    title, content, size: 'small'|'medium'|'large'|'full', showFooter,
    saveButtonText, cancelButtonText, showCancelButton,
    onSave: (popup) => {}, onShow: (popup) => {}, onTabChange: (tabId, popup) => {}
});
Layer8MPopup.close() / .getBody()
```

### Layer8MConfirm
```js
await Layer8MConfirm.show({ title, message, confirmText, cancelText, destructive })
await Layer8MConfirm.confirmDelete(entityName)
```

### Layer8MDatePicker
```js
Layer8MDatePicker.show({ value, minDate, maxDate, title, onSelect: (timestamp, dateStr) => {} })
```

### Layer8MReferencePicker
```js
Layer8MReferencePicker.show({ endpoint, modelName, idColumn, displayColumn, displayFormat, selectColumns, pageSize, currentValue, onChange })
Layer8MReferencePicker.getValue(input) / .setValue(input, id, display, item)
```

### Layer8MRenderers
```js
renderEnum, renderBoolean({trueText,falseText}), renderDate, renderMoney, renderPercentage,
renderPhone, renderSSN, renderHours, renderPeriod, renderRating, renderProgress, renderPriority,
renderEmployeeName, renderMinutes, renderCount, createStatusRenderer
```

### Layer8MForms
```js
Layer8MForms.renderForm(formDef, data, readonly)
Layer8MForms.getFormData(container) / .validateForm(container) / .showErrors(container, errors)
Layer8MForms.initFormFields(container) / .initInlineTableHandlers(container, formDef)
```

Extended fields: currency, percentage, phone, SSN, URL, rating, hours, EIN, routingNumber, colorCode, inlineTable, time, tags, multiselect, richtext.

### Layer8MTable
Constructor: `(containerId, config)`.
```js
new Layer8MTable('id', { endpoint, modelName, columns, rowsPerPage: 15, transformData, statusField, onCardClick, getItemId })
```

### Layer8MEditTable (extends Layer8MTable)
Adds onAdd, onEdit, onDelete (null = hidden/read-only), onRowClick, addButtonText.

### Layer8MNav
```js
Layer8MNav.showHome() / .navigateToModule(key) / .navigateToSubModule(mod, sub)
Layer8MNav.navigateToService(mod, sub, svc) / .navigateBack() / .getCurrentState()
```

Looks up columns/forms/transforms from registered `window.MobileXXX` objects (`getColumns`, `getFormDef`, `getTransformData`).

#### LAYER8M_NAV_CONFIG
```js
{ modules: [{ key, label, icon, hasSubModules }],
  hcm: { subModules: [...], services: { 'core-hr': [{ key, label, icon, endpoint, model, idField, readOnly?, supportedViews? }] } },
  icons: {}, getIcon(key) }
```

### Layer8MModuleRegistry
```js
window.MobileHCM = Layer8MModuleRegistry.create('MobileHCM', { 'Core HR': MobileCoreHR, ... });
// Provides: getColumns, getFormDef, getEnums, getPrimaryKey, getRender, hasModel, getAllModels, getModuleName
```

### Layer8MViewFactory
Same interface as desktop. Types: table, chart, kanban, calendar, timeline, gantt, tree, wizard.

Auto-detect chart when columns have both `type: 'date'` and `type: 'money'`.

### Layer8MDataSource
```js
new Layer8MDataSource({ endpoint, modelName, columns, pageSize: 15, baseWhereClause, transformData, onDataLoaded, onError, onMetadata })
// fetchData(page), buildQuery, setBaseWhereClause, setFilter, clearFilters, setSort, getTotalPages
```

### Extensibility
- `Layer8MReferenceRegistry.register(registryObj)` -- project-specific reference configs
- `Layer8SvgFactory.registerTemplate(key, fn)` -- project-specific SVG illustrations

## LogServicesRequired

Every project MUST include `log-vnet` and `log-agent` as separate binaries.

| Service | Directory | Image | Base | K8s Kind |
|---------|-----------|-------|------|----------|
| log-vnet | `go/<project>/log-vnet/` | `saichler/<project>-log-vnet` | `<project>-security` | DaemonSet (hostNetwork) |
| log-agent | `go/<project>/log-agent/` | `saichler/<project>-log-agent` | `<project>-security` | DaemonSet |

Each needs: `main.go`, `Dockerfile`, `build.sh`. Copy from `../probler/go/prob/log-vnet/` and `log-agent/`.

Checklist: both in `build-all-images.sh`, all four K8s YAMLs, `deploy.sh`/`undeploy.sh`, `kind-start.sh`. Dockerfiles use `saichler/<project>-security:latest`. log-agent LOGPATH = `/data/logs/<project>`.

## LoginJsonAdaptation

When copying `login.json` from l8erp, update immediately:

| Field | l8erp Default | Change To |
|-------|--------------|-----------|
| `login.appTitle` | `"ERP by Layer 8"` | Project name |
| `login.appDescription` | `"Enterprise Resource Planning"` | Project description |
| `app.apiPrefix` | `"/erp"` | Project's PREFIX (`grep "PREFIX" go/<project>/common/defaults.go`) |

Leaving `apiPrefix` as `/erp` causes 404 on every API call.

```bash
grep "apiPrefix" go/<project>/ui/web/login.json  # Must NOT contain "/erp"
```

## LoginableEntityUserProvisioning

Entities representing loginable persons MUST provision a user via Security API with a `Portal` field. Provisioning MUST happen in the UI layer (JavaScript), NEVER in Go ServiceCallback (never import `l8secure`).

### Required user fields
`userId` (entity PK), `email`, `fullName`, `accountStatus: 'ACCOUNT_STATUS_ACTIVE'`, `portal: '<subdir>/app.html'`, `password: { hash: '<default>' }`, `roles: { '<role>': true }`

### Pattern
1. Create `<project>-user-provisioning.js` -- POST to `/73/users` via `Layer8DConfig.resolveEndpoint`
2. Wire into init file by overriding `_openAddModal` for the loginable entity model
3. Include script in app.html before init JS

### Identifying loginable entities
Has `email` + intended for external users, PRD describes a portal, has `userId`/`password`/`accountStatus` fields, or PRD mentions login/self-registration.

Mock data generators must create users explicitly via `map[string]interface{}` posted to `/project/73/users`.

Canonical reference: l8physio `physio-user-provisioning.js` and `physio-init.js` lines 73-119.

## MainPackageMinimal

`package main` must contain ONLY entry point logic. All business logic in dedicated packages.

**Belongs in main**: resource/config creation, vnic init, calling `Activate` functions, DB start, signal wait.

**Does NOT belong**: structs with methods, data processing, service callbacks, helpers, non-trivial constants.

```go
// CORRECT
func main() {
    res := common.CreateResources("engine")
    nic := startVnic(res)
    engine.NewEngine(nic, res).Run()
    common.WaitForSignal(res)
}
```

## Maintainability

### File Size
Max 500 lines per file. Proactively split at 450. Stop and refactor if exceeded.

### ServiceName / ServiceArea
ServiceName max 10 characters. ServiceArea same for all services in a module.

### ServiceCallback Auto-Generate ID
Every `*ServiceCallback.go` must auto-generate PK on POST: `common.GenerateID(&entity.PrimaryKeyField)` in `Before()`, guarded by `if action == ifs.POST`, between type assertion and `validate()`.

### UI Type Registration
Register model types in `go/erp/ui/main.go` using `introspect.AddPrimaryKeyDecorator` + `registry.Register`.

### Read Before Implementing
Always read ALL code of referenced components before implementing similar ones.

### Duplication Prevention
- **Second Instance Rule**: Extract shared abstraction immediately on second instance, never copy+replace
- **Copy-Paste Detection**: If diff shows only namespace/identifier changes, extract shared component
- **Config vs Logic**: Module files = data only (config, enums, columns, forms). All behavior in shared components
- **New Module Checklist**: If module needs its own nav/CRUD/form/service-lookup logic, fix the shared component instead
- **Audit**: >80% similarity across modules = mandatory refactoring candidate
- **Facades**: 100% delegation wrapper = dead weight, delete it
- **Backport**: Improvements in newer modules must be applied to all existing modules
- **Boilerplate ceiling**: >50 lines of structural code for a new module = shared layer needs improvement

## MobileRules

### Anti-Patterns
Do NOT: hardcode sidebar `<a>` tags, custom headers, skip `Layer8MModuleRegistry`/dynamic nav, invent new app init flow. Follow l8erp patterns.

### Desktop/Mobile Functional Parity
Every feature must work on both platforms. When touching any section, audit entire section for parity gaps. A feature includes all downstream effects (e.g., module disable must hide nav on both platforms).

Verification: detail popups on both? interactive features on both? shared scripts in both app.html files? no placeholder sections on one side while functional on the other?

## MockDataRules

### Location and Process
All files in `go/tests/mocks/`. Phased, dependency-ordered.

1. Read protobuf files -- exact field names, cross-module refs
2. Determine phase ordering (foundation first, 5-10 phases)
3. Add name arrays to `data.go`
4. Add ID slices to `store.go` (module-prefix names that could collide)
5. Create generator files (`gen_<module>_<group>.go`, <500 lines each)
6. Create phase files (`<module>_phases.go`)
7. Update `main.go`
8. `go build ./tests/mocks/` and `go vet`

### Endpoint Construction
Format: `/erp/{ServiceArea}/{ServiceName}`. Use exact `ServiceName` constant from `*Service.go` (max 10 chars, often abbreviated).

```bash
grep "ServiceName = " go/erp/{module}/**/*Service.go
```

### Cross-Module Phase Order
FIN/HCM circular dependency resolved by splitting FIN:
1. FIN Foundation (1-3) -- CurrencyIDs, Vendors, Customers
2. HCM (all) -- EmployeeIDs, DepartmentIDs
3. FIN Remaining (4-9) -- needs DepartmentIDs, EmployeeIDs
4. SCM, Sales, MFG, CRM, PRJ, BI, DOC, ECOM, COMP

`pickRef(store.XxxIDs, index)` silently returns `""` when slice is empty. Before adding `ValidateRequired` for a FK, verify the referenced module runs first in `main_phases.go`.

| Field | Source Module | Used By |
|-------|-------------|---------|
| CurrencyIDs | FIN Phase 1 | ALL |
| EmployeeIDs | HCM Phase 1-3 | FIN 8, CRM, PRJ, MFG, Sales |
| DepartmentIDs | HCM Phase 1 | FIN 4+8, PRJ, MFG |
| VendorIDs | FIN Phase 2 | SCM, MFG |
| CustomerIDs | FIN Phase 2 | Sales, ECOM, CRM |
| ItemIDs | SCM Phase 1 | Sales, MFG, ECOM |

### Key Patterns
- Flavorable distributions: first 60% APPROVED, next 20% IN_PROGRESS, rest cycle
- Money in cents: `int64(rand.Intn(rangeSize) + minimum)`
- IDs: `fmt.Sprintf("<prefix>-%03d", i+1)`
- Always `createAuditInfo()` for audit fields
- Check `len(store.*IDs) > 0` for optional cross-module refs

## ModconfigFailureNoLogout

When copying `app.js` from l8erp, `Layer8DModuleFilter.load()` fetches `/0/ModConfig` -- an l8erp-specific service. On 404, it internally calls `logout()` causing infinite redirect loop. try/catch does NOT help (logout fires before promise rejects).

Fix: Remove the block if project has no ModConfig service. Also check for other l8erp-specific fetches: currency cache (`/erp/40/Currency`), exchange rates (`/erp/40/XchgRate`), any `/erp/` prefix endpoints.

## ModuleInitSectionSelector

In `Layer8DModuleFactory.create()`, `sectionSelector` MUST equal `defaultModule`. Navigation searches for `[data-module="${sectionSelector}"]` which matches submodule names, not section names.

```javascript
// CORRECT
Layer8DModuleFactory.create({
    defaultModule: 'planning',
    sectionSelector: 'planning',    // must match defaultModule
    ...
});
```

Error: `<Module> section container not found`.

## MoneyFieldTypeMapping

Before using form factory methods, check protobuf type. Fields with Go type starting `*`, `[]`, or `map[` are NOT scalars -- using `f.text()`/`f.number()` shows `[object Object]`.

```bash
grep -A 30 "type ModelName struct" go/types/<module>/*.pb.go
```

| Go Type | Form Factory |
|---------|-------------|
| `string`, `int32`, `float64`, `bool` | `f.text()`, `f.number()`, `f.checkbox()` |
| `*erp.Money` | `f.money()` |
| `*erp.Address` | `...f.address(prefix)` |
| `*erp.ContactInfo` | `...f.contact(prefix)` |
| `*erp.DateRange` | Two `f.date()` calls |
| `*erp.AuditInfo` | `...f.audit()` |
| Any new `*erp.X` | Create handler first in form-factory, forms-fields, forms-data |

Symptoms: `[object Object]` in field, empty on edit, save corrupts data.

## NeverActOnQuestions

When user asks a question, ONLY answer it. Do NOT edit files, modify plans, or change code. Wait for explicit approval before taking action.

## NoGoGenerics

Do NOT use Go generics (type parameters). Use interfaces, concrete types, or `interface{}`/`any`.

```go
// WRONG
func Filter[T any](items []T, fn func(T) bool) []T { ... }
// CORRECT
func FilterEmployees(items []*Employee, fn func(*Employee) bool) []*Employee { ... }
```

## PlanRequirements

### Approval Workflow
Write plans to `./plans/<name>.md`. Stop and wait for explicit user approval. Do not implement, do not ask "should I proceed?".

### Duplication Audit
Before writing to `./plans/`, audit for duplicate behavioral code. If 2+ files share the same logic (differing only in config values), plan MUST include extraction Phase 0.

Categorize every line as behavioral (auth, nav, CRUD, forms, data fetching, DOM, CSS) vs configuration (namespace, columns, forms, enums, PKs, nav items). If `behavioral_lines * instances > 100`, extraction mandatory. Phase 0 refactors the original pattern first.

### Platform Completeness
Plans covering only one platform when multiple exist are incomplete. Audit must be Component x Platform. Traceability matrix must include Platform column. Verification must cover all platforms.

### Traceability and Verification
Every plan MUST include:
1. **Traceability matrix** -- maps every gap/action item to a phase. Orphans = planning errors.
2. **Final verification phase** -- smoke-test every affected section E2E (navigate, data loads, row click, forms submit).

```markdown
| # | Section | Gap / Action Item | Platform | Phase |
|---|---------|-------------------|----------|-------|
```

Rule compliance must systematically check ALL applicable rules, not cherry-pick.

## PlatformConversionDataFlow

When converting between platforms, trace data flow end-to-end before writing code.

### 5-Step Protocol

**Step 0: Feature inventory** -- enumerate EVERY interactive element (table chrome, CRUD, state management, nav, supplementary features). Missing inventory items will not be ported.

**Step 1: Trace source data flow** -- for each interaction: where does data come from? how does it reach the handler? what transforms apply?

**Step 2: Map to target equivalents** -- does target framework provide the same data? If yes, use directly (no extra server calls). If no, document the gap.

**Step 3: Trace every link in target pipeline** -- verify each intermediate layer (factory, adapter, registry) forwards data. Catches "dropped parameter" bugs where A has data, C needs it, but B doesn't forward it.

**Step 4: Verify parity** -- same data source type? same server call count? same data types? Click through actual UI.

### Hidden Container Rendering
Components in hidden tabs (charts, canvas, maps) need deferred initialization. Render on tab activation, not popup open. Container has 0x0 dimensions when hidden.

### Never Bypass Existing Abstractions
Extend wrappers to support new features. Never replace `helper(args)` with `underlying.api(reconstructedArgs)`.

### Parity Plans Must Trace Data Transforms
Field parity comparisons must verify value TYPE, not just name. Add a "Value Type Match?" column. Common mismatches: enum int vs string label, unix timestamp vs formatted date, nested object vs flat string.

| Raw Server Type | Common Transform | Breaks When |
|----------------|-----------------|-------------|
| Enum integer | String label | `.toUpperCase()`, display |
| Unix timestamp | Formatted date | `.includes()`, display |
| Nested object | Flat string | String methods |
| Boolean | Label | `.toUpperCase()` |

### Common Traps
- Adding features not in source platform
- Replicating untested patterns across files
- Checking structure not behavior
- Assuming transport parity (`Auth.patch()` may not exist)
- Rendering into hidden containers
- Bypassing existing wrappers

## PortalsSameWebServer

One web server process per project. Portals (different UI for different audience) are subdirectories under `go/<project>/ui/web/`, never separate binaries/Docker images/K8s deployments.

```
go/<project>/ui/web/
├── app.html              # Admin/main portal
├── member/               # Member portal — subdirectory
│   ├── app.html
│   └── js/
├── ess/                  # ESS portal — subdirectory
│   └── app.html
└── l8ui/
```

All portals served by single `go/<project>/ui/main.go`. Type registrations for all portals go in that single `main.go`. Two web servers with `hostNetwork: true` on same node bind same port and crash.

## PrdCompliance

All PRDs must comply with all rules at `../l8book/rules`, follow l8erp architecture, and include a detailed compliance checklist.

### Compliance Checklist

**Project Structure**: follows l8erp layout, directory/file naming matches l8erp.

**Protobuf**: enum zero=UNSPECIFIED, list types use `repeated X list = 1`, no direct struct refs between Prime Objects (ID only), children are embedded `repeated` fields.

**Service**: ServiceName <= 10 chars, ServiceArea consistent within module, ServiceCallback auto-generates PK on POST, types registered in UI main.go.

**UI**: all module integration steps planned, desktop/mobile parity, immutable entities have read-only UI, child types use inline tables, components follow l8ui guide at `../l8ui/rules/`.

**Mock Data**: all services have generators, phase ordering accounts for cross-module deps.

**Deployment**: build.sh + Dockerfile + K8s YAMLs (all 4 modes: local/baremetal/GKE/KIND) + KIND scripts + run-local.sh.

**Configuration**: login.json adapted, ModConfig handling addressed.

### Architecture (follow l8erp)

```
go/
├── <module>/
│   ├── common/                      # PREFIX, defaults
│   ├── <submodule>/
│   │   ├── <entity>Service.go       # ServiceName, ServiceArea
│   │   └── <entity>ServiceCallback.go
│   ├── ui/
│   │   ├── main.go                  # UI server + type registration
│   │   └── web/                     # app.html, login.html, login.json, l8ui/, sections/, m/
│   ├── main/                        # Backend main.go
│   └── vnet/                        # Vnet main.go
├── types/<module>/                  # Generated .pb.go
├── tests/mocks/                     # Mock data
proto/
├── make-bindings.sh
├── <module>.proto
```

**Service pattern**: one service per Prime Object, ServiceCallback with Before/After hooks, children embedded as `repeated` fields.

**UI pattern**: config + enums + columns + forms + init per submodule, init calls `Layer8DModuleFactory.create()`.

### UI Rules

PRD UI sections must use l8ui components: Layer8DTable/Layer8MTable for tables, form framework for forms, Layer8DModuleFactory for nav, registered view types for kanban/chart/timeline/etc., `--layer8d-*` CSS tokens for theming, Layer8M* for mobile. Rules at `../l8ui/rules/`.

## PrdL8uiIncludesAudit

Every UI PRD MUST contain an "L8UI Includes Audit" section listing every CSS and JS file from `desktop-script-loading-order.md` and `mobile-script-loading-order.md`, each marked included or N/A with reason. PRD without this section MUST NOT be written to `./plans/`.

### Desktop CSS (36 files)
layer8d-theme-tokens, layer8d-theme, layer8d-animations, layer8d-scrollbar, layer8-print, layer8-section-layout, layer8-section-responsive, layer8d-table, layer8d-chart, layer8d-kanban, layer8d-timeline, layer8d-calendar, layer8d-tree-grid, layer8d-gantt, layer8d-wizard, layer8d-widget, layer8-view-switcher, layer8-markdown, l8agent-chat, l8agent-bubble, l8sys, l8health, l8sys-modules, l8logs, l8dataimport, layer8d-toggle-tree, layer8d-popup, layer8d-popup-forms, layer8d-form-fields, layer8-file-upload, layer8d-popup-inline-table, layer8d-popup-content, layer8d-datepicker, layer8d-reference-picker, layer8d-input-formatter, layer8d-notification.

### Desktop JS (97 files)
Full list includes: theme-switcher, config, websocket, utils, renderers, reference-registry, portal-switcher, enum-factory, ref-factory, column-factory, form-factory, form-factory-presets, svg-factory, module-config-factory, section-generator, notification, 6 input-formatter files, format-display, forms-fields, forms-fields-ext, forms-data, forms-pickers, forms-modal, forms, popup, 4 datepicker files, 6 reference-picker files, 6 table files, csv-export, excel-export, pdf-export, export-helper, file-upload, data-source, view-factory, view-switcher, 3 chart files, 3 kanban files, timeline, 3 calendar files, 3 tree-grid files, 3 gantt files, 2 wizard files, widget, service-registry, module-crud, module-navigation, toggle-tree, module-filter, permission-filter, module-factory-core, module-factory, markdown, 5 l8agent files, l8sys-config, l8health, 5 l8security files, 2 l8security-events files, l8sys-dependency-graph, l8sys-modules-map, l8sys-modules, l8logs, 4 l8dataimport files, l8sys-init.

### Mobile
Same audit for all mobile CSS/JS from mobile-script-loading-order.md.

First implementation phase must create `app.html` and `m/app.html` with ALL checked files included.

## PrimeObjectReferences

### Rule 1: What IS a Prime Object

Entity that can exist independently with its own identity and lifecycle. Gets its own service directory. ALL four tests must pass:
1. **Independence**: exists without a parent
2. **Own lifecycle**: CRUD independent of parent
3. **Direct query need**: users query it across all parents
4. **No parent ID dependency**: identity stands alone

**NOT Prime Objects** (embed as `repeated` fields): line items, entries/details, components/operations, assignments/members, child records, config children, status/history records. Key indicator: has a required `parent_id` field.

```protobuf
message SalesOrder {
    string sales_order_id = 1;
    repeated SalesOrderLine lines = 20;      // Embedded child
}
message SalesOrderLine {
    string line_id = 1;
    string item_id = 2;    // Ref to Prime Object by ID
}
```

### Rule 2: Cross-References Between Prime Objects

Prime Objects MUST NEVER contain `*OtherPrimeType` or `[]*OtherPrimeType`. Reference other Prime Objects ONLY via string ID fields. Direct struct refs cause "Decorator Not Found" errors because introspector creates duplicate nodes.

Allowed: shared/common types (`erp.Money`, `erp.Address`, `erp.AuditInfo`), embedded child types, string ID fields.

### Rule 3: UI Implications

**Prime Object** gets: config entry, columns, forms, nav entry, type registration, reference registry entry, mock data generator with ID slice.

**Child Type** gets NONE of the above -- appears only as `f.inlineTable()` within parent's form.

```javascript
// CORRECT: child as inline table in parent form
f.section('Order Lines', [
    ...f.inlineTable('lines', 'Order Lines', [
        { key: 'lineId', label: 'Line ID', hidden: true },
        { key: 'itemId', label: 'Item', type: 'reference', lookupModel: 'ScmItem' },
    ]),
])
```

## ProtobufRules

### Model Names Must Match Protobuf Types

Use protobuf type name everywhere (L8Query, JS config, forms, columns, nav), NOT ServiceName.

| ServiceName | Protobuf Type (use this) |
|---|---|
| `Sprint` | `BugsSprint` |
| `Territory` | `SalesTerritory` |
| `DlvryOrder` | `ScmDeliveryOrder` |
| `MfgWorkOrd` | `MfgWorkOrder` |
| `ImprtTmpl` | `L8ImportTemplate` |

Never omit module prefix in JS: `SalesReturnOrder` not `ReturnOrder`, `ScmWarehouse` not `Warehouse`.

```bash
grep "type Sales" go/types/sales/*.pb.go | grep "struct {"
```

Error: `Cannot find node for table <wrong-name>`, HTTP 400.

### Enum Zero Value

Every enum MUST have `[PREFIX_]FIELD_NAME_UNSPECIFIED = 0`. Zero must NOT be a valid state.

```protobuf
enum AccountType {
  ACCOUNT_TYPE_UNSPECIFIED = 0;
  ACCOUNT_TYPE_ASSET = 1;
}
```

```bash
grep -A1 "^enum " proto/*.proto | grep "= 0" | grep -iv "unspecified\|invalid\|unknown"
```

### List Type Convention

```protobuf
message SomeEntityList {
  repeated SomeEntity list = 1;       // MUST be named "list"
  l8api.L8MetaData metadata = 2;      // MUST be field 2
}
```

### Protobuf Generation

Run `cd proto && ./make-bindings.sh` after ANY `.proto` change. NEVER compile individual proto files manually. Before running, ensure `docker run` uses `-i` not `-it`. After running, verify `.pb.go` files exist and `go build ./...` passes.

```bash
grep -A 30 "type TypeName struct" go/types/<module>/*.pb.go | grep 'json:"'
```

## ReferenceRegistryCompleteness

Every `lookupModel` in forms MUST have a reference registry entry. Missing entries cause silent failures or console warnings.

```bash
# Find missing registrations
grep -rh "lookupModel: '" go/<project>/ui/web --include="*-forms.js" | \
  sed "s/.*lookupModel: '\\([^']*\\)'.*/\\1/" | sort -u > /tmp/used.txt
grep -rhoE "(simple|coded|batch|batchIdOnly|idOnly|person)\\(['\"][^'\"]+['\"]" \
  go/<project>/ui/web/js/reference-registry-*.js | \
  sed "s/.*(['\"]\\([^'\"]*\\)['\"].*/\\1/" | sort -u > /tmp/registered.txt
comm -23 /tmp/used.txt /tmp/registered.txt
```

Checklist: create desktop `reference-registry-<module>.js`, mobile `layer8m-reference-registry-<module>.js`, add to both `app.html` files, register all lookupModels.

Error: `Reference input missing required config: fieldName`

## RegistrationPage

Standalone registration at `register/index.html`. POST `/captcha` for CAPTCHA image (base64 PNG). POST `/register` with `{user, pass, captcha}`. Redirects to login on success.

## ReportInfraBugs

Do NOT work around infrastructure bugs. Report: what fails, where the bug is, expected vs actual behavior, impact.

### No Silent Fallbacks in UI Code

Required values missing must fail visibly (`console.error`), NOT silently return defaults.

```javascript
// WRONG
return item.id || item.Id || '';  // silently returns '' for every item

// CORRECT
console.error(`Cannot resolve item ID for model "${this.config.modelName}".`);
return undefined;
```

Applies to: ID resolution, config lookups, data transforms, endpoint resolution. Does NOT apply to: optional display defaults (`item.name || '-'`), user-facing empty states, documented optional config defaults.

## ReuseExistingModuleForms

When building a portal/view showing data from an existing module, NEVER redefine forms/enums/renderers. Include the existing module's JS files and use them directly via `Layer8DFormsModal.openViewForm()`, `Layer8DServiceRegistry`, or module namespace (e.g., `Payroll.forms.Payslip`).

## RunLocalScript

Every project with fully implemented PRD MUST include `go/run-local.sh`. Copy from `l8erp/go/run-local.sh` and adapt: binary names, package paths, web asset paths, mock data path, ports, credentials, database name.

The script must:
1. Clean and fetch dependencies
2. Start infrastructure (DB container)
3. Build all binaries into `demo/`
4. Copy web assets
5. Generate `kill_demo.sh`
6. Start services in order (vnet first)
7. Upload mock data
8. Wait for user, then clean up

PRDs must include a "Local Development Setup" section.

## ScriptLoadingOrder

### Desktop

CSS first (theme, section layout, components, view system, markdown, file upload, SYS module, AI agent), then JS in strict dependency order:

1. App shell: `sections.js`, `app.js`
2. Core: `layer8d-config`, `utils`, `renderers`, `reference-registry`
3. Factories: enum, ref, column, form, form-presets, svg, module-config, section-generator
4. Reference registries (project-specific)
5. Notification
6. Input formatters (6 files in order)
7. Markdown, file upload, CSV export
8. Forms (fields, fields-ext, data, pickers, modal, forms facade)
9. Popup
10. Date picker (4 files)
11. Reference picker (6 files)
12. Table (6 files)
13. View system: view-factory, view-switcher, data-source, chart (3), kanban (2), timeline, calendar (2), gantt (2), tree-grid, wizard (2), widget
14. Module abstractions: service-registry, module-crud, module-navigation, toggle-tree, module-filter, module-factory-core, module-factory
15. Module data (per module: section-config, config, enums, columns, forms, init)
16. SYS module (config, health, security files, modules, logs, dataimport, init)
17. AI agent (enums, columns, forms, chat)

### Mobile

CSS (12 files), then JS:
1. Mobile core: config, portal-switcher, auth, utils
2. Notification, input-formatter
3. Desktop shared: config, utils, renderers, reference-registry
4. Factories: enum, ref, column, form
5. Module registry factory
6. Mobile components: popup, confirm, table, edit-table, forms (5 files), datepicker, reference-registry
7. Project-specific reference registries
8. Reference picker, renderers, data-source
9. Shared: markdown, file-upload, csv-export
10. Module data
11. Toggle tree, module filter, SYS modules
12. Nav core (crud, data, nav) BEFORE nav configs
13. Nav configs (project-specific) AFTER nav core
14. Mobile views: view-factory, view-switcher, chart, kanban, calendar, timeline, gantt, tree-grid, wizard
15. Widget, theme-switcher
16. AI agent mobile
17. `app-core.js`

## SecurityConfigStructure

Security config JSON at `go/secure/plugin/<project>/<project>.json` defines users, roles, credentials, sysconfig. Do NOT implement custom data filtering in ServiceCallbacks.

```json
{
  "credentials": { "postgres": { "aside": "dbuser", "yside": "5432", "zside": "dbpwd" } },
  "key": "<AES key>", "secret": "<shared secret>",
  "roles": { "<role>": { "rules": { ... } } },
  "users": { "<user>": { "userName": "x", "password": "x", "roles": { "admin": true } } },
  "sysconfig": { "dataStoreType": 1, "dataStoreName": "mydb", "webPort": 2773 }
}
```

### Allow Rules (action-level)
```json
{ "ruleId": "x", "elemType": "Appointment", "allowed": true,
  "actions": { "-999": true }, "attributes": { "*": "*" } }
```
Actions: `-999`=all, `1`=POST, `2`=PUT, `3`=PATCH, `4`=DELETE, `5`=GET.

### Deny Rules (row-level scoping)
```json
{ "ruleId": "x", "elemType": "Appointment", "allowed": false, "actions": {},
  "attributes": {
    "Appointment": "select * from Appointment where clientId!=${userId}",
    "appointment.therapistnotes": ""
  }
}
```
- Keys WITHOUT dots = row-level L8Query filter (matching rows DENIED/removed)
- Keys WITH dots = field-level denial (field blanked)
- `${userId}` = only supported placeholder, substituted at runtime
- Deny = negative filter: `where buyerId!=${userId}` means user sees only their own rows

Pipeline: JSON -> protojson.Unmarshal -> AES encrypt -> .sec file -> runtime decrypt -> SecurityProvider.init() -> ScopeView() filters GET results.

Canonical reference: `l8secure/go/secure/plugin/phy/phy.json`.

## SecurityRules

### Security Provider Interface

All AAA MUST go through `ifs.ISecurityProvider`. No custom auth middleware, no hardcoded permission checks, no independent session management.

### Provisioning: Only Config JSON or Security API

Users/roles/credentials provisioned ONLY via:
1. Security config JSON consumed by `ISecurityProvider` at startup
2. Security API (same endpoints l8ui Security section uses, service area 73)

Never create a project-owned users service. Mock generators seed business data via Security API, not project-specific endpoints.

```bash
grep -rn 'Post.*"/[^"]*/users"' go/tests/ go/*/main/ go/*/ui/ 2>/dev/null
# Must target service area 73 (shared Security API)
```

### Never Import l8secure

No project may import ANY package from `github.com/saichler/l8secure/`. Use `map[string]interface{}` and POST to Security API endpoint (service area 73).

```go
// CORRECT
userData := map[string]interface{}{
    "userId": entity.EntityId, "email": entity.Email,
    "accountStatus": "ACCOUNT_STATUS_ACTIVE", "portal": "member/app.html",
    "password": map[string]interface{}{"hash": "defaultpassword"},
    "roles": map[string]bool{"member": true},
}
_, err := l8c.PostEntity("users", 73, userData, vnic)
```

```bash
grep -rn "l8secure" go/ --include="*.go" | grep -v vendor/  # Must return ZERO
```

## SetupConfiguration

### login.json
```json
{
  "login": { "appTitle": "My App", "authEndpoint": "/auth", "redirectUrl": "/app.html",
             "sessionTimeout": 30, "tfaEnabled": true },
  "app": { "dateFormat": "mm/dd/yyyy", "apiPrefix": "/erp", "healthPath": "/0/Health" }
}
```
`resolveEndpoint(path)` prepends `apiPrefix`: `/30/Employee` -> `/erp/30/Employee`.

### L8Query
```
select * from Employee where lastName=Smith limit 10 page 0 sort-by lastName
select employeeId,lastName from Employee where departmentId=D001 limit 15 page 2
```

### Authentication
Bearer token in `sessionStorage.bearerToken`. Desktop: `getAuthHeaders()`. Mobile: `Layer8MAuth.get/post/put/delete()`.

## SharedComponentsReference

### Layer8EnumFactory
```js
factory.create([['Unspecified', null, ''], ['Active', 'active', 'layer8d-status-active']])
// -> { enum: {0:'Unspecified',1:'Active'}, values: {'active':1}, classes: {1:'layer8d-status-active'} }
factory.simple(['Unspecified', 'Type A', 'Type B'])  // -> { enum }
factory.withValues([['Full-Time', 'full-time']])     // -> { enum, values }
```

### Layer8RefFactory
```js
ref.simple('Model', 'modelId', 'name', 'Label')
ref.person('Person', 'personId', 'lastName', 'firstName')
ref.coded('Entity', 'entityId', 'code', 'name')
ref.idOnly('LineItem', 'lineId')
```

### Layer8ColumnFactory
```js
col.id('modelId'), col.col('field', 'Label'), col.boolean('isActive', 'Active'),
col.date('createdDate', 'Created'), col.money('amount', 'Amount'),
col.status('status', 'Status', enums.STATUS_VALUES, render.status),
col.enum('type', 'Type', null, render.type),
col.custom('key', 'Label', (item) => item.x, { sortKey: 'key' })
```

### Layer8FormFactory
```js
f.form('Model', [f.section('Info', [
    ...f.text('code', 'Code', true), ...f.textarea('desc', 'Desc'),
    ...f.select('status', 'Status', enums.STATUS, true),
    ...f.reference('managerId', 'Manager', 'Employee'),
    ...f.date('startDate', 'Start'), ...f.money('amount', 'Amount'),
    ...f.checkbox('isActive', 'Active'), ...f.number('qty', 'Qty')
])])
```

### Column Schema
Desktop: `{ key, label, sortKey, filterKey, enumValues, render }`. Mobile adds: `primary`, `secondary`, `hidden`.

### Form Field Types
`text`, `email`, `tel`, `number`, `textarea`, `date`, `datetime`, `select`, `checkbox`, `currency`, `percentage`, `phone`, `ssn`, `reference`, `url`, `rating`, `hours`, `ein`, `routingNumber`, `colorCode`, `period`, `file`

`readOnly: true` renders display-only span, skipped during data collection.

### Data Collection

| Type | Stored Value |
|---|---|
| currency | Cents (integer) |
| percentage | Decimal (0.75) |
| hours | Total minutes |
| date | Unix timestamp (0=Current) |
| reference | ID value |
| checkbox | 1 or 0 |
| period | `{periodType, periodYear, periodValue}` |

### Layer8Markdown
`Layer8Markdown.render(text)`, `Layer8Markdown.renderInto(element, text)`

### Layer8FileUpload
`Layer8FileUpload.upload(file, docId, version)` -> `{ storagePath, fileName, fileSize, mimeType, checksum }`. `Layer8FileUpload.download(storagePath, fileName)`. Max 5MB. Form: `f.file(key, label, required)`.

### Layer8CsvExport
`Layer8CsvExport.export({ modelName, serviceName, serviceArea, filename })`. Backend: `POST /erp/0/CsvExport`. Export button auto-appears in pagination bars.

## SingleDisplayFormatterRule

All read-only display logic for field types MUST live in ONE place: `formatFieldDisplayValue()` in `layer8d-forms-fields.js`. Both `isReadOnly` block and `formatInlineTableCell` delegate to it.

When adding a new field type:
1. Add editable case to `switch` in `generateFieldHtml`
2. Add display case to `formatFieldDisplayValue`
3. Done. No other files need updating.

```javascript
// WRONG: display logic in isReadOnly block
// CORRECT: add to formatFieldDisplayValue only
function formatFieldDisplayValue(field, value) {
    switch (field.type) { case 'newType': return formatNewType(value); }
}
```

## SingleOwnerDatabaseTable

Only ONE process may activate the ORM service for a given Prime Object. Dual activation causes silently divergent caches.

1. **One owner**: ORM service activated in exactly one process
2. **Remote access**: other processes use vnic/vnet RPC
3. **No local shortcuts**: never activate local ORM for a table you don't own

```go
// WRONG: Process B also activates PhyClient
services.ActivateAllServices(...)  // B's cache diverges from A

// CORRECT: Process B calls via vnet
vnic.Post("PhyClient", 50, clientData)
vnic.Get("PhyClient", 50, query)
```

## SpecialCases

### Read-Only Services
```js
{ key: 'health', endpoint: '/0/Health', model: 'L8Health', idField: 'service', readOnly: true }
```

### Transform Data
```js
MobileMyModule.transformData = function(item) {
    return { displayField: item.rawField || 'Unknown' };
};
```

### Custom CRUD Handlers (Desktop)

Override factory CRUD in init file. `Layer8DModuleCRUD` has NO static `_openAddModal`/etc. -- those exist only on the module namespace after `attach()`. Capture the namespace's own method BEFORE overwriting.

```js
var origInit = window.initializeMyModule;
window.initializeMyModule = function() {
    if (origInit) origInit();
    var origOpenAdd = MyModule._openAddModal;
    MyModule._openAddModal = function(service) {
        if (service.model === 'SpecialModel') {
            MyCustomCRUD.openAdd(service);
        } else {
            origOpenAdd.call(MyModule, service);  // NOT Layer8DModuleCRUD._openAddModal
        }
    };
};
```

## StackedPopupDomScoping

Never use `document.getElementById()` inside a popup. Scope lookups to the active popup's body because stacked popups have duplicate IDs in the DOM.

```javascript
// CORRECT
let form = null;
if (typeof Layer8DPopup !== 'undefined') {
    const body = Layer8DPopup.getBody(); // topmost popup body
    if (body) form = body.querySelector('#layer8d-edit-form');
}
if (!form) form = document.getElementById('layer8d-edit-form'); // fallback
```

- Desktop: `Layer8DPopup.getBody()` returns topmost popup body
- Mobile: `popup.body` passed directly to callbacks, already scoped

Symptom: editing child row saves blank/wrong data. Silent -- no console errors.

## TemplateLiteralTernaryEdits

When wrapping content in `${condition ? \`...\` : ''}` inside a template literal, the edit MUST cover BOTH the opening AND closing of the new nesting level.

```javascript
// WRONG: partial edit only covers opening -- ternary never closed
${items.length > 0 ? `
<div class="section">...

// CORRECT: edit includes closing tags so ternary is properly closed
${items.length > 0 ? `
<div class="section">...</div>
` : ''}
```

After ANY such edit: run `node -c <file>`, count `${`/`}` matches, verify every `? \`` has `` \` : ''}`.

Symptom: functions undefined, file silently fails to parse.

## TestDataFieldVerification

Every key in `map[string]interface{}` for HTTP POST/PUT MUST exist on the target protobuf struct. Unknown fields cause `400 Bad Request` with `Cannot find pb for method POST`.

```bash
grep -A 40 "type <TypeName> struct" go/types/<module>/*.pb.go
```

Use JSON name from `protobuf:` tag (the `json=fieldName,proto3"` part), not standalone `json:` tag.

Common mistakes: field from related type, invented field names, Go field name instead of JSON name (`AlarmId` vs `alarmId`).

## TestLocationAndApproach

All tests MUST live in `go/tests/`. No `_test.go` files alongside source code. Tests MUST use system API (IVNic, HTTP endpoints) end-to-end, never call unexported/internal functions.

```bash
find go/ -name "*_test.go" -not -path "go/tests/*"  # Must return nothing
```

## VendorAndGit

### Rule 1: NEVER Edit Vendor
Never edit any file in `vendor/`. Make changes in the actual sibling project source (e.g., `../l8orm/go/orm/...`).

### Rule 2: NEVER Run Vendor/Module Commands
Never run `go mod tidy`, `go mod vendor`, `go mod init`, `rm -rf vendor/go.mod/go.sum`.

### Rule 3: NEVER Run Git Commands Unless Instructed

### Rule 4: `vendor/` Must NEVER Be Tracked in Git
`.gitignore` must exclude `go/vendor/` (an uncommented `go/vendor/` line, not `# vendor/`). A real incident: this rule was violated when a project's `.gitignore` had it commented out, committing the entire vendor tree. If you ever find `vendor/` tracked, untrack it immediately (`git rm -r --cached go/vendor`) and fix `.gitignore` before doing anything else — do not wait to be asked.

### Dependency Location
Search in `go/vendor/github.com/saichler/<dependency>/...`, not in sibling directories or module cache.

## VerifyAppHtmlScriptsAgainstLoadingOrder

After ANY UI PRD implementation, verify `app.html` and `m/app.html` script tags against canonical loading order. Missing script causes silent cascading failure -- `ReferenceError` kills IIFEs, breaks all navigation/tables/CRUD/forms.

```bash
grep 'src="l8ui/' go/<project>/ui/web/app.html | sed 's/.*src="//' | sed 's/".*//'
```

Commonly missed: `layer8-module-factory-core.js`, `layer8d-service-registry.js`, `layer8d-forms-fields.js`.

Symptom: `Uncaught ReferenceError: <GlobalName> is not defined`, page loads but nothing works.

## VerifyPrdCompletenessBeforeDone

Do NOT report PRD complete until every component is verified as implemented. Walk the PRD section by section.

1. **Extract deliverables**: services, binaries, data flows, startup behavior, integrations
2. **Verify in code**:
```bash
grep -rn "ServiceName" go/<project>/ --include="*.go" | grep -v vendor/
grep -rn "Activate" go/<project>/*/main.go | grep -v vendor/
```
3. **Trace data end-to-end**: origin -> transform -> destination -> UI endpoint
4. **Verify UI displays data** (not just loads)

Common gaps: persist layer has constants but no implementation, startup warm-up never written, pipeline middle stage hollow, scheduled tasks never triggered, services work in isolation but never connect.
