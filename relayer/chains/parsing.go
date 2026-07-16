package chains

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// IbcMessage is the type used for parsing all possible properties of IBC messages
type IbcMessage struct {
	EventType string
	Info      ibcMessageInfo
}

type ibcMessageInfo interface {
	ParseAttrs(log *zap.Logger, attrs []sdk.Attribute)
	MarshalLogObject(enc zapcore.ObjectEncoder) error
}

// IbcMessagesFromEvents parses all events within a transaction to find IBC messages
func IbcMessagesFromEvents(
	log *zap.Logger,
	events []abci.Event,
	chainID string,
	height uint64,
) []IbcMessage {
	batch := ParseIBCEventBatch(log, events, IBCEventMetadata{ChainID: chainID, Height: height})
	logIBCEventIssues(log, batch.Issues)
	return batch.ClassicMessages
}

// ParseIBCEventBatch preserves raw event evidence while keeping v2 packet
// observations separate from the existing Classic runtime.
func ParseIBCEventBatch(log *zap.Logger, events []abci.Event, metadata IBCEventMetadata) IBCEventBatch {
	correlations, issues := correlateMessageActions(events)
	attachIssueEvidence(issues, events, metadata)
	batch := IBCEventBatch{Issues: issues}
	for i, event := range events {
		ingestIBCEvent(log, event, i, correlations[i], metadata, &batch)
	}
	return batch
}

func ingestIBCEvent(
	log *zap.Logger,
	event abci.Event,
	eventIndex int,
	correlation eventCorrelation,
	metadata IBCEventMetadata,
	batch *IBCEventBatch,
) {
	attributes := cloneABCIAttributes(event.Attributes)
	if isPacketEventType(event.Type) {
		ingestPacketEvent(log, event, eventIndex, correlation, metadata, attributes, batch)
		return
	}
	ingestClassicEvent(log, event, eventIndex, correlation, metadata, attributes, batch)
}

func ingestPacketEvent(
	log *zap.Logger,
	event abci.Event,
	eventIndex int,
	correlation eventCorrelation,
	metadata IBCEventMetadata,
	attributes []protocol.EventAttribute,
	batch *IBCEventBatch,
) {
	packetProtocol, err := classifyPacketEvent(event.Type, attributes)
	evidence := makeEventEnvelope(event.Type, eventIndex, packetProtocol, correlation.action, attributes, metadata)
	if correlation.err != nil {
		batch.addIssueWithEvent(eventIndex, event.Type, correlation.err, evidence)
	}
	if err != nil {
		batch.addIssueWithEvent(eventIndex, event.Type, err, evidence)
		return
	}
	envelope := evidence
	validEnvelope := batch.addEnvelope(envelope, eventIndex)
	if packetProtocol == protocol.ProtocolClassic {
		batch.addClassicMessage(parseIBCMessageFromEvent(log, sdk.StringifyEvent(event), metadata.ChainID, metadata.Height))
		return
	}
	if correlation.err == nil && validEnvelope {
		batch.addV2Packet(envelope, eventIndex)
	}
}

func ingestClassicEvent(
	log *zap.Logger,
	event abci.Event,
	eventIndex int,
	correlation eventCorrelation,
	metadata IBCEventMetadata,
	attributes []protocol.EventAttribute,
	batch *IBCEventBatch,
) {
	message := parseIBCMessageFromEvent(log, sdk.StringifyEvent(event), metadata.ChainID, metadata.Height)
	if message == nil || message.Info == nil {
		return
	}
	batch.ClassicMessages = append(batch.ClassicMessages, *message)
	envelope := makeEventEnvelope(event.Type, eventIndex, protocol.ProtocolClassic, correlation.action, attributes, metadata)
	batch.addEnvelope(envelope, eventIndex)
	if correlation.err != nil {
		batch.addIssueWithEvent(eventIndex, event.Type, correlation.err, envelope)
	}
}

func makeEventEnvelope(
	eventType string,
	eventIndex int,
	eventProtocol protocol.Protocol,
	action protocol.MessageAction,
	attributes []protocol.EventAttribute,
	metadata IBCEventMetadata,
) protocol.EventEnvelope {
	return protocol.EventEnvelope{
		Protocol:   eventProtocol,
		Type:       eventType,
		Height:     metadata.Height,
		TxHash:     metadata.TxHash,
		EventIndex: uint32(eventIndex),
		Action:     action,
		Attributes: attributes,
	}
}

