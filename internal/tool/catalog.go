package tool

// ConfigField describes one credential or config value for an integration.
type ConfigField struct {
	Key         string
	Label       string
	Placeholder string
	Secret      bool // render as password input
}

// Integration is a predefined template for a Tool — shown during onboarding.
type Integration struct {
	ID          string
	Label       string
	Type        string
	Category    string // "messaging" | "commerce" | "payments" | "utilities"
	Description string
	Fields      []ConfigField
}

// Catalog is the full list of supported integrations shown during onboarding.
var Catalog = []Integration{
	// ── Messaging ────────────────────────────────────────────────────────────
	{
		ID:          "whatsapp",
		Label:       "WhatsApp Business",
		Type:        "whatsapp",
		Category:    "messaging",
		Description: "Send and receive messages via the WhatsApp Business Cloud API.",
		Fields: []ConfigField{
			{Key: "phone_number_id", Label: "Phone Number ID", Placeholder: "1234567890"},
			{Key: "access_token", Label: "Access Token", Placeholder: "EAAxxxxxx…", Secret: true},
			{Key: "webhook_verify_token", Label: "Webhook Verify Token", Placeholder: "my_verify_token", Secret: true},
		},
	},
	{
		ID:          "telegram",
		Label:       "Telegram Bot",
		Type:        "telegram",
		Category:    "messaging",
		Description: "Automate conversations via a Telegram bot. Free, no approval required.",
		Fields: []ConfigField{
			{Key: "bot_token", Label: "Bot Token", Placeholder: "123456:ABC-DEF…", Secret: true},
		},
	},
	{
		ID:          "instagram",
		Label:       "Instagram",
		Type:        "instagram",
		Category:    "messaging",
		Description: "Automate DMs and comments via the Instagram Graph API. Requires Meta App Review.",
		Fields: []ConfigField{
			{Key: "page_access_token", Label: "Page Access Token", Placeholder: "EAAxxxxxx…", Secret: true},
			{Key: "instagram_account_id", Label: "Instagram Account ID", Placeholder: "17841xxxxxxxxx"},
		},
	},

	// ── Commerce ─────────────────────────────────────────────────────────────
	{
		ID:          "tiktok_shop",
		Label:       "TikTok Shop",
		Type:        "tiktok_shop",
		Category:    "commerce",
		Description: "Manage orders and products on TikTok Shop (also covers Tokopedia).",
		Fields: []ConfigField{
			{Key: "app_key", Label: "App Key", Placeholder: "xxxxxxxxxxxxxxxx"},
			{Key: "app_secret", Label: "App Secret", Placeholder: "xxxxxxxxxxxxxxxx", Secret: true},
		},
	},
	{
		ID:          "shopee",
		Label:       "Shopee",
		Type:        "shopee",
		Category:    "commerce",
		Description: "Manage orders, products, and promotions via the Shopee Open Platform.",
		Fields: []ConfigField{
			{Key: "partner_id", Label: "Partner ID", Placeholder: "1234567"},
			{Key: "partner_key", Label: "Partner Key", Placeholder: "xxxxxxxx…", Secret: true},
			{Key: "shop_id", Label: "Shop ID", Placeholder: "1234567"},
		},
	},

	// ── Payments ─────────────────────────────────────────────────────────────
	{
		ID:          "xendit",
		Label:       "Xendit",
		Type:        "xendit",
		Category:    "payments",
		Description: "Accept GoPay, OVO, DANA, QRIS, and cards via one payment aggregator.",
		Fields: []ConfigField{
			{Key: "secret_key", Label: "Secret Key", Placeholder: "xnd_production_xxxxxx…", Secret: true},
		},
	},
	{
		ID:          "midtrans",
		Label:       "Midtrans",
		Type:        "midtrans",
		Category:    "payments",
		Description: "Accept GoPay, OVO, DANA, QRIS, and cards. Owned by GoTo group.",
		Fields: []ConfigField{
			{Key: "server_key", Label: "Server Key", Placeholder: "Mid-server-xxxxxx…", Secret: true},
			{Key: "client_key", Label: "Client Key", Placeholder: "Mid-client-xxxxxx…"},
		},
	},

	// ── Utilities ────────────────────────────────────────────────────────────
	{
		ID:          "google_calendar",
		Label:       "Google Calendar",
		Type:        "google_calendar",
		Category:    "utilities",
		Description: "Manage appointments and availability. Ideal for service businesses.",
		Fields: []ConfigField{
			{Key: "client_id", Label: "OAuth Client ID", Placeholder: "xxxxxx.apps.googleusercontent.com"},
			{Key: "client_secret", Label: "OAuth Client Secret", Placeholder: "GOCSPX-xxxxxx…", Secret: true},
		},
	},
}

// IntegrationGroup is a labeled group of integrations for display.
type IntegrationGroup struct {
	Category     string
	Integrations []Integration
}

// CatalogByCategory returns the catalog grouped by category in display order.
func CatalogByCategory() []IntegrationGroup {
	order := []string{"messaging", "commerce", "payments", "utilities"}
	labels := map[string]string{
		"messaging":  "Messaging",
		"commerce":   "Commerce",
		"payments":   "Payments",
		"utilities":  "Utilities",
	}
	index := make(map[string][]Integration)
	for _, integ := range Catalog {
		index[integ.Category] = append(index[integ.Category], integ)
	}
	var groups []IntegrationGroup
	for _, cat := range order {
		if items, ok := index[cat]; ok {
			groups = append(groups, IntegrationGroup{Category: labels[cat], Integrations: items})
		}
	}
	return groups
}

// CatalogByID returns a map of integration ID → Integration for fast lookup.
func CatalogByID() map[string]Integration {
	m := make(map[string]Integration, len(Catalog))
	for _, integ := range Catalog {
		m[integ.ID] = integ
	}
	return m
}

// CatalogByType returns a map of integration type → Integration for fast lookup.
func CatalogByType() map[string]Integration {
	m := make(map[string]Integration, len(Catalog))
	for _, integ := range Catalog {
		m[integ.Type] = integ
	}
	return m
}
