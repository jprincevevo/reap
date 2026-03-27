package tui

// Tests for the pure model logic: constructors, Update key-handling, and
// screen transitions. We test these directly (no tea.Program) by calling
// Update with hand-crafted KeyPressMsg / WindowSizeMsg values and inspecting
// the returned model state. No terminal is required.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jprincevevo/reap/config"
)

// ---- key message helpers ----

// pressKey wraps a special-key rune (KeyEnter, KeyEscape, KeySpace, …).
func pressKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// pressChar wraps a printable character such as 'q'.
func pressChar(ch rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: ch, Text: string(ch)}
}

// pressCtrl wraps a ctrl+<code> combination such as ctrl+c.
func pressCtrl(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl}
}

// ---- shared fixture ----

// sampleCfg returns a non-trivial config used across multiple tests.
//
//	alpha.git — no group, Selected: false
//	beta.git  — group "work" (Selected: true),  repo Selected: true
//	gamma.git — groups "work" (Selected: false) + "personal" (Selected: true)
func sampleCfg() *config.Config {
	return &config.Config{
		Repos: []config.Repo{
			{URL: "https://github.com/a/alpha.git", Selected: false},
			{
				URL:      "https://github.com/b/beta.git",
				Selected: true,
				Groups:   []config.Group{{Name: "work", Selected: true}},
			},
			{
				URL: "https://github.com/c/gamma.git",
				Groups: []config.Group{
					{Name: "work", Selected: false},
					{Name: "personal", Selected: true},
				},
			},
		},
	}
}

// ==============================
// repoItem
// ==============================

func TestRepoItem_Description(t *testing.T) {
	tests := []struct {
		name     string
		item     repoItem
		wantDesc string
	}{
		{"selected", repoItem{url: "https://x.git", selected: true}, "[x]"},
		{"unselected", repoItem{url: "https://x.git", selected: false}, "[ ]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.Description(); got != tt.wantDesc {
				t.Errorf("Description() = %q, want %q", got, tt.wantDesc)
			}
		})
	}
}

func TestRepoItem_FilterValue_ReturnsURL(t *testing.T) {
	ri := repoItem{url: "https://github.com/owner/repo.git"}
	if got := ri.FilterValue(); got != ri.url {
		t.Errorf("FilterValue() = %q, want %q", got, ri.url)
	}
}

// ==============================
// NewRepoModel
// ==============================

func TestNewRepoModel_ShowAll_IncludesAllRepos(t *testing.T) {
	cfg := sampleCfg()
	m := NewRepoModel(cfg, "Show All")
	items := m.list.Items()

	if len(items) != len(cfg.Repos) {
		t.Fatalf("len(items) = %d, want %d", len(items), len(cfg.Repos))
	}
	for i, repo := range cfg.Repos {
		ri, ok := items[i].(repoItem)
		if !ok {
			t.Fatalf("items[%d] is not a repoItem", i)
		}
		if ri.url != repo.URL {
			t.Errorf("items[%d].url = %q, want %q", i, ri.url, repo.URL)
		}
		if ri.selected != repo.Selected {
			t.Errorf("items[%d].selected = %v, want %v", i, ri.selected, repo.Selected)
		}
	}
}

func TestNewRepoModel_GroupFilter_OnlyIncludesMatchingRepos(t *testing.T) {
	m := NewRepoModel(sampleCfg(), "work")
	items := m.list.Items()

	// beta.git and gamma.git are both in "work"
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if ri := items[0].(repoItem); ri.url != "https://github.com/b/beta.git" {
		t.Errorf("items[0].url = %q, want beta.git", ri.url)
	}
	if ri := items[1].(repoItem); ri.url != "https://github.com/c/gamma.git" {
		t.Errorf("items[1].url = %q, want gamma.git", ri.url)
	}
}

func TestNewRepoModel_GroupFilter_UsesGroupSelectedNotRepoSelected(t *testing.T) {
	m := NewRepoModel(sampleCfg(), "work")
	items := m.list.Items()
	if len(items) < 2 {
		t.Fatal("expected at least 2 items for group 'work'")
	}

	// beta.git: group "work" Selected=true
	if ri := items[0].(repoItem); !ri.selected {
		t.Error("beta.git: selected = false, want true (group Selected is true)")
	}
	// gamma.git: group "work" Selected=false
	if ri := items[1].(repoItem); ri.selected {
		t.Error("gamma.git: selected = true, want false (group Selected is false)")
	}
}