func (batch *IBCEventBatch) addEnvelope(envelope protocol.EventEnvelope, eventIndex int) bool {
	if err := envelope.Validate(); err != nil {
		batch.addIssueWithEvent(eventIndex, envelope.Type, err, envelope)
		return false
	}
	batch.Envelopes = append(batch.Envelopes, envelope)
	return true
}

func (batch *IBCEventBatch) addClassicMessage(message *IbcMessage) {
	if message != nil && message.Info != nil {
		batch.ClassicMessages = append(batch.ClassicMessages, *message)
	}
}

func (batch *IBCEventBatch) addV2Packet(envelope protocol.EventEnvelope, eventIndex int) {
	packet, err := parseV2PacketEvent(envelope)
	if err != nil {
		batch.addIssueWithEvent(eventIndex, envelope.Type, err, envelope)
		return
	}
	batch.V2Packets = append(batch.V2Packets, packet)
}

func (batch *IBCEventBatch) addIssueWithEvent(
	eventIndex int,
	eventType string,
	err error,
	event protocol.EventEnvelope,
) {
	batch.Issues = append(batch.Issues, newIBCEventIssueWithEvent(eventIndex, eventType, err, event))
}

func attachIssueEvidence(issues []IBCEventIssue, events []abci.Event, metadata IBCEventMetadata) {
	for i := range issues {
		eventIndex := int(issues[i].EventIndex)
		if eventIndex >= len(events) {
			continue
		}
		raw := events[eventIndex]
		evidence := makeEventEnvelope(raw.Type, eventIndex, protocol.ProtocolUnspecified, protocol.MessageAction{}, cloneABCIAttributes(raw.Attributes), metadata)
		issues[i].Event = &evidence
	}
}

func logIBCEventIssues(log *zap.Logger, issues []IBCEventIssue) {
	if log == nil {
		return
	}
	for _, issue := range issues {
		log.Debug("IBC event ingestion issue",
			zap.Uint32("event_index", issue.EventIndex),
			zap.String("event_type", issue.EventType),
			zap.Error(issue.Err),
		)
	}
}

type messageInfo interface {
	ibcMessageInfo
	ParseAttrs(log *zap.Logger, attrs []sdk.Attribute)
}

func ParseIBCMessageFromEvent(
	log *zap.Logger,
	event sdk.StringEvent,
	chainID string,
	height uint64,
) *IbcMessage {
	if !isClassicPacketStringEvent(event) {
		return nil
	}
	return parseIBCMessageFromEvent(log, event, chainID, height)
}

func isClassicPacketStringEvent(event sdk.StringEvent) bool {
	if !isPacketEventType(event.Type) {
		return true
	}
	packetProtocol, err := classifyPacketEvent(event.Type, protocolAttributes(event.Attributes))
	return err == nil && packetProtocol == protocol.ProtocolClassic
}

func protocolAttributes(attributes []sdk.Attribute) []protocol.EventAttribute {
	converted := make([]protocol.EventAttribute, len(attributes))
	for i, attribute := range attributes {
		converted[i] = protocol.EventAttribute{Key: attribute.Key, Value: attribute.Value}
	}
	return converted
}

func parseIBCMessageFromEvent(
	log *zap.Logger,
	event sdk.StringEvent,
	chainID string,
	height uint64,
) *IbcMessage {
	msgInfo := packetOrChannelMessageInfo(event.Type, height)
	if msgInfo == nil {
		msgInfo = connectionOrClientMessageInfo(event.Type, height)
	}
	if msgInfo == nil {
		msgInfo = clientICQMessageInfo(event.Type, chainID, height)
	}
	if msgInfo == nil {
		return nil
	}
	msgInfo.ParseAttrs(log, event.Attributes)
	return &IbcMessage{
		EventType: event.Type,
		Info:      msgInfo,
	}
}

