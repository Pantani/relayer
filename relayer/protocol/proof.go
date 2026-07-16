package protocol

import "fmt"

// ProofKind identifies the state proven for a packet message.
type ProofKind string

const (
	ProofPacketCommitment ProofKind = "packet_commitment"
	ProofAcknowledgement  ProofKind = "acknowledgement"
	ProofReceiptAbsence   ProofKind = "receipt_absence"
	ProofNextSequenceRecv ProofKind = "next_sequence_receive"
)

// ProofEnvelope contains proof bytes with a dependency-neutral height.
type ProofEnvelope struct {
	Protocol Protocol
	Kind     ProofKind
	Height   Height
	Data     []byte
}

// Validate checks proof shape and protocol-specific proof kinds.
func (p ProofEnvelope) Validate() error {
	if err := p.Protocol.Validate(); err != nil {
		return err
	}
	if err := p.Height.Validate(); err != nil {
		return err
	}
	if len(p.Data) == 0 {
		return fmt.Errorf("proof data cannot be empty")
	}
	if p.Protocol == ProtocolV2 {
		return validateV2ProofKind(p.Kind)
	}
	return validateClassicProofKind(p.Kind)
}

// Clone returns an independent proof envelope.
func (p ProofEnvelope) Clone() ProofEnvelope {
	clone := p
	clone.Data = cloneBytes(p.Data)
	return clone
}

func validateClassicProofKind(kind ProofKind) error {
	switch kind {
	case ProofPacketCommitment, ProofAcknowledgement, ProofReceiptAbsence, ProofNextSequenceRecv:
		return nil
	default:
		return fmt.Errorf("classic proof kind %q is not supported", kind)
	}
}

func validateV2ProofKind(kind ProofKind) error {
	switch kind {
	case ProofPacketCommitment, ProofAcknowledgement, ProofReceiptAbsence:
		return nil
	default:
		return fmt.Errorf("v2 proof kind %q is not supported", kind)
	}
}
