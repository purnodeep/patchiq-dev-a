package agent

// Refresh is exported for testing. It triggers an immediate re-read of settings from the store.
func (w *SettingsWatcher) Refresh() { w.refresh() }