func packetOrChannelMessageInfo(eventType string, height uint64) messageInfo {
	switch eventType {
	case chantypes.EventTypeSendPacket, chantypes.EventTypeRecvPacket, chantypes.EventTypeWriteAck,
		chantypes.EventTypeAcknowledgePacket, chantypes.EventTypeTimeoutPacket:
		return &PacketInfo{Height: height}
	case chantypes.EventTypeChannelOpenInit, chantypes.EventTypeChannelOpenTry,
		chantypes.EventTypeChannelOpenAck, chantypes.EventTypeChannelOpenConfirm,
		chantypes.EventTypeChannelCloseInit, chantypes.EventTypeChannelClosed, chantypes.EventTypeChannelCloseConfirm:
		return &ChannelInfo{Height: height}
	default:
		return nil
	}
}

func connectionOrClientMessageInfo(eventType string, height uint64) messageInfo {
	switch eventType {
	case conntypes.EventTypeConnectionOpenInit, conntypes.EventTypeConnectionOpenTry,
		conntypes.EventTypeConnectionOpenAck, conntypes.EventTypeConnectionOpenConfirm:
		return &ConnectionInfo{Height: height}
	case clienttypes.EventTypeCreateClient, clienttypes.EventTypeUpdateClient,
		clienttypes.EventTypeUpgradeClient, clienttypes.EventTypeSubmitMisbehaviour:
		return new(ClientInfo)
	default:
		return nil
	}
}

func clientICQMessageInfo(eventType, chainID string, height uint64) messageInfo {
	if eventType != string(processor.ClientICQTypeRequest) && eventType != string(processor.ClientICQTypeResponse) {
		return nil
	}
	return &ClientICQInfo{Height: height, Source: chainID}
}

func (msg *IbcMessage) parseIBCPacketReceiveMessageFromEvent(
	log *zap.Logger,
	event sdk.StringEvent,
	chainID string,
	height uint64,
) *IbcMessage {
	var pi *PacketInfo
	if msg.Info == nil {
		pi = &PacketInfo{Height: height}
		msg.Info = pi
	} else {
		pi = msg.Info.(*PacketInfo)
	}
	pi.ParseAttrs(log, event.Attributes)
	if event.Type != chantypes.EventTypeWriteAck {
		msg.EventType = event.Type
	}
	return msg
}

// ClientInfo contains the consensus height of the counterparty chain for a client.
type ClientInfo struct {
	ClientID        string
	ConsensusHeight clienttypes.Height
	Header          []byte
}

func NewClientInfo(
	clientID string,
	consensusHeight clienttypes.Height,
	header []byte,
) *ClientInfo {
	return &ClientInfo{
		clientID, consensusHeight, header,
	}
}

func (c ClientInfo) ClientState(trustingPeriod time.Duration) provider.ClientState {
	return provider.ClientState{
		ClientID:        c.ClientID,
		ConsensusHeight: c.ConsensusHeight,
		TrustingPeriod:  trustingPeriod,
		Header:          c.Header,
	}
}

func (res *ClientInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("client_id", res.ClientID)
	enc.AddUint64("consensus_height", res.ConsensusHeight.RevisionHeight)
	enc.AddUint64("consensus_height_revision", res.ConsensusHeight.RevisionNumber)
	return nil
}

func (res *ClientInfo) ParseAttrs(log *zap.Logger, attributes []sdk.Attribute) {
	for _, attr := range attributes {
		res.parseClientAttribute(log, attr)
	}
}

func (res *ClientInfo) parseClientAttribute(log *zap.Logger, attr sdk.Attribute) {
	switch attr.Key {
	case clienttypes.AttributeKeyClientID:
		res.ClientID = attr.Value
	case clienttypes.AttributeKeyConsensusHeight:
		revisionSplit := strings.Split(attr.Value, "-")
		if len(revisionSplit) != 2 {
			log.Error("Error parsing client consensus height",
				zap.String("client_id", res.ClientID),
				zap.String("value", attr.Value),
			)
			return
		}
		revisionNumberString := revisionSplit[0]
		revisionNumber, err := strconv.ParseUint(revisionNumberString, 10, 64)
		if err != nil {
			log.Error("Error parsing client consensus height revision number",
				zap.Error(err),
			)
			return
		}
		revisionHeightString := revisionSplit[1]
		revisionHeight, err := strconv.ParseUint(revisionHeightString, 10, 64)
		if err != nil {
			log.Error("Error parsing client consensus height revision height",
				zap.Error(err),
			)
			return
		}
		res.ConsensusHeight = clienttypes.Height{
			RevisionNumber: revisionNumber,
			RevisionHeight: revisionHeight,
		}
	case legacyAttributeKeyHeader:
		data, err := hex.DecodeString(attr.Value)
		if err != nil {
			log.Error("Error parsing client header",
				zap.String("header", attr.Value),
				zap.Error(err),
			)
			return
		}
		res.Header = data
	}
}

