package model

type JSONBatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type JSONBatchResponse []JSONBatchResponseItem
