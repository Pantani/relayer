package chains

import (
	"errors"
	"fmt"
	"strconv"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/relayer/v2/relayer/protocol"
)

const messageIndexAttribute = "msg_index"

var (
	ErrInvalidMessageIndex      = errors.New("invalid message index")
	ErrConflictingMessageIndex  = errors.New("conflicting message index")
	ErrConflictingMessageAction = errors.New("conflicting message action")
	ErrMessageActionRequired    = errors.New("message action required")
)

// IBCEventMetadata identifies the block and optional transaction containing an
// event batch.
type IBCEventMetadata struct {
	ChainID string
	Height  uint64
	TxHash  string
}

// IBCEventBatch separates the existing Classic runtime input from observed v2
// packets. V2Packets is intentionally not consumed by a chain processor yet.
type IBCEventBatch struct {
	Envelopes       []protocol.EventEnvelope
	ClassicMessages []IbcMessage
	V2Packets       []V2PacketEvent
	Issues          []IBCEventIssue
}

// V2PacketEvent holds the lossless event and its decoded protocol-neutral
// packet. Acknowledgement is present only for write_acknowledgement.
type V2PacketEvent struct {
	Event           protocol.EventEnvelope
	Observation     protocol.PacketObservation
	Acknowledgement *protocol.Acknowledgement
}

// IBCEventIssue describes why a raw event could not be correlated or parsed.
type IBCEventIssue struct {
	EventIndex uint32
	EventType  string
	Err        error
	// Event is diagnostic evidence and may use ProtocolUnspecified when
	// classification itself failed. It is never added to IBCEventBatch.Envelopes.
	Event *protocol.EventEnvelope
}

type eventCorrelation struct {
	action protocol.MessageAction
	err    error
}

type messageActionIndex struct {
	actions   map[uint32]string
	conflicts map[uint32]struct{}
}

type indexedActionCandidate struct {
	index   uint32
	action  string
	present bool
	invalid bool
	err     error
}

type legacyActionState struct {
	action    protocol.MessageAction
	err       error
	nextIndex uint32
}

func correlateMessageActions(events []abci.Event) ([]eventCorrelation, []IBCEventIssue) {
	actions, issues := buildMessageActionIndex(events)
	correlations := make([]eventCorrelation, len(events))
	legacy := legacyActionState{}
	for i, event := range events {
		legacy.observe(event)
		correlations[i] = actions.resolve(event, legacy.correlation())
	}
	return correlations, issues
}

func buildMessageActionIndex(events []abci.Event) (messageActionIndex, []IBCEventIssue) {
	index := newMessageActionIndex()
	issues := make([]IBCEventIssue, 0)
	for i, event := range events {
		candidate := indexedMessageAction(event)
		if candidate.invalid {
			index.invalidate(candidate.index)
		}
		if candidate.err != nil {
			issues = append(issues, newIBCEventIssue(i, event.Type, candidate.err))
			continue
		}
		if candidate.present {
			index.add(candidate.index, candidate.action)
		}
	}
	return index, issues
}

func newMessageActionIndex() messageActionIndex {
	return messageActionIndex{
		actions:   make(map[uint32]string),
		conflicts: make(map[uint32]struct{}),
	}
}

func indexedMessageAction(event abci.Event) indexedActionCandidate {
	if event.Type != sdk.EventTypeMessage {
		return indexedActionCandidate{}
	}
	action, actionPresent, actionErr := uniqueABCIAttribute(event.Attributes, sdk.AttributeKeyAction, ErrConflictingMessageAction)
	if !actionPresent {
		return indexedActionCandidate{}
	}
	index, indexed, indexErr := messageIndex(event.Attributes)
	if indexErr != nil {
		return indexedActionCandidate{err: indexErr}
	}
	if actionErr != nil {
		return indexedActionCandidate{index: index, invalid: indexed, err: actionErr}
	}
	if action == "" {
		return indexedActionCandidate{index: index, invalid: indexed, err: fmt.Errorf("%w: empty action", ErrMessageActionRequired)}
	}
	if !indexed {
		return indexedActionCandidate{}
	}
	return indexedActionCandidate{index: index, action: action, present: true}
}

