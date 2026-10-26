package domain

import "time"

// AccountEventType — событие банка по счёту клиента.
type AccountEventType string

const (
	EventArrestImposed AccountEventType = "arrest_imposed" // счёт арестован
	EventArrestLifted  AccountEventType = "arrest_lifted"  // арест снят
)

// AccountEvent — событие от банка. ID уникален: повтор события ничего не меняет.
type AccountEvent struct {
	ID         string           `json:"id"`
	ClientID   string           `json:"client_id"`
	Type       AccountEventType `json:"type"`
	OccurredAt time.Time        `json:"occurred_at"`
}
