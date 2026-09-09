package meta

import "testing"

func TestNormalizeThreadIncludesCustomizationAndMetadata(t *testing.T) {
	thread := normalizeThread(map[string]any{
		"thread_key": map[string]any{"thread_fbid": "10"}, "name": "Group", "thread_type": "GROUP", "messages_count": "42", "approval_mode": true,
		"customization_info": map[string]any{"emoji": "ok", "participant_customizations": []any{map[string]any{"participant_id": "1", "nickname": "Mew"}}},
		"thread_admins":      []any{map[string]any{"id": "1"}},
		"all_participants":   map[string]any{"edges": []any{map[string]any{"node": map[string]any{"messaging_actor": map[string]any{"id": "1", "name": "Mew"}}}}},
		"joinable_mode":      map[string]any{"mode": "1", "link": "https://example.invalid/join"},
	})
	if thread.ID != "10" || thread.Name != "Group" || thread.Emoji != "ok" || thread.MessageCount != 42 || !thread.ApprovalMode {
		t.Fatalf("unexpected thread: %#v", thread)
	}
	if thread.Nicknames["1"] != "Mew" || len(thread.Admins) != 1 || len(thread.Participants) != 1 || !thread.Joinable {
		t.Fatalf("missing metadata: %#v", thread)
	}
}
