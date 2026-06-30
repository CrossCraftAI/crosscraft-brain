# TODO — crosscraft-brain

---

## Brand consistency fixes ✅

All three hardcoded dark colors that leaked through merge — fixed 2026-06-21.
Root cause: React Flow / shadcn props, not Tailwind classes, so the rebrand sweep missed them.

| # | File | Line | Offending | Fix |
|---|------|------|-----------|-----|
| 1 | `apps/web/src/components/Editor.tsx` | 458 | ~~`<Background color="#1c2230" />`~~ | `color="var(--border-2)"` (Cloudy `#b1ada1`) |
| 2 | `apps/web/src/components/Editor.tsx` | 463 | ~~`maskColor="rgba(8,11,17,0.7)"`~~ | `maskColor="rgba(244,243,238,0.85)"` (Pampas `--bg`) |
| 3 | `apps/web/src/components/ui/dialog.tsx` | 19 | ~~`bg-black/60`~~ | `bg-[color-mix(in_srgb,var(--text)_60%,transparent)]` (warm near-black) |

---

## Mobile / Client Track — the crosscraft engine as a mobile backend

Rationale: crosscraft's Go binary is a **highly efficient, single-binary workflow
executor**. iOS/Android apps integrate via the REST API and push notifications;
the server handles all heavy lifting (OAuth2, API orchestration, AI, file
processing, scheduling) — the mobile app is a thin UI + triggers.

### Mobile enablers — built 2026-06-21

- [x] **API key auth** — `internal/auth`: bearer-token middleware (`cc_<nanoid>`),
      SHA-256 hashed, optional enforcement (`AUTH_REQUIRED=true`). Keys created
      via `POST /api/keys`, listed/deleted via `GET/DELETE /api/keys`. Keys can
      be embedded in webhook URLs (`?api_key=...`), making mobile-triggered
      webhooks trivially authenticated.
- [x] **Push notification node** — `core.pushNotification`: FCM HTTP v1 sender,
      JWT-assertion token exchange from service-account JSON. Sends to a device
      token with title/body/data payload. Works on both Android and iOS.
- [x] **Form trigger** — `core.formTrigger`: like webhook trigger but with
      required-field validation; designed for mobile form POSTs.
- [x] **Webhook Respond node** — `core.webhookRespond`: workflow reaches this
      node, sends a custom HTTP response (JSON body + status) to the caller,
      then suspends. Enables "POST form → process → respond with result → wait
      for next action" mobile interaction loops.
- [x] **FCM credential type** — `fcmServiceAccount` (project_id + service-account
      JSON key) in `credtype.Default()`.
- [x] **`db/mobile_schema.sql`** — `api_keys` table with hash index.

### Mobile enablers — next up

- [x] **OAuth2 for mobile** — PKCE flow (`S256`, `code_challenge`) shipped.
      `credtype.OAuth2.PKCE`, code_verifier generation/challenge, enabled on
      Google OAuth2.
- [x] **Load-options** — dynamic dropdown endpoint shipped.
      `GET /api/nodes/{type}/options?param=...&query=...&credentialId=...`
- [ ] **Deep-link resume** — mobile apps need to open `crosscraft://resume/{id}`
      URLs that POST to `/api/resume/{id}`. A mobile-optimized resume endpoint
      that accepts simpler payloads and returns compact JSON.
- [ ] **Barcode / QR trigger** — a `core.qrTrigger` that accepts `?code=...`
      query param (from mobile camera scanner), validates code format, and
      triggers workflows. Zero-config path: `POST /api/webhook/scan?api_key=...`
      with `{"code": "..."}` works today; this node adds code-format validation
      and lookup (SKU, serial, GS1).
- [ ] **Mobile webhook templates** — pre-built workflow templates for common
      mobile patterns: "Scan → Lookup → Respond", "Form → Validate → Notify",
      "Location → Geofence → Alert".
- [ ] **React Native / Flutter SDK** — thin TypeScript client lib that wraps the
      REST API + SSE stream + push notification registration. Ships as an npm
      package (`@crosscraft/mobile-client`).
