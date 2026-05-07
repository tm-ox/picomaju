package payment

// Provider identifies a payment processor.
type Provider string

const (
	ProviderStripe Provider = "stripe"
	ProviderXendit Provider = "xendit"
)

// CreditPack is a one-time credit purchase option.
type CreditPack struct {
	ID          string
	Credits     int
	USDCents    int    // e.g. 500 = $5.00
	IDRAmount   int    // in IDR
	Label       string
	Description string
}

// Plan is a recurring subscription option.
type Plan struct {
	ID             string
	Label          string
	CreditsPerMonth int    // 0 = unlimited
	USDCents       int    // monthly
	IDRAmount      int    // monthly in IDR
	StripePriceID  string // populated once Stripe products are created
	XenditPlanID   string // populated once Xendit plans are created
}

var CreditPacks = []CreditPack{
	{ID: "credits_100", Credits: 100, USDCents: 500, IDRAmount: 80000, Label: "100 credits", Description: "Good for ~100 agent messages"},
	{ID: "credits_500", Credits: 500, USDCents: 2000, IDRAmount: 320000, Label: "500 credits", Description: "Good for ~500 agent messages"},
	{ID: "credits_1500", Credits: 1500, USDCents: 5000, IDRAmount: 790000, Label: "1,500 credits", Description: "Good for ~1,500 agent messages"},
}

var Plans = []Plan{
	{ID: "starter", Label: "Starter", CreditsPerMonth: 300, USDCents: 1200, IDRAmount: 190000},
	{ID: "pro", Label: "Pro", CreditsPerMonth: 0, USDCents: 2900, IDRAmount: 460000},
}

func CreditPackByID(id string) (CreditPack, bool) {
	for _, p := range CreditPacks {
		if p.ID == id {
			return p, true
		}
	}
	return CreditPack{}, false
}

func PlanByID(id string) (Plan, bool) {
	for _, p := range Plans {
		if p.ID == id {
			return p, true
		}
	}
	return Plan{}, false
}
