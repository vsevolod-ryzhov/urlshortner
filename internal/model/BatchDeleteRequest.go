package model

type BatchDeleteItem string

type BatchDeleteRequest []BatchDeleteItem

type DeleteTask struct {
	UserID  string
	ShortID string
}