- [ ] **SSE push bridge** — when a workflow reaches a `core.pushNotification`
      node, optionally bridge the SSE stream to the mobile device via FCM data
      message so the app can update its UI live.
- [ ] **Offline queue** — mobile-optimized trigger that accepts batched/
      timestamped items and replays them in order when connectivity returns.
- [ ] **Biometric / device attestation** — credential type that validates mobile
      device integrity (iOS DeviceCheck, Android SafetyNet/Play Integrity).

### Reprioritised existing items (mobile-first ordering)

| Priority | Item | Why mobile-first |
|----------|------|------------------|
| 1 | OAuth2 PKCE (done) | Mobile apps can't keep client secrets |
| 2 | Webhook Respond (done) | Mobile needs request → response loops |
| 3 | Load-options (done) | Mobile pickers (spreadsheets, channels) |
| 4 | SSE stream optimisation | Live run monitoring on mobile |
| 5 | Webhook trigger templates | Common mobile interaction patterns |
| 6 | Error + Execute Workflow (done) | Compose workflows from mobile triggers |
| 7 | Form Trigger (done) | Mobile form submissions |
| 8 | Push notifications (done) | Re-engage mobile users |
| 9 | API key auth (done) | Authenticate mobile clients |

---

## Integration Nodes Roadmap (Go-native)

Build a first-party catalog of integration nodes, in Go, prioritising the stacks our
users live in: **Google → Microsoft → Adobe**, then the long tail. n8n's node catalog
is the reference for *which operations matter* (resource → operation shape); the
implementation is our own native-Go `NodeDefinition`s, which buys us connection
pooling, real concurrency, streaming uploads/downloads, typed official SDKs, and a
single static binary.

> Legend: `[ ]` not started · `[~]` in progress · `[x]` done.
> Each node's bullets are its **operations** (n8n-style). A node is "done" only with
> its trigger(s), credential type, golden-path test, and a palette icon/description.

---

## How nodes work here (so this list is actionable)

- A node is a `schema.NodeDefinition` in `server/internal/nodes/<pack>/…`, registered
  in [main.go](server/cmd/crosscraft/main.go) via `registry.New().Register(...)`.
- Built-ins (`set`, `if`, `http`, `code`, `wait`, triggers) live in
  [nodes/core](server/internal/nodes/core); AI in [nodes/ai](server/internal/nodes/ai).
- New packs go in `server/internal/nodes/{google,microsoft,adobe}` and register the
  same way. Group them with `group: 'integration'` (or `'trigger'`/`'transform'`).
- Credentials: the AES-256-GCM store (`store.CreateCredential` / `ctx.Credential`)
  already holds arbitrary JSON. Param type `credential` + `credentialType` wires the
  picker. **Missing piece:** an OAuth2 authorization-code flow (see Phase 0).
- **Definition of Done per node:** operations implemented · OAuth2/credential type ·
  pagination + rate-limit/retry · trigger (poll or webhook) where n8n has one ·
  one end-to-end test (httptest or sandbox) · icon + description + param schema.

---

## Phase 0 — Foundational infra (blocks every OAuth integration)

These are prerequisites, not optional. Build once, reuse everywhere.

- [x] **OAuth2 credential flow** — `internal/oauth`: authorization-code
      (`GET /api/oauth2/auth-url` + `/callback`) **and** client-credentials
      (server-to-server). Refresh + persist back to the encrypted credential blob.
      **PKCE shipped** (S256 code challenge, enabled for Google OAuth2).
- [x] **Credential *types* registry** — `internal/credtype` + `GET /api/credential-types`
      (Google / Microsoft / Adobe IMS / generic OAuth2 / header-auth / Adobe Sign).
- [x] **Per-service token source** — auto-refreshing `*http.Client` via
      `oauth.ClientForCredential`, wired to nodes through `ExecContext.AuthorizedClient`.
- [x] **Declarative REST node framework** — `internal/rest`: data-defined
      resources/operations → `NodeDefinition` (path interpolation, query/JSON body,
      header/OAuth2 auth, retry, response→items, shared-param dedupe, `BaseURLParam`).
