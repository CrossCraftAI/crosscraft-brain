// Package credtype declares credential types: what fields a credential needs and,
// for OAuth2 types, the provider endpoints. The canvas renders the form / "Connect"
// button from these descriptors; the oauth service uses the OAuth2 spec to run the
// authorization-code flow. Mirrors n8n's credential types.
package credtype

// Field is one input on a credential form.
type Field struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "string" | "password"
	Required    bool   `json:"required,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

// OAuth2 is the provider spec for an OAuth2 credential type. For the generic type
// the endpoints are empty and read from the credential's own data instead.
type OAuth2 struct {
	AuthURL    string            `json:"authUrl"`
	TokenURL   string            `json:"tokenUrl"`
	Scopes     []string          `json:"scopes,omitempty"`
	AuthParams map[string]string `json:"authParams,omitempty"` // extra authorize-URL params
	// GrantType: "" / "authorization_code" (user redirect flow) or
	// "client_credentials" (server-to-server; no AuthURL, no user step).
	GrantType string `json:"grantType,omitempty"`
	// PKCE enables Proof Key for Code Exchange (RFC 7636, S256) for this type.
	// Mobile/native clients need this since they cannot keep a client secret.
	PKCE bool `json:"pkce,omitempty"`
}

// Type describes a credential type.
type Type struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"displayName"`
	Fields      []Field `json:"fields"`
	OAuth2      *OAuth2 `json:"oauth2,omitempty"`
}

// IsOAuth2 reports whether this type uses the OAuth2 flow.
func (t Type) IsOAuth2() bool { return t.OAuth2 != nil }

// Registry is an in-memory set of credential types.
type Registry struct{ m map[string]Type }

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{m: map[string]Type{}} }

// Register adds types (chainable).
func (r *Registry) Register(ts ...Type) *Registry {
	for _, t := range ts {
		r.m[t.Name] = t
	}
	return r
}

// Get returns a type by name.
func (r *Registry) Get(name string) (Type, bool) { t, ok := r.m[name]; return t, ok }

// All returns every registered type (for GET /api/credential-types).
func (r *Registry) All() []Type {
	out := make([]Type, 0, len(r.m))
	for _, t := range r.m {
		out = append(out, t)
	}
	return out
}

var clientFields = []Field{
	{Name: "clientId", Label: "Client ID", Type: "string", Required: true},
	{Name: "clientSecret", Label: "Client Secret", Type: "password", Required: true},
	{Name: "scope", Label: "Scopes (space-separated)", Type: "string"},
}