// alias type to the provider types, used for adding parser methods
type PacketInfo provider.PacketInfo

func (res *PacketInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddUint64("sequence", res.Sequence)
	enc.AddString("src_channel", res.SourceChannel)
	enc.AddString("src_port", res.SourcePort)
	enc.AddString("dst_channel", res.DestChannel)
	enc.AddString("dst_port", res.DestPort)
	return nil
}

// parsePacketInfo is treated differently from the others since it can be constructed from the accumulation of multiple events
func (res *PacketInfo) ParseAttrs(log *zap.Logger, attrs []sdk.Attribute) {
	for _, attr := range attrs {
		res.parsePacketAttribute(log, attr)
	}
}

func (res *PacketInfo) parsePacketAttribute(log *zap.Logger, attr sdk.Attribute) {
	var err error
	switch attr.Key {
	case chantypes.AttributeKeySequence:
		res.Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
		if err != nil {
			log.Error("Error parsing packet sequence",
				zap.String("value", attr.Value),
				zap.Error(err),
			)
			return
		}
	case chantypes.AttributeKeyTimeoutTimestamp:
		res.TimeoutTimestamp, err = strconv.ParseUint(attr.Value, 10, 64)
		if err != nil {
			log.Error("Error parsing packet timestamp",
				zap.Uint64("sequence", res.Sequence),
				zap.String("value", attr.Value),
				zap.Error(err),
			)
			return
		}
	// NOTE: deprecated per IBC spec
	case legacyAttributeKeyPacketData:
		res.Data = []byte(attr.Value)
	case chantypes.AttributeKeyDataHex:
		data, err := hex.DecodeString(attr.Value)
		if err != nil {
			log.Error("Error parsing packet data",
				zap.Uint64("sequence", res.Sequence),
				zap.Error(err),
			)
			return
		}
		res.Data = data
	// NOTE: deprecated per IBC spec
	case legacyAttributeKeyPacketAck:
		res.Ack = []byte(attr.Value)
	case chantypes.AttributeKeyAckHex:
		data, err := hex.DecodeString(attr.Value)
		if err != nil {
			log.Error("Error parsing packet ack",
				zap.Uint64("sequence", res.Sequence),
				zap.String("value", attr.Value),
				zap.Error(err),
			)
			return
		}
		res.Ack = data
	case chantypes.AttributeKeyTimeoutHeight:
		timeoutSplit := strings.Split(attr.Value, "-")
		if len(timeoutSplit) != 2 {
			log.Error("Error parsing packet height timeout",
				zap.Uint64("sequence", res.Sequence),
				zap.String("value", attr.Value),
			)
			return
		}
		revisionNumber, err := strconv.ParseUint(timeoutSplit[0], 10, 64)
		if err != nil {
			log.Error("Error parsing packet timeout height revision number",
				zap.Uint64("sequence", res.Sequence),
				zap.String("value", timeoutSplit[0]),
				zap.Error(err),
			)
			return
		}
		revisionHeight, err := strconv.ParseUint(timeoutSplit[1], 10, 64)
		if err != nil {
			log.Error("Error parsing packet timeout height revision height",
				zap.Uint64("sequence", res.Sequence),
				zap.String("value", timeoutSplit[1]),
				zap.Error(err),
			)
			return
		}
		res.TimeoutHeight = clienttypes.Height{
			RevisionNumber: revisionNumber,
			RevisionHeight: revisionHeight,
		}
	case chantypes.AttributeKeySrcPort:
		res.SourcePort = attr.Value
	case chantypes.AttributeKeySrcChannel:
		res.SourceChannel = attr.Value
	case chantypes.AttributeKeyDstPort:
		res.DestPort = attr.Value
	case chantypes.AttributeKeyDstChannel:
		res.DestChannel = attr.Value
	case chantypes.AttributeKeyChannelOrdering:
		res.ChannelOrder = attr.Value
	}
}