- [x] **Pagination / rate-limit / retry** — 429/5xx retry with `Retry-After` done.
      **Cursor / page-token / offset pagination shipped** (`rest.Pagination`, auto
      page-walking with max-pages guard).
- [x] **Binary data handling** — in-memory base64 via `Item.Binary` works (Drive media
      upload/download). **Streaming binary store shipped** (`internal/store/binary.go`):
      `BinaryStore` interface with disk-backed and in-memory implementations; streaming
      Put/Get/Delete/Exists/GetURL/Cleanup; size limits; retry support; path-traversal
      protection. Files are keyed by execution run for automatic lifecycle management.
- [x] **Load-options ("resource locator")** — `GET /api/nodes/{type}/options?param=...`
      shipped. `NodeDefinition.LoadOptions` + `ParamSchema.HasDynamicOptions` +
      `Registry.LoadOptions`; UI gets `hasLoadOptions` in descriptors.
- [x] **Trigger infra** — **schedule/cron trigger shipped** (`internal/scheduler` +
      `core.scheduleTrigger`, interval + 5-field cron via robfig/cron).
      **Generalised polling triggers shipped** (`internal/triggers/polling.go`):
      reusable `Poller` abstraction with rate-limiting, idle backoff, deduplication
      across sessions, cursor management, and convenience constructors for Sheets/Gmail/
      Drive/Calendar/Forms triggers. Durable state via `ExecContext.State`.
- [x] **Generic escape hatches** — `core.http` already works with header-auth
      credentials. **Microsoft Graph generic shipped** (`microsoft.graph`: GET/POST/
      PATCH/PUT/DELETE with full URL override via `BaseURLParam`).

---

## Phase 1 — Google Workspace & Cloud  (`nodes/google`)

**Go SDKs:** `google.golang.org/api/<svc>/<ver>` (sheets/v4, gmail/v1, calendar/v3,
drive/v3, docs/v1, slides/v1, people/v1, tasks/v1, forms/v1, chat/v1, youtube/v3,
analyticsdata/v1beta, …), `cloud.google.com/go/*` (storage, bigquery, firestore,
pubsub, translate, language, vision), auth via `golang.org/x/oauth2/google`.
**Auth:** OAuth2 (per-user) + Service Account / domain-wide delegation option.

### Workspace
- [x] **Google Sheets** — shipped: spreadsheet get/create; values get/append/update/clear.
      _Remaining:_ delete spreadsheet, delete rows/cols, header→object row mapping,
      **Trigger** (rowAdded/rowUpdated polling).
- [x] **Gmail** (read) — shipped: message list/get, label list.
      _Remaining:_ send/reply (MIME build), drafts, threads, mark read / labels,
      **Trigger** (polling).
- [x] **Google Calendar** — shipped: event list/get/create/delete, calendar list.
      _Remaining:_ event update, free/busy availability, **Trigger**.
- [x] **Google Drive** — shipped: file list/get/delete, folder create, **media
      upload/download** (`google.driveUpload` / `google.driveDownload`; multipart +
      `alt=media`). _Remaining:_ copy/move/share, create-from-text, shared drives,
      **Trigger**; true streaming via the binary store.
- [x] **Google Docs** — Document: Create, Get, Update (insert/replace text, styling), Delete
- [x] **Google Slides** — Presentation: Create, Get, Replace Text, Get Page Thumbnail
- [x] **Google Contacts (People API)** — Contact: Create, Get, List, Update, Delete
- [x] **Google Tasks** — Task: Create, Get, List, Update, Delete; Task List: CRUD
- [x] **Google Forms** + **Trigger** (new response) — Form: Get, List; Response: Get, List
- [x] **Google Chat** — Message: Send; Space: Get, List; Member: List
- [ ] **Gemini** — *already covered by AI nodes; add a Google-auth variant if needed*

### Google Cloud
- [ ] **Google Cloud Storage** — Bucket: CRUD; Object: Upload (stream), Download
      (stream), Get, Get Many, Update, Delete
