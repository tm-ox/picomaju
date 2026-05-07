package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const xenditBaseURL = "https://api.xendit.co"

type xenditInvoiceRequest struct {
	ExternalID  string            `json:"external_id"`
	Amount      int               `json:"amount"`
	Currency    string            `json:"currency"`
	Description string            `json:"description"`
	SuccessURL  string            `json:"success_redirect_url"`
	FailureURL  string            `json:"failure_redirect_url"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type xenditInvoiceResponse struct {
	ID         string `json:"id"`
	InvoiceURL string `json:"invoice_url"`
	ExternalID string `json:"external_id"`
	Status     string `json:"status"`
}

// XenditCheckoutURL creates a Xendit invoice and returns the hosted payment URL.
func XenditCheckoutURL(cfg Config, packID, planID string) (string, error) {
	var amount int
	var description string
	var externalID string
	metadata := map[string]string{}

	if packID != "" {
		pack, ok := CreditPackByID(packID)
		if !ok {
			return "", fmt.Errorf("unknown credit pack %q", packID)
		}
		amount = pack.IDRAmount
		description = pack.Label + " — PicoMaju"
		externalID = fmt.Sprintf("pm-credits-%s-%d", packID, time.Now().Unix())
		metadata["type"] = "credits"
		metadata["pack_id"] = packID
		metadata["credits"] = fmt.Sprintf("%d", pack.Credits)
	} else if planID != "" {
		plan, ok := PlanByID(planID)
		if !ok {
			return "", fmt.Errorf("unknown plan %q", planID)
		}
		amount = plan.IDRAmount
		description = plan.Label + " subscription — PicoMaju"
		externalID = fmt.Sprintf("pm-plan-%s-%d", planID, time.Now().Unix())
		metadata["type"] = "subscription"
		metadata["plan_id"] = planID
	} else {
		return "", fmt.Errorf("packID or planID required")
	}

	payload := xenditInvoiceRequest{
		ExternalID:  externalID,
		Amount:      amount,
		Currency:    "IDR",
		Description: description,
		SuccessURL:  cfg.BaseURL + "/license/checkout/success",
		FailureURL:  cfg.BaseURL + "/license",
		Metadata:    metadata,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", xenditBaseURL+"/v2/invoices", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(cfg.XenditAPIKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("xendit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("xendit error: HTTP %d", resp.StatusCode)
	}

	var inv xenditInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&inv); err != nil {
		return "", fmt.Errorf("xendit response: %w", err)
	}
	return inv.InvoiceURL, nil
}
