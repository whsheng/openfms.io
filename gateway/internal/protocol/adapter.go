package protocol

// PacketScanner handles packet boundary detection from TCP stream
type PacketScanner interface {
	// Scan extracts complete packet from buffer
	// completePacket: the extracted packet (without escape sequences, checksum)
	// restBuffer: remaining unprocessed bytes
	Scan(buffer []byte) (completePacket []byte, restBuffer []byte, err error)
}

// ProtocolAdapter translates between binary protocol and standard message format
type ProtocolAdapter interface {
	// Decode translates binary packet to standard message
	Decode(packet []byte) (*StandardMessage, error)

	// Encode translates standard command to binary
	Encode(cmd StandardCommand) ([]byte, error)

	// IsHeartbeat checks if packet is a heartbeat
	IsHeartbeat(packet []byte) bool

	// GenerateHeartbeatAck creates heartbeat acknowledgment
	GenerateHeartbeatAck(packet []byte) ([]byte, error)

	// Protocol returns protocol identifier
	Protocol() string
}

// Detector identifies protocol type from initial bytes
type Detector interface {
	// Match detects protocol from header bytes
	// Returns adapter and true if matched
	Match(headerBytes []byte) (ProtocolAdapter, bool)
}
