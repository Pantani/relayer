package protocol

import "fmt"

// EventAttribute preserves an event attribute's order, duplicates, and index flag.
type EventAttribute struct {
	Key   string
	Value string
	Index bool
}

// MessageAction identifies the message associated with an event.
type MessageAction struct {
	Present bool
	Index   uint32
	Type    string
}

// EventEnvelope is the lossless protocol-neutral event representation.
type EventEnvelope struct {
	Protocol   Protocol
	Type       string
	Height     uint64
	TxHash     string
	EventIndex uint32
	Action     MessageAction
	Attributes []EventAttribute
}

// Validate checks the event's structural contract.
func (e EventEnvelope) Validate() error {
	if err := e.Protocol.Validate(); err != nil {
		return err
	}
	if e.Type == "" {
		return fmt.Errorf("event type cannot be empty")
	}
	if e.Action.Present && e.Action.Type == "" {
		return fmt.Errorf("present message action requires a type")
	}
	if !e.Action.Present && (e.Action.Index != 0 || e.Action.Type != "") {
		return fmt.Errorf("absent message action cannot carry an index or type")
	}
	return validateAttributes(e.Attributes)
}

// RequireAction rejects events that cannot yet be correlated to a message.
func (e EventEnvelope) RequireAction() error {
	if !e.Action.Present {
		return fmt.Errorf("event message action is required")
	}
	return nil
}

// AttributeValues returns all values for a key in their original order.
func (e EventEnvelope) AttributeValues(key string) []string {
	values := make([]string, 0)
	for _, attribute := range e.Attributes {
		if attribute.Key == key {
			values = append(values, attribute.Value)
		}
	}
	return values
}

// Clone returns an independent event envelope.
func (e EventEnvelope) Clone() EventEnvelope {
	clone := e
	clone.Attributes = append([]EventAttribute(nil), e.Attributes...)
	return clone
}

func validateAttributes(attributes []EventAttribute) error {
	for i, attribute := range attributes {
		if attribute.Key == "" {
			return fmt.Errorf("event attribute %d has an empty key", i)
		}
	}
	return nil
}