- [ ] **BigQuery** — Execute Query (SQL); Record: Insert, Get Many; Dataset/Table: manage
- [ ] **Cloud Firestore** — Document: Create, Get, Get Many, Update, Delete, Query;
      Collection: list
- [ ] **Cloud Pub/Sub** — Publish Message; Subscription: Pull (+ trigger)
- [ ] **Cloud Translation** — Translate Text, Detect Language
- [ ] **Cloud Natural Language** — Analyze Sentiment / Entities / Syntax, Classify
- [ ] **Cloud Vision** — Label/Text/Face/Safe-search Detection (OCR)
- [ ] **Cloud Speech-to-Text / Text-to-Speech** — Transcribe / Synthesize (stream)

### Google Marketing / Media
- [ ] **Google Analytics (GA4)** — Report: Run; User Activity
- [ ] **Google Ads** — Campaign/AdGroup: Get, Get Many; report queries
- [ ] **Google Search Console** — Search Analytics query; Sitemaps
- [ ] **YouTube** — Video: Upload (stream), Get, Get Many, Update, Delete, Rate;
      Channel/Playlist/PlaylistItem: manage; Comment, Subscription, Search
- [ ] **Google Business Profile** + **Trigger** — Post, Review (reply), Location
- [ ] **Google Perspective** — Analyze Comment (toxicity)

---

## Phase 2 — Microsoft 365 & Azure  (`nodes/microsoft`)

**Go SDKs:** `github.com/microsoftgraph/msgraph-sdk-go` (Kiota) for 365; auth
`github.com/Azure/azure-sdk-for-go/sdk/azidentity`; Azure data via
`github.com/Azure/azure-sdk-for-go/sdk/...` (azblob, azcosmos); MSSQL via
`github.com/microsoft/go-mssqldb`. **Auth:** OAuth2 (Microsoft identity platform).

### Microsoft 365 (Graph) — **shipped** (declarative, `microsoftOAuth2Api`, Graph v1.0)
- [x] **Outlook** — core mail (list/get/send, …)
- [x] **Microsoft Calendar (Graph)** — events: list/get/create/delete
- [x] **Excel (Graph)** — worksheets + tables (rows)
- [x] **OneDrive** — files/folders (metadata)
- [x] **Microsoft Teams** — channels + messages
- [x] **Microsoft To Do** — task lists + tasks

### Microsoft tail — **complete**

- [x] **Flesh out shipped services** — Outlook: reply, move, drafts, folders,
      attachments; Excel: range get/update + workbook sessions; Teams: channel CRUD;
      Calendar: update + list calendars; OneDrive: copy/move/share/search; To Do: full CRUD.
- [x] **SharePoint** (Graph `…/sites/{siteId}`) — Site: Get/Search/List; List:
      Get/List/Create/Update/Delete; List Item: Get/List/Create/Update/Delete; Drive/File:
      list/get.
- [x] **OneNote** (Graph) — Notebook: Get/List/Create; Section: Get/List/Create; Page:
      Get/List/Create/Delete.
- [x] **Microsoft Graph (generic)** — raw authenticated Graph call (GET/POST/PATCH/PUT/DELETE)
      with full URL override via `BaseURLParam`; the escape hatch for anything unwrapped.
- [ ] **Triggers** (Outlook / Teams / OneDrive / SharePoint) — Graph **change-notification
      subscriptions** (webhooks) with subscription create/renew/validate, into the durable
      `wait`/resume plumbing; **delta-query polling** fallback. Needs Phase-0 trigger infra.
- [ ] **OneDrive / SharePoint media** — upload (`PUT /content`; resumable upload session
      for >4 MB) + download (`GET /content`) into `Item.Binary`, mirroring
      `google.driveUpload/Download`; true streaming via the binary store.
- [ ] **Dynamics 365 (CRM)** — Web API (`/api/data/v9.2`): Account, Contact, Lead,
      Opportunity + arbitrary entity: Create/Get/Get Many (OData `$filter`/`$select`)/
      Update/Delete. Declarative + `BaseURLParam` for the org URL; a `dynamicsOAuth2Api`
      cred (resource-scoped token).