func (index messageActionIndex) add(messageIndex uint32, action string) {
	if _, conflicted := index.conflicts[messageIndex]; conflicted {
		return
	}
	current, exists := index.actions[messageIndex]
	if exists && current != action {
		delete(index.actions, messageIndex)
		index.conflicts[messageIndex] = struct{}{}
		return
	}
	index.actions[messageIndex] = action
}

func (index messageActionIndex) invalidate(messageIndex uint32) {
	delete(index.actions, messageIndex)
	index.conflicts[messageIndex] = struct{}{}
}

func (index messageActionIndex) resolve(event abci.Event, legacy eventCorrelation) eventCorrelation {
	messageIndex, present, err := messageIndex(event.Attributes)
	if err != nil {
		return eventCorrelation{err: err}
	}
	if !present {
		return legacy
	}
	if _, conflicted := index.conflicts[messageIndex]; conflicted {
		return eventCorrelation{err: fmt.Errorf("%w: index %d", ErrConflictingMessageAction, messageIndex)}
	}
	action, found := index.actions[messageIndex]
	if !found {
		return eventCorrelation{err: fmt.Errorf("%w: index %d", ErrMessageActionRequired, messageIndex)}
	}
	return eventCorrelation{action: protocol.MessageAction{Present: true, Index: messageIndex, Type: action}}
}

func (state *legacyActionState) observe(event abci.Event) {
	if event.Type != sdk.EventTypeMessage || hasABCIAttribute(event.Attributes, messageIndexAttribute) {
		return
	}
	action, present, err := uniqueABCIAttribute(event.Attributes, sdk.AttributeKeyAction, ErrConflictingMessageAction)
	if !present {
		return
	}
	messageIndex := state.nextIndex
	state.nextIndex++
	if err != nil {
		state.poison(err)
		return
	}
	if action == "" {
		state.poison(ErrMessageActionRequired)
		return
	}
	state.action = protocol.MessageAction{Present: true, Index: messageIndex, Type: action}
	state.err = nil
}

func (state *legacyActionState) poison(err error) {
	state.action = protocol.MessageAction{}
	state.err = err
}

func (state legacyActionState) correlation() eventCorrelation {
	return eventCorrelation{action: state.action, err: state.err}
}

func messageIndex(attributes []abci.EventAttribute) (uint32, bool, error) {
	value, present, err := uniqueABCIAttribute(attributes, messageIndexAttribute, ErrConflictingMessageIndex)
	if err != nil || !present {
		return 0, present, err
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, true, ErrInvalidMessageIndex
	}
	return uint32(parsed), true, nil
}

func uniqueABCIAttribute(attributes []abci.EventAttribute, key string, conflict error) (string, bool, error) {
	value := ""
	present := false
	for _, attribute := range attributes {
		if attribute.Key != key {
			continue
		}
		if present && value != attribute.Value {
			return "", true, fmt.Errorf("%w: key %q", conflict, key)
		}
		value, present = attribute.Value, true
	}
	return value, present, nil
}

func hasABCIAttribute(attributes []abci.EventAttribute, key string) bool {
	for _, attribute := range attributes {
		if attribute.Key == key {
			return true
		}
	}
	return false
}

func cloneABCIAttributes(attributes []abci.EventAttribute) []protocol.EventAttribute {
	cloned := make([]protocol.EventAttribute, len(attributes))
	for i, attribute := range attributes {
		cloned[i] = protocol.EventAttribute{Key: attribute.Key, Value: attribute.Value, Index: attribute.Index}
	}
	return cloned
}

func newIBCEventIssue(index int, eventType string, err error) IBCEventIssue {
	return IBCEventIssue{EventIndex: uint32(index), EventType: eventType, Err: err}
}

func newIBCEventIssueWithEvent(index int, eventType string, err error, event protocol.EventEnvelope) IBCEventIssue {
	owned := event.Clone()
	return IBCEventIssue{EventIndex: uint32(index), EventType: eventType, Err: err, Event: &owned}
}
