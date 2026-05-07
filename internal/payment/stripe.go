package payment

import (
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

// StripeCheckoutURL creates a Stripe Checkout session and returns the redirect URL.
// packID is a CreditPack.ID; planID is a Plan.ID. Exactly one must be non-empty.
func StripeCheckoutURL(cfg Config, packID, planID string) (string, error) {
	stripe.Key = cfg.StripeSecretKey

	successURL := cfg.BaseURL + "/license/checkout/success?session_id={CHECKOUT_SESSION_ID}"
	cancelURL := cfg.BaseURL + "/license"

	var params *stripe.CheckoutSessionParams

	if packID != "" {
		pack, ok := CreditPackByID(packID)
		if !ok {
			return "", fmt.Errorf("unknown credit pack %q", packID)
		}
		params = &stripe.CheckoutSessionParams{
			Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
			SuccessURL: stripe.String(successURL),
			CancelURL:  stripe.String(cancelURL),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Quantity: stripe.Int64(1),
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency:   stripe.String("usd"),
						UnitAmount: stripe.Int64(int64(pack.USDCents)),
						ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
							Name:        stripe.String(pack.Label),
							Description: stripe.String(pack.Description),
						},
					},
				},
			},
			Metadata: map[string]string{
				"type":     "credits",
				"pack_id":  packID,
				"credits":  fmt.Sprintf("%d", pack.Credits),
			},
		}
	} else if planID != "" {
		plan, ok := PlanByID(planID)
		if !ok {
			return "", fmt.Errorf("unknown plan %q", planID)
		}
		if plan.StripePriceID == "" {
			return "", fmt.Errorf("plan %q has no Stripe price ID configured", planID)
		}
		params = &stripe.CheckoutSessionParams{
			Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
			SuccessURL: stripe.String(successURL),
			CancelURL:  stripe.String(cancelURL),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(plan.StripePriceID),
					Quantity: stripe.Int64(1),
				},
			},
			Metadata: map[string]string{
				"type":    "subscription",
				"plan_id": planID,
			},
		}
	} else {
		return "", fmt.Errorf("packID or planID required")
	}

	s, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe checkout session: %w", err)
	}
	return s.URL, nil
}
