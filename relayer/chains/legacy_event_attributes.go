package chains

// Legacy event attribute names were removed from ibc-go v11, but older chains
// can still emit them while a relayer is upgrading. Keep the wire names local
// so Classic event ingestion remains backwards compatible.
const (
	legacyAttributeKeyPacketData = "packet_data"
	legacyAttributeKeyPacketAck  = "packet_ack"
	legacyAttributeKeyHeader     = "header"
)
