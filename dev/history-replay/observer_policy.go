//go:build linux && amd64

package main

import (
	"errors"
	"time"
)

// ObserverPolicy extends the unchanged workflow protocol with frozen research
// instrumentation. Absence is supported only by legacy v2 manifests.
type ObserverPolicy struct {
	Schema            string `json:"schema"`
	SamplerAffinity   string `json:"samplerAffinity"`
	HeartbeatAffinity string `json:"heartbeatAffinity"`
	SampleNS          int64  `json:"sampleNS"`
	MaxGapNS          int64  `json:"maxGapNS"`
	PendingRecords    int    `json:"pendingRecords"`
	PendingBytes      int64  `json:"pendingBytes"`
	FrameBytes        int    `json:"frameBytes"`
	TotalBytes        int64  `json:"totalBytes"`
	DrainNS           int64  `json:"drainNS"`
	MaxRows           int    `json:"maxRows"`
}

func readObserverPolicy(m Manifest) (ObserverPolicy, error) {
	var p ObserverPolicy
	if m.Schema == manifestSchema {
		if !absentBinding(m.Observer) {
			return p, errors.New("v2 cannot enable observation implicitly")
		}
		return p, nil
	}
	if err := validateBinding(m.Observer); err != nil {
		return p, err
	}
	if err := readJSON(m.Observer.Path, &p); err != nil {
		return p, err
	}
	if p.Schema != "buildopt.history-replay/observer-policy/v1" || p.SampleNS != int64(100*time.Millisecond) || p.MaxGapNS != int64(500*time.Millisecond) || p.MaxRows < 1 || p.MaxRows > 10000 {
		return p, errors.New("unsupported observation policy or weakened cadence/gap criterion")
	}
	if err := p.writer().validate(); err != nil {
		return p, err
	}
	native, err := parseCPUSet(m.Affinity)
	if err != nil {
		return p, err
	}
	sampler, err := parseCPUSet(p.SamplerAffinity)
	if err != nil {
		return p, err
	}
	heartbeat, err := parseCPUSet(p.HeartbeatAffinity)
	if err != nil {
		return p, err
	}
	if sampler.Count() != 1 || heartbeat.Count() != 1 {
		return p, errors.New("observer and heartbeat require one CPU each")
	}
	for cpu := 0; cpu < 1024; cpu++ {
		if sampler.IsSet(cpu) && (native.IsSet(cpu) || heartbeat.IsSet(cpu)) || heartbeat.IsSet(cpu) && native.IsSet(cpu) {
			return p, errors.New("observation and native CPU masks overlap")
		}
	}
	return p, nil
}

func (p ObserverPolicy) writer() bufferedConfig {
	return bufferedConfig{p.PendingRecords, p.PendingBytes, p.FrameBytes, p.TotalBytes, p.DrainNS, ""}
}
