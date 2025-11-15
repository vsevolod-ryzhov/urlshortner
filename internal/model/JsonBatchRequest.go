package model

type JSONBatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type JSONBatchRequest []JSONBatchRequestItem