### Azure — **shipped** (declarative REST + native auth)
- [x] **Azure Blob Storage** — Container: list/create/delete; Blob: list/upload/
      download/delete/copy (8 ops). Shared Key HMAC-SHA256 auth via stdlib crypto.
      Declarative REST. Credential: `azureStorage`.
- [x] **Azure Cosmos DB** — Database: list/get/create/delete; Container: list/create/
      delete; Item: get/create/update/delete/query (12 ops). Master Key HMAC auth.
      Declarative REST. Credential: `azureCosmos`.
- [x] **Microsoft SQL Server** — Query many/single, Execute, Stored Procedure (4 ops).
      `azure.mssql` node with `mssql` credential. Ready for `go-mssqldb` driver.
-[~] **PostgreSQL** — Execute Query/Insert/Update/Delete: a **DB node** via
      `github.com/jackc/pgx` (connection-string credential), parameterized queries.      
- [x] **Power BI** — Dataset: list/get/pushRows/refresh; Report: list/get; Dashboard:
      list/get; Groups: list (9 ops). OAuth2 auth. Declarative REST via `azurePowerBI`.
- [x] **Azure DevOps** — Work Item: list/get/create/update/delete; Pipeline: list/get/
      run; Repo: list/get; Pull Request: list/get/create (13 ops). PAT Basic auth.
      `BaseURLParam` per-org. Declarative REST via `azureDevOps`.
- [x] **Azure OpenAI** — Completion, Chat, Embeddings, DALL-E, Whisper transcribe/
      translate (6 ops). API key auth. `BaseURLParam` per-resource. Declarative REST.

---

## Phase 3 — Adobe  (`nodes/adobe`)

**Note:** Adobe ships **no official Go SDKs** → REST on the declarative framework.
Auth is ready: **`adobeOAuth2Api`** (IMS server-to-server / client-credentials) and
**`adobeSignApi`** (integration key). The remaining Adobe APIs below are mostly
**async job** flows (submit → poll → download) over **binary**, so they need the
Phase-0 streaming binary store + a small job-poll helper before they're built.

- [x] **Adobe Acrobat Sign** (e-signature) — shipped: agreement list/get/create/send/
      cancel/getDocuments/signingUrls/getEvents, reminders, library documents, webhooks
      (13 ops). Auth: integration key (Bearer) via `adobeSignApi`; account shard
      overridable per node (`baseUrl`).
- [x] **Adobe PDF Services API** — shipped: Create PDF, Export PDF, OCR, Compress,
      Combine/Merge, Split, Extract (text/tables/figures), Document Generation,
      job status/download (10 ops). Auth via `adobeOAuth2Api` (IMS server-to-server).
- [x] **Adobe Firefly Services** — Generate Image (text-to-image), Generative Fill,
      Generative Expand, Upscale (4 ops). Auth via `adobeOAuth2Api`.
- [x] **Adobe Photoshop API** — Apply Edits, Smart Object replace, Run Action, Create
      Rendition, job status/manifest (6 ops). Auth via `adobeOAuth2Api`.
- [x] **Adobe Lightroom API** — Auto-Tone, Apply Preset, Edit, Get Rendition, asset
      list/get/upload (7 ops). Auth via `adobeOAuth2Api`.
- [x] **Adobe Experience Manager (AEM) Assets** — Upload, Get, Update Metadata,
      Delete, Get Rendition, list assets/folders, create folder (8 ops). Base URL
      overridable. Auth via `adobeOAuth2Api`.
- [x] **Adobe Analytics** — Run Report, Top Items, list Segments/Metrics/Dimensions,
      get Segment/Dimension items (7 ops). Auth via `adobeOAuth2Api`.
- [x] **Adobe Stock** — Search, Get Details, License, Download, list/get licenses,
      list collections (7 ops). Auth via `adobeOAuth2Api`.
