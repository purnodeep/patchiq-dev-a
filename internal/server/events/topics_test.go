package events

import "testing"

func TestAllTopics_IncludesInventoryScanCompleted(t *testing.T) {
	topics := AllTopics()
	found := false
	for _, topic := range topics {
		if topic == InventoryScanCompleted {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllTopics() missing inventory.scan_completed")
	}
}

func TestAllTopics_NoDuplicates(t *testing.T) {
	topics := AllTopics()
	seen := make(map[string]bool, len(topics))
	for _, topic := range topics {
		if seen[topic] {
			t.Errorf("AllTopics() has duplicate topic: %s", topic)
		}
		seen[topic] = true
	}
}

func TestTopicSet_MatchesAllTopics(t *testing.T) {
	topics := AllTopics()
	set := TopicSet()

	if len(set) != len(topics) {
		t.Errorf("TopicSet() has %d entries, AllTopics() has %d (possible duplicates)", len(set), len(topics))
	}

	for _, topic := range topics {
		if _, ok := set[topic]; !ok {
			t.Errorf("TopicSet() missing topic: %s", topic)
		}
	}
}

func TestAllTopics_ContainsPolicyAutoDeployed(t *testing.T) {
	if _, ok := TopicSet()[PolicyAutoDeployed]; !ok {
		t.Error("AllTopics() missing PolicyAutoDeployed")
	}
}

func TestAllTopics_ContainsHubSyncTopics(t *testing.T) {
	for _, topic := range []string{HubSyncConfigUpdated, HubSyncTriggered} {
		if _, ok := TopicSet()[topic]; !ok {
			t.Errorf("AllTopics() missing %s", topic)
		}
	}
}