// alias type to the provider types, used for adding parser methods
type ChannelInfo provider.ChannelInfo

func (res *ChannelInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("channel_id", res.ChannelID)
	enc.AddString("port_id", res.PortID)
	enc.AddString("counterparty_channel_id", res.CounterpartyChannelID)
	enc.AddString("counterparty_port_id", res.CounterpartyPortID)
	return nil
}

func (res *ChannelInfo) ParseAttrs(log *zap.Logger, attrs []sdk.Attribute) {
	for _, attr := range attrs {
		res.parseChannelAttribute(attr)
	}
}

// parseChannelAttribute parses channel attributes from an event.
// If the attribute has already been parsed into the channelInfo,
// it will not overwrite, and return true to inform the caller that
// the attribute already exists.
func (res *ChannelInfo) parseChannelAttribute(attr sdk.Attribute) {
	switch attr.Key {
	case chantypes.AttributeKeyPortID:
		res.PortID = attr.Value
	case chantypes.AttributeKeyChannelID:
		res.ChannelID = attr.Value
	case chantypes.AttributeCounterpartyPortID:
		res.CounterpartyPortID = attr.Value
	case chantypes.AttributeCounterpartyChannelID:
		res.CounterpartyChannelID = attr.Value
	case chantypes.AttributeKeyConnectionID:
		res.ConnID = attr.Value
	case chantypes.AttributeKeyVersion:
		res.Version = attr.Value
	}
}

// alias type to the provider types, used for adding parser methods
type ConnectionInfo provider.ConnectionInfo

func (res *ConnectionInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("connection_id", res.ConnID)
	enc.AddString("client_id", res.ClientID)
	enc.AddString("counterparty_connection_id", res.CounterpartyConnID)
	enc.AddString("counterparty_client_id", res.CounterpartyClientID)
	return nil
}

func (res *ConnectionInfo) ParseAttrs(log *zap.Logger, attrs []sdk.Attribute) {
	for _, attr := range attrs {
		res.parseConnectionAttribute(attr)
	}
}

func (res *ConnectionInfo) parseConnectionAttribute(attr sdk.Attribute) {
	switch attr.Key {
	case conntypes.AttributeKeyConnectionID:
		res.ConnID = attr.Value
	case conntypes.AttributeKeyClientID:
		res.ClientID = attr.Value
	case conntypes.AttributeKeyCounterpartyConnectionID:
		res.CounterpartyConnID = attr.Value
	case conntypes.AttributeKeyCounterpartyClientID:
		res.CounterpartyClientID = attr.Value
	}
}

type ClientICQInfo struct {
	Source     string
	Connection string
	Chain      string
	QueryID    provider.ClientICQQueryID
	Type       string
	Request    []byte
	Height     uint64
}

func (res *ClientICQInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("connection_id", res.Connection)
	enc.AddString("chain_id", res.Chain)
	enc.AddString("query_id", string(res.QueryID))
	enc.AddString("type", res.Type)
	enc.AddString("request", hex.EncodeToString(res.Request))
	enc.AddUint64("height", res.Height)

	return nil
}

func (res *ClientICQInfo) ParseAttrs(log *zap.Logger, attrs []sdk.Attribute) {
	for _, attr := range attrs {
		if err := res.parseAttribute(attr); err != nil {
			panic(fmt.Errorf("failed to parse attributes from client ICQ message: %w", err))
		}
	}
}

func (res *ClientICQInfo) parseAttribute(attr sdk.Attribute) (err error) {
	switch attr.Key {
	case "connection_id":
		res.Connection = attr.Value
	case "chain_id":
		res.Chain = attr.Value
	case "query_id":
		res.QueryID = provider.ClientICQQueryID(attr.Value)
	case "type":
		res.Type = attr.Value
	case "request":
		res.Request, err = hex.DecodeString(attr.Value)
		if err != nil {
			return err
		}
	case "height":
		res.Height, err = strconv.ParseUint(attr.Value, 10, 64)
		if err != nil {
			return err
		}
	}
	return nil
}
