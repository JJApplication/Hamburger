package stat

// RecordObservation publishes one completed gateway request to historical
// storage. It is intentionally safe when history is disabled.
func (m *StatManager) RecordObservation(observation RequestSnapshot) {
	if m == nil || m.history == nil {
		return
	}
	m.history.RecordObservation(observation)
}

// HistoryCapabilities returns resource availability for the public API.
func (m *StatManager) HistoryCapabilities() map[string]bool {
	capabilities := map[string]bool{
		"system_cpu":      false,
		"system_memory":   false,
		"system_network":  false,
		"system_disk_io":  false,
		"process_cpu":     false,
		"process_memory":  false,
		"process_disk_io": false,
		"program_traffic": false,
	}
	if m == nil || m.history == nil {
		return capabilities
	}
	if m.sampler != nil {
		for key, value := range m.sampler.capabilities().mapValue() {
			capabilities[key] = value
		}
	}
	return capabilities
}