- [x] **Adobe Commerce (Magento)** — Customer CRUD, Product CRUD, Order list/get/create/
      cancel, Invoice list/get/create, store views/config (17 ops). Base URL
      overridable. Auth via `adobeCommerceApi` (access token).
- [ ] **Adobe Target** — Activity/Offer/Audience: manage (lower priority)

---

## Phase 4 — Core "function" nodes (n8n built-ins we still owe)

Beyond integrations, n8n ships logic/utility nodes. Several already exist; the rest
round out the editor so workflows don't need the Code node for everything.

**Have:** `manualTrigger`, `webhookTrigger`, `set` (Edit Fields), `if`, `http`,
`code`, `wait` — plus the Phase-4 batch below (all in `nodes/core`, unit-tested).

- [x] **Flow** (shipped): Switch, Filter, Merge (append), Split Out, Aggregate, Limit,
      Sort, Remove Duplicates, No Operation, Stop & Error, **Compare Datasets**
      (`core.compareDatasets`: dual-input, 4 output ports).
      **Loop / Split In Batches** shipped (`core.loop`: forEach + splitBatches).
- [x] **Triggers:** Schedule/Cron **shipped** (`core.scheduleTrigger`).
      **Form Trigger shipped** (`core.formTrigger`), **Error Trigger shipped**
      (`core.errorTrigger`), **Execute Workflow shipped** (`core.executeWorkflow`
      + engine `RunSubWorkflow`).
      **Read Email (IMAP/POP3) shipped** (`core.readEmail`: basic IMAP/POP3 via stdlib).
- [x] **Data:** shipped: Date & Time (now/parse/add/subtract), Crypto (hash / HMAC /
      Base64), Rename Keys, **Extract From File** (CSV/JSON/text), **Convert to File**
      (CSV/JSON), **Compression** (gzip/zip compress+decompress), **HTML Extract**
      (tag-strip), **JSON** (parse/stringify), **Sort Keys**.
      **Edit Image** (`core.editImage`: resize/rotate/convert/info),
      **Extract From File** (extended: XML/PDF basic/ODS),
      **Spreadsheet File** (`core.spreadsheetFile`: CSV/XLSX read/write/append),
      **Markdown** (`core.markdown`: toHtml/toText/extract).
- [x] **Comms primitives:** **Send Email (SMTP)**, **Execute Command**, **RSS Read**
      (RSS 2.0 + Atom 1.0) shipped. **Push Notification (FCM) shipped**
      (`core.pushNotification`).
      **Read Email (IMAP/POP3)** (`core.readEmail`), **FTP/SFTP** (`core.ftp`), **SSH** (`core.ssh`).
      **Webhook Respond shipped** (`core.webhookRespond`).
- [ ] **AI cluster (LangChain-style):** AI Agent, Basic LLM Chain, Q&A/Retrieval Chain,
      Vector Store (Pinecone/PGVector), Embeddings, Memory, Tool nodes, Output Parser,
      Text Splitter, Document Loader  *(builds on existing `nodes/ai` + goja tools)*

---

## Phase 5 — Common integrations backlog (broader n8n catalog, prioritised)

Ordered roughly by demand. Most are REST → declarative framework; webhooks where the
provider supports them.

- [x] **Communication:** **Slack** (8 ops: message CRUD, channel/user/file list),
      **Discord** (8 ops: message CRUD, channel, guild/member list), **Telegram** (6 ops),
      **Twilio** (4 ops: SMS/WhatsApp/voice). Shipped in `nodes/comm`.
- [x] **Productivity / PM:** **Notion** (9 ops: page/database/block/user/search),
      **Airtable** (5 ops), **Asana** (8 ops), **Trello** (8 ops), **ClickUp** (7 ops),
      **Jira** (7 ops), **Linear** (7 ops), **Todoist** (9 ops). Shipped in `nodes/productivity`.
- [x] **CRM / Marketing:** **HubSpot**, **Pipedrive**, **Mailchimp**, **SendGrid** shipped
      in `nodes/crm`.
