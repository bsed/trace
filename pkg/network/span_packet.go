package network

// SpansPacket pinpoint data
type SpansPacket struct {
	Type     uint16   `msg:"type"` // tcp or udp
	AppName  string   `msg:"appName"`
	AgentID  string   `msg:"agentID"`
	SpanTime int64    `msg:"spanTime"`
	Payload  []*Spans `msg:"payload"`
}

// NewSpansPacket ...
func NewSpansPacket() *SpansPacket {
	return &SpansPacket{
		Payload: make([]*Spans, 0),
	}
}

// NewSpans ...
func NewSpans() *Spans {
	return &Spans{}
}

// type DataType : SpanV2 SpanChunk AgentStat AgentStatBatch
// Spans data
type Spans struct {
	Type  uint16 `msg:"type"`
	Spans []byte `msg:"spans"`
}