// Default returns the built-in credential types.
func Default() *Registry {
	return NewRegistry().Register(
		Type{
			Name: "googleOAuth2Api", DisplayName: "Google OAuth2", Fields: clientFields,
			OAuth2: &OAuth2{
				AuthURL:    "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:   "https://oauth2.googleapis.com/token",
				AuthParams: map[string]string{"access_type": "offline", "prompt": "consent"},
				PKCE:       true, // mobile clients can't keep client secrets
			},
		},
		Type{
			Name: "microsoftOAuth2Api", DisplayName: "Microsoft OAuth2", Fields: clientFields,
			OAuth2: &OAuth2{
				AuthURL:    "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL:   "https://login.microsoftonline.com/common/oauth2/v2.0/token",
				Scopes:     []string{"offline_access"},
				AuthParams: map[string]string{"prompt": "consent"},
			},
		},
		Type{
			Name: "oAuth2Api", DisplayName: "Generic OAuth2",
			Fields: append([]Field{
				{Name: "authUrl", Label: "Authorization URL", Type: "string", Required: true},
				{Name: "tokenUrl", Label: "Token URL", Type: "string", Required: true},
			}, clientFields...),
			OAuth2: &OAuth2{}, // endpoints come from the credential data
		},
		Type{
			Name: "httpHeaderAuth", DisplayName: "Header Auth (API key)",
			Fields: []Field{
				{Name: "name", Label: "Header Name", Type: "string", Required: true, Placeholder: "Authorization"},
				{Name: "value", Label: "Header Value", Type: "password", Required: true},
			},
		},
		Type{
			Name: "adobeSignApi", DisplayName: "Adobe Acrobat Sign (Integration Key)",
			Fields: []Field{
				{Name: "accessToken", Label: "Integration Key / Access Token", Type: "password", Required: true},
			},
		},
		Type{
			Name: "adobeOAuth2Api", DisplayName: "Adobe IMS (Server-to-Server)", Fields: clientFields,
			OAuth2: &OAuth2{TokenURL: "https://ims-na1.adobelogin.com/ims/token/v3", GrantType: "client_credentials"},
		},
		// ── Adobe Commerce ──────────────────────────────────────────────────────
		Type{Name: "adobeCommerceApi", DisplayName: "Adobe Commerce (Magento) Access Token", Fields: []Field{
			{Name: "accessToken", Label: "Access Token", Type: "password", Required: true},
		}},
		// ── Communication ───────────────────────────────────────────────────────
		Type{Name: "slackApi", DisplayName: "Slack (Bot Token)", Fields: []Field{
			{Name: "accessToken", Label: "Bot Token (xoxb-...)", Type: "password", Required: true},
		}},
		Type{Name: "discordApi", DisplayName: "Discord (Bot Token)", Fields: []Field{
			{Name: "accessToken", Label: "Bot Token", Type: "password", Required: true},
		}},
		Type{Name: "telegramApi", DisplayName: "Telegram Bot API", Fields: []Field{
			{Name: "accessToken", Label: "Bot Token", Type: "password", Required: true},
		}},
		Type{Name: "twilioApi", DisplayName: "Twilio", Fields: []Field{
			{Name: "accountSid", Label: "Account SID", Type: "string", Required: true},
			{Name: "authToken", Label: "Auth Token", Type: "password", Required: true},
		}},
		// ── Productivity / PM ────────────────────────────────────────────────────
		Type{Name: "notionApi", DisplayName: "Notion (Integration Token)", Fields: []Field{
			{Name: "accessToken", Label: "Internal Integration Token (secret_...)", Type: "password", Required: true},
		}},
		Type{Name: "airtableTokenApi", DisplayName: "Airtable (Personal Access Token)", Fields: []Field{
			{Name: "accessToken", Label: "Personal Access Token (pat...)", Type: "password", Required: true},
		}},
		Type{Name: "linearApi", DisplayName: "Linear API Key", Fields: []Field{
			{Name: "accessToken", Label: "API Key", Type: "password", Required: true},
		}},
		Type{Name: "todoistApi", DisplayName: "Todoist API Token", Fields: []Field{
			{Name: "accessToken", Label: "API Token", Type: "password", Required: true},
		}},
		Type{Name: "asanaApi", DisplayName: "Asana (Personal Access Token)", Fields: []Field{
			{Name: "accessToken", Label: "Personal Access Token", Type: "password", Required: true},
		}},
		Type{Name: "clickUpApi", DisplayName: "ClickUp API Token", Fields: []Field{
			{Name: "accessToken", Label: "API Token", Type: "password", Required: true},
		}},
		Type{Name: "jiraCloudApi", DisplayName: "Jira Cloud (Email + API Token)", Fields: []Field{
			{Name: "email", Label: "Account Email", Type: "string", Required: true},
			{Name: "apiToken", Label: "API Token", Type: "password", Required: true},
			{Name: "subdomain", Label: "Subdomain (e.g. mycompany)", Type: "string", Required: true},
		}},
		Type{Name: "trelloApi", DisplayName: "Trello (API Key + Token)", Fields: []Field{
			{Name: "apiKey", Label: "API Key", Type: "string", Required: true},
			{Name: "accessToken", Label: "Token", Type: "password", Required: true},
		}},
		// ── CRM / Marketing ──────────────────────────────────────────────────────
		Type{Name: "hubspotApi", DisplayName: "HubSpot (Private App Token)", Fields: []Field{
			{Name: "accessToken", Label: "Private App Access Token", Type: "password", Required: true},
		}},
		Type{Name: "mailchimpApi", DisplayName: "Mailchimp", Fields: []Field{
			{Name: "accessToken", Label: "API Key", Type: "password", Required: true},
			{Name: "server", Label: "Server Prefix (e.g. us1)", Type: "string", Required: true},
		}},
		Type{Name: "sendgridApi", DisplayName: "SendGrid API Key", Fields: []Field{
			{Name: "accessToken", Label: "API Key (SG....)", Type: "password", Required: true},
		}},
		Type{Name: "pipedriveApi", DisplayName: "Pipedrive API Token", Fields: []Field{
			{Name: "accessToken", Label: "API Token", Type: "password", Required: true},
		}},
		// ── Payments / Commerce ───────────────────────────────────────────────────
		Type{Name: "stripeApi", DisplayName: "Stripe Secret Key", Fields: []Field{
			{Name: "accessToken", Label: "Secret Key (sk_live_... or sk_test_...)", Type: "password", Required: true},
		}},
		// ── Phase 5: Remaining Integrations ──────────────────────────────────────
	Type{Name: "salesforceApi", DisplayName: "Salesforce", Fields: clientFields,
		OAuth2: &OAuth2{
			AuthURL:    "https://login.salesforce.com/services/oauth2/authorize",
			TokenURL:   "https://login.salesforce.com/services/oauth2/token",
			AuthParams: map[string]string{"prompt": "consent"},
		}},
	Type{Name: "paypalApi", DisplayName: "PayPal", Fields: clientFields,
		OAuth2: &OAuth2{
			AuthURL:  "https://www.paypal.com/connect",
			TokenURL: "https://api-m.paypal.com/v1/oauth2/token",
		}},
	Type{Name: "shopifyApi", DisplayName: "Shopify (Access Token)", Fields: []Field{
		{Name: "accessToken", Label: "Access Token (shpat_...)", Type: "password", Required: true},
	}},
	Type{Name: "quickbooksApi", DisplayName: "QuickBooks Online", Fields: clientFields,
		OAuth2: &OAuth2{
			AuthURL:    "https://appcenter.intuit.com/connect/oauth2",
			TokenURL:   "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer",
			AuthParams: map[string]string{"prompt": "consent"},
		}},
	Type{Name: "xeroApi", DisplayName: "Xero", Fields: clientFields,
		OAuth2: &OAuth2{
			AuthURL:  "https://login.xero.com/identity/connect/authorize",
			TokenURL: "https://identity.xero.com/connect/token",
		}},
	Type{Name: "squareApi", DisplayName: "Square (Access Token)", Fields: []Field{
		{Name: "accessToken", Label: "Access Token (EAAAl...)", Type: "password", Required: true},
	}},
	Type{Name: "woocommerceApi", DisplayName: "WooCommerce (API Key)", Fields: []Field{
		{Name: "accessToken", Label: "Consumer Key:Consumer Secret (base64)", Type: "password", Required: true},
	}},
	// ── Cloud / Storage ──────────────────────────────────────────────────────
		Type{Name: "awsIam", DisplayName: "AWS IAM (Access Key)", Fields: []Field{
			{Name: "accessKey", Label: "Access Key ID (AKIA...)", Type: "string", Required: true},
			{Name: "secretKey", Label: "Secret Access Key", Type: "password", Required: true},
			{Name: "region", Label: "Region", Type: "string", Required: true, Placeholder: "us-east-1"},
		}},
		// ── Dev / DevOps ──────────────────────────────────────────────────────────
		Type{Name: "githubApi", DisplayName: "GitHub (Personal Access Token)", Fields: []Field{
			{Name: "accessToken", Label: "Personal Access Token", Type: "password", Required: true},
		}},
		Type{Name: "gitlabApi", DisplayName: "GitLab (Personal Access Token)", Fields: []Field{
			{Name: "accessToken", Label: "Personal Access Token", Type: "password", Required: true},
			{Name: "baseUrl", Label: "Base URL (default: gitlab.com)", Type: "string", Placeholder: "https://gitlab.com"},
		}},
		Type{Name: "sentryApi", DisplayName: "Sentry Auth Token", Fields: []Field{
			{Name: "accessToken", Label: "Auth Token", Type: "password", Required: true},
		}},
		// ── Mobile / Push ──────────────────────────────────────────────────────
		Type{Name: "fcmServiceAccount", DisplayName: "FCM Service Account", Fields: []Field{
			{Name: "project_id", Label: "Firebase Project ID", Type: "string", Required: true},
			{Name: "service_account_json", Label: "Service Account JSON Key", Type: "password", Required: true,
				Placeholder: "Paste the entire service-account JSON key file"},
		}},
		Type{Name: "apiKey", DisplayName: "API Key (Mobile Client)", Fields: []Field{
			{Name: "accessToken", Label: "API Key (cc_...)", Type: "password", Required: true},
		}},
		// ── Azure ──────────────────────────────────────────────────────────────
		Type{Name: "azureStorage", DisplayName: "Azure Blob Storage (Account Key)", Fields: []Field{
			{Name: "accountName", Label: "Storage Account Name", Type: "string", Required: true},
			{Name: "accessKey", Label: "Access Key (Primary or Secondary)", Type: "password", Required: true},
		}},
		Type{Name: "azureCosmos", DisplayName: "Azure Cosmos DB (Master Key)", Fields: []Field{
			{Name: "accountName", Label: "Account Name (or endpoint)", Type: "string", Required: true},
			{Name: "accessKey", Label: "Primary Master Key", Type: "password", Required: true},
		}},
		Type{Name: "mssql", DisplayName: "Microsoft SQL Server", Fields: []Field{
			{Name: "server", Label: "Server (host:port)", Type: "string", Required: true, Placeholder: "localhost:1433"},
			{Name: "database", Label: "Database Name", Type: "string", Required: true},
			{Name: "user", Label: "Username", Type: "string", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		}},
		Type{Name: "azurePowerBI", DisplayName: "Azure Power BI (OAuth2)", Fields: clientFields,
			OAuth2: &OAuth2{
				AuthURL:    "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL:   "https://login.microsoftonline.com/common/oauth2/v2.0/token",
				AuthParams: map[string]string{"resource": "https://analysis.windows.net/powerbi/api"},
			}},
		Type{Name: "azureDevOps", DisplayName: "Azure DevOps (PAT)", Fields: []Field{
			{Name: "accessToken", Label: "Personal Access Token (PAT)", Type: "password", Required: true},
		}},
		Type{Name: "azureOpenAI", DisplayName: "Azure OpenAI (API Key)", Fields: []Field{
			{Name: "accessToken", Label: "API Key", Type: "password", Required: true},
			{Name: "resourceName", Label: "Resource Name", Type: "string", Required: true},
		}},
		// ── Database ───────────────────────────────────────────────────────────
		Type{Name: "postgres", DisplayName: "PostgreSQL", Fields: []Field{
			{Name: "dsn", Label: "Connection String (optional)", Type: "string", Placeholder: "postgres://user:pass@host:5432/dbname"},
			{Name: "user", Label: "Username", Type: "string"},
			{Name: "password", Label: "Password", Type: "password"},
			{Name: "host", Label: "Host", Type: "string", Placeholder: "localhost"},
			{Name: "port", Label: "Port", Type: "string", Placeholder: "5432"},
			{Name: "dbname", Label: "Database Name", Type: "string"},
		}},
	)
}