- [x] **Dev / DevOps:** **GitHub** (14 ops: repos, issues, PRs, files, commits, releases),
      **GitLab** (10 ops), **Sentry** (8 ops). Shipped in `nodes/dev`.
- [x] **Cloud / Storage / DB:** **AWS** (S3: 6 ops, SES: 4 ops, SQS: 4 ops, Lambda: 2 ops,
      DynamoDB: 7 ops with typed-attribute unwrapping). SigV4 signing via stdlib crypto.
      Shipped in `nodes/aws`. **PostgreSQL** shipped (`nodes/database`).
      **MongoDB** (10 ops: find/findOne/insertOne/insertMany/updateOne/updateMany/
      deleteOne/deleteMany/aggregate/trigger) shipped. **MySQL** (5 ops: query:many/
      query:one/exec/storedProcedure:exec/trigger) shipped. **Redis** (16 ops: get/set/
      delete/expire/incr/decr/hget/hset/hgetall/lpush/rpop/sadd/smembers/publish/
      subscribe/trigger) shipped. **Snowflake** (4 ops: query:many/query:one/exec/
      trigger) shipped. **Supabase** (6 ops: select/insert/update/delete/rpc/trigger)
      shipped. **Dropbox** (8 ops: list/upload/download/delete/move/copy/share/trigger)
      shipped. **Box** (8 ops: list/upload/download/delete/move/copy/share/trigger)
      shipped. All in `nodes/database` and `nodes/storage`.
- [x] **Payments / Commerce:** **Stripe** (16 ops), **PayPal** (12 ops: orders, payments,
      refunds, webhooks, invoices), **Square** (12 ops: payments, orders, customers,
      refunds, locations). Shipped in `nodes/payments`.
- [x] **E-commerce:** **Shopify** (13 ops: products, orders, customers, draft orders,
      inventory), **WooCommerce** (14 ops: products, orders, customers, coupons,
      reports). Shipped in `nodes/commerce`.
- [x] **Accounting:** **QuickBooks Online** (11 ops: invoices, customers, expenses,
      P&L/balance reports), **Xero** (12 ops: invoices, contacts, bank transactions,
      reports, accounts). Shipped in `nodes/accounting`.
- [x] **CRM extended:** **Salesforce** (12 ops: accounts, contacts, leads,
      opportunities, SOQL query, object describe). Shipped in `nodes/crm`.
- [x] **AI / ML:** OpenAI, Hugging Face, Cohere, Mistral, Pinecone, Qdrant, ElevenLabs,
      Stability AI, Perplexity — shipped in `nodes/ai`
- [x] **Generic protocols:** GraphQL, gRPC, SOAP, MQTT, AMQP/RabbitMQ, Kafka, NATS,
      WebSocket — shipped in `nodes/protocols`

---

## Why Go makes these "highly efficient" (design notes)

- **Official typed SDKs** for Google & Microsoft (Kiota-generated Graph SDK) → less
  hand-rolled REST, fewer bugs, native streaming.
- **Connection pooling & keep-alive** shared across runs (one `*http.Client` per
  credential) instead of per-request clients.
- **Streaming binary I/O** for Drive/OneDrive/GCS/Blob/PDF — never buffer whole files;
  pipe through the run with bounded memory.
- **Real concurrency** — fan-out over items/pages with a bounded pool (reuse the
  engine's worker-pool pattern); rate-limit centrally.
- **Single static binary** — every integration ships in the one `crosscraft` binary;
  no per-node runtime, no plugin installs.

## Cross-cutting checklist (apply to every node)

- [ ] Credential type registered (OAuth2 scopes / API key fields)
- [ ] Pagination + rate-limit + retry (`Retry-After`, backoff)
- [ ] Streaming for any file upload/download
- [ ] Trigger (polling cursor or webhook) where n8n has one
- [ ] `continueOnFail` + structured error items (don't kill the run on one bad item)
- [ ] Load-options for pickers (spreadsheets, mailboxes, channels…)
- [ ] Golden-path test (httptest mock or sandbox account) + palette icon/description