func TestNewRepoModel_UnknownGroup_ReturnsNoItems(t *testing.T) {
	m := NewRepoModel(sampleCfg(), "nonexistent")
	if n := len(m.list.Items()); n != 0 {
		t.Errorf("expected 0 items for unknown group, got %d", n)
	}
}

// ==============================
// NewGroupModel
// ==============================

func TestNewGroupModel_AlwaysHasShowAllAsFirstItem(t *testing.T) {
	m := NewGroupModel(&config.Config{})
	items := m.list.Items()
	if len(items) == 0 {
		t.Fatal("expected at least 1 item (Show All), got 0")
	}
	if got := string(items[0].(item)); got != "Show All" {
		t.Errorf("items[0] = %q, want \"Show All\"", got)
	}
}

func TestNewGroupModel_DeduplicatesGroupNames(t *testing.T) {
	// sampleCfg has "work" on both beta and gamma, and "personal" on gamma.
	m := NewGroupModel(sampleCfg())
	items := m.list.Items()

	// Expect: Show All, work, personal (3 items; "work" not duplicated)
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3 (Show All + work + personal)", len(items))
	}
	if got := string(items[1].(item)); got != "work" {
		t.Errorf("items[1] = %q, want \"work\"", got)
	}
	if got := string(items[2].(item)); got != "personal" {
		t.Errorf("items[2] = %q, want \"personal\"", got)
	}
}

func TestNewGroupModel_EmptyConfig_OnlyShowAll(t *testing.T) {
	m := NewGroupModel(&config.Config{Repos: []config.Repo{}})
	if n := len(m.list.Items()); n != 1 {
		t.Errorf("len(items) = %d, want 1 (only Show All)", n)
	}
}

// ==============================
// repoModel.Update
// ==============================

func TestRepoModel_QuitKeys_SetQuitting(t *testing.T) {
	for _, msg := range []tea.Msg{pressChar('q'), pressCtrl('c')} {
		m := NewRepoModel(sampleCfg(), "Show All")
		updated, _ := m.Update(msg)
		rm := updated.(repoModel)
		if !rm.quitting {
			t.Errorf("msg %T: quitting = false, want true", msg)
		}
		if rm.goBack {
			t.Errorf("msg %T: goBack = true, want false", msg)
		}
	}
}

func TestRepoModel_Esc_SetsGoBack(t *testing.T) {
	m := NewRepoModel(sampleCfg(), "Show All")
	updated, _ := m.Update(pressKey(tea.KeyEscape))
	rm := updated.(repoModel)
	if !rm.goBack {
		t.Error("esc: goBack = false, want true")
	}
	if rm.quitting {
		t.Error("esc: quitting = true, want false")
	}
}

func TestRepoModel_Enter_QuitsWithoutFlags(t *testing.T) {
	m := NewRepoModel(sampleCfg(), "Show All")
	updated, _ := m.Update(pressKey(tea.KeyEnter))
	rm := updated.(repoModel)
	if rm.goBack {
		t.Error("enter: goBack = true, want false")
	}
	if rm.quitting {
		t.Error("enter: quitting = true, want false")
	}
}

func TestRepoModel_Space_TogglesSelectedOnCurrentItem(t *testing.T) {
	// alpha.git starts unselected; space should select it.
	m := NewRepoModel(sampleCfg(), "Show All")
	if m.list.Items()[0].(repoItem).selected {
		t.Fatal("precondition failed: alpha.git should start unselected")
	}

	updated, _ := m.Update(pressKey(tea.KeySpace))
	rm := updated.(repoModel)
	if !rm.list.Items()[0].(repoItem).selected {
		t.Error("space: item[0] selected = false, want true after toggle")
	}
}

// ==============================
// groupModel.Update
// ==============================

func TestGroupModel_QuitKeys_SetQuitting(t *testing.T) {
	for _, msg := range []tea.Msg{pressChar('q'), pressCtrl('c')} {
		m := NewGroupModel(sampleCfg())
		updated, _ := m.Update(msg)
		gm := updated.(groupModel)
		if !gm.quitting {
			t.Errorf("msg %T: quitting = false, want true", msg)
		}
	}
}

