package webhook

type WebhookArgs struct {
	EventID  string `json:"event_id"`
	LedgerID string `json:"ledger_id"`
}

func (WebhookArgs) Kind() string {
	return "webhook_delivery"
}

type WebhookEndpoint struct {
	ID, URL, Secret string
}
