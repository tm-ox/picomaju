package payment

import "os"

// Config holds payment provider credentials loaded from environment variables.
type Config struct {
	StripeSecretKey      string
	StripeWebhookSecret  string
	XenditAPIKey         string
	XenditWebhookToken   string
	BaseURL              string // e.g. https://picomaju.example.com — used for Stripe redirect URLs
}

// LoadConfig reads payment config from environment variables.
func LoadConfig() Config {
	return Config{
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		XenditAPIKey:        os.Getenv("XENDIT_API_KEY"),
		XenditWebhookToken:  os.Getenv("XENDIT_WEBHOOK_TOKEN"),
		BaseURL:             os.Getenv("PICOMAJU_BASE_URL"),
	}
}

func (c Config) StripeConfigured() bool {
	return c.StripeSecretKey != "" && c.StripeWebhookSecret != ""
}

func (c Config) XenditConfigured() bool {
	return c.XenditAPIKey != "" && c.XenditWebhookToken != ""
}