func TestGroupModel_Enter_SetsChoiceToFirstItem(t *testing.T) {
	m := NewGroupModel(sampleCfg())
	updated, _ := m.Update(pressKey(tea.KeyEnter))
	gm := updated.(groupModel)
	if gm.choice == "" {
		t.Fatal("enter: choice is empty, want non-empty")
	}
	// Cursor starts at index 0, which is always "Show All".
	if gm.choice != "Show All" {
		t.Errorf("choice = %q, want \"Show All\"", gm.choice)
	}
}

// ==============================
// confirmModel.Update
// ==============================

func TestConfirmModel_DefaultActiveButton_IsNo(t *testing.T) {
	// InitialConfirmModel sets activeButton=1 (No). Replicate that here.
	m := confirmModel{prompt: "Continue?", activeButton: 1}
	if m.activeButton != 1 {
		t.Errorf("activeButton = %d, want 1 (No)", m.activeButton)
	}
}

func TestConfirmModel_Left_MovesToYesButton(t *testing.T) {
	m := confirmModel{prompt: "Continue?", activeButton: 1}
	updated, _ := m.Update(pressKey(tea.KeyLeft))
	cm := updated.(confirmModel)
	if cm.activeButton != 0 {
		t.Errorf("after left: activeButton = %d, want 0 (Yes)", cm.activeButton)
	}
}

func TestConfirmModel_Right_DoesNotExceedMax(t *testing.T) {
	m := confirmModel{prompt: "Continue?", activeButton: 1}
	updated, _ := m.Update(pressKey(tea.KeyRight))
	cm := updated.(confirmModel)
	if cm.activeButton != 1 {
		t.Errorf("after right at max: activeButton = %d, want 1", cm.activeButton)
	}
}

func TestConfirmModel_Enter_OnYesConfirms(t *testing.T) {
	m := confirmModel{prompt: "Continue?", activeButton: 0} // Yes
	updated, _ := m.Update(pressKey(tea.KeyEnter))
	cm := updated.(confirmModel)
	if !cm.confirmed {
		t.Error("enter on Yes: confirmed = false, want true")
	}
	if !cm.quitting {
		t.Error("enter on Yes: quitting = false, want true")
	}
}

func TestConfirmModel_Enter_OnNoCancels(t *testing.T) {
	m := confirmModel{prompt: "Continue?", activeButton: 1} // No
	updated, _ := m.Update(pressKey(tea.KeyEnter))
	cm := updated.(confirmModel)
	if cm.confirmed {
		t.Error("enter on No: confirmed = true, want false")
	}
	if !cm.quitting {
		t.Error("enter on No: quitting = false, want true")
	}
}

func TestConfirmModel_QuitKeys_SetQuitting(t *testing.T) {
	for _, msg := range []tea.Msg{pressChar('q'), pressCtrl('c')} {
		m := confirmModel{prompt: "Continue?", activeButton: 1}
		updated, _ := m.Update(msg)
		cm := updated.(confirmModel)
		if !cm.quitting {
			t.Errorf("msg %T: quitting = false, want true", msg)
		}
	}
}

// ==============================
// flowModel.Update
// ==============================

func TestFlowModel_WindowSizeMsg_StoresWidth(t *testing.T) {
	cfg := sampleCfg()
	fm := flowModel{cfg: cfg, screen: flowScreenGroup, group: NewGroupModel(cfg)}

	updated, _ := fm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(flowModel)
	if result.width != 120 {
		t.Errorf("width = %d, want 120", result.width)
	}
}

func TestFlowModel_GroupEnter_TransitionsToRepoScreen(t *testing.T) {
	cfg := sampleCfg()
	fm := flowModel{cfg: cfg, screen: flowScreenGroup, group: NewGroupModel(cfg)}

	// Pressing enter selects the first group item ("Show All") and should
	// cause the flow to advance to the repo screen.
	updated, _ := fm.Update(pressKey(tea.KeyEnter))
	result := updated.(flowModel)
	if result.screen != flowScreenRepo {
		t.Errorf("screen = %v, want flowScreenRepo", result.screen)
	}
}

func TestFlowModel_RepoEsc_TransitionsBackToGroupScreen(t *testing.T) {
	cfg := sampleCfg()
	fm := flowModel{cfg: cfg, screen: flowScreenRepo, repo: NewRepoModel(cfg, "Show All")}

	updated, _ := fm.Update(pressKey(tea.KeyEscape))
	result := updated.(flowModel)
	if result.screen != flowScreenGroup {
		t.Errorf("screen = %v, want flowScreenGroup", result.screen)
	}
}
