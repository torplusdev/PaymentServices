package data

import "time"

type Connections struct {
	Hosts []string
}

type FrequencyContentMetadata struct {
	Cid         []byte
	Frequency   uint64
	LastUpdated time.Time
}

type PpBoomElement struct {
	Data []byte
	Freq uint64
}
