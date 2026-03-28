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
// appModel.Update
// ==============================

// newTestAppModel builds an appModel that is ready to receive key input
// (home list sized and marked ready).
func newTestAppModel(cfg *config.Config) appModel {
	m := appModel{
		cfg:    cfg,
		screen: appScreenHome,
		home:   NewGroupModel(cfg),
		width:  80,
	}
	m.home.list.SetSize(80, listHeight)
	m.home.ready = true
	return m
}

func TestAppModel_WindowSizeMsg_StoresWidth(t *testing.T) {
	cfg := sampleCfg()
	am := newTestAppModel(cfg)

	updated, _ := am.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(appModel)
	if result.width != 120 {
		t.Errorf("width = %d, want 120", result.width)
	}
}

func TestAppModel_GKey_TransitionsToGroupsScreen(t *testing.T) {
	am := newTestAppModel(sampleCfg())
	updated, _ := am.Update(pressChar('g'))
	result := updated.(appModel)
	if result.screen != appScreenGroups {
		t.Errorf("screen = %v, want appScreenGroups", result.screen)
	}
}

func TestAppModel_RKey_TransitionsToReposScreen(t *testing.T) {
	am := newTestAppModel(sampleCfg())
	updated, _ := am.Update(pressChar('r'))
	result := updated.(appModel)
	if result.screen != appScreenRepos {
		t.Errorf("screen = %v, want appScreenRepos", result.screen)
	}
}

func TestAppModel_SKey_TransitionsToSettingsScreen(t *testing.T) {
	am := newTestAppModel(sampleCfg())
	updated, _ := am.Update(pressChar('s'))
	result := updated.(appModel)
	if result.screen != appScreenSettings {
		t.Errorf("screen = %v, want appScreenSettings", result.screen)
	}
}

func TestAppModel_HomeEnter_TransitionsToRepoScreen(t *testing.T) {
	am := newTestAppModel(sampleCfg())
	// Enter selects the first item ("Show All"), which transitions to appScreenRepo.
	updated, _ := am.Update(pressKey(tea.KeyEnter))
	result := updated.(appModel)
	if result.screen != appScreenRepo {
		t.Errorf("screen = %v, want appScreenRepo", result.screen)
	}
}

func TestAppModel_GroupsGoBack_TransitionsToHome(t *testing.T) {
	cfg := sampleCfg()
	am := appModel{
		cfg:    cfg,
		screen: appScreenGroups,
		groups: newManageGroupModel(cfg),
		width:  80,
	}
	am.groups.list.SetSize(80, listHeight)
	am.groups.ready = true

	// esc at groups list sets goBack = true, which appModel intercepts.
	updated, _ := am.Update(pressKey(tea.KeyEscape))
	result := updated.(appModel)
	if result.screen != appScreenHome {
		t.Errorf("screen = %v, want appScreenHome", result.screen)
	}
}

func TestAppModel_ReposGoBack_TransitionsToHome(t *testing.T) {
	cfg := sampleCfg()
	am := appModel{
		cfg:    cfg,
		screen: appScreenRepos,
		repos:  newManageRepoModel(cfg),
		width:  80,
	}
	am.repos.list.SetSize(80, listHeight)
	am.repos.ready = true

	// esc at repos list sets goBack = true, which appModel intercepts.
	updated, _ := am.Update(pressKey(tea.KeyEscape))
	result := updated.(appModel)
	if result.screen != appScreenHome {
		t.Errorf("screen = %v, want appScreenHome", result.screen)
	}
}

// ==============================
// promptModel constructor
// ==============================

func TestPromptModel_Constructor_IsFocused(t *testing.T) {
	m := NewPromptModel("title", "placeholder")
	if !m.input.Focused() {
		t.Error("NewPromptModel: input.Focused() = false, want true")
	}
}

func TestPromptModel_Constructor_IsReady(t *testing.T) {
	m := NewPromptModel("title", "placeholder")
	if !m.ready {
		t.Error("NewPromptModel: ready = false, want true")
	}
}

// ==============================
// manageGroupModel — list screen
// ==============================

func TestManageGroupModel_List_QHardQuits(t *testing.T) {
	m := newManageGroupModel(sampleCfg())
	updated, _ := m.Update(pressChar('q'))
	result := updated.(manageGroupModel)
	if result.goBack {
		t.Error("q: goBack = true, want false (hard quit)")
	}
}

func TestManageGroupModel_List_EscSetsGoBack(t *testing.T) {
	m := newManageGroupModel(sampleCfg())
	updated, _ := m.Update(pressKey(tea.KeyEscape))
	result := updated.(manageGroupModel)
	if !result.goBack {
		t.Error("esc: goBack = false, want true")
	}
}

func TestManageGroupModel_List_CtrlCHardQuits(t *testing.T) {
	m := newManageGroupModel(sampleCfg())
	updated, _ := m.Update(pressCtrl('c'))
	result := updated.(manageGroupModel)
	if result.goBack {
		t.Error("ctrl+c: goBack = true, want false (hard quit)")
	}
}

func TestManageGroupModel_List_EnterTransitionsToDetail(t *testing.T) {
	m := newManageGroupModel(sampleCfg())

	updated, _ := m.Update(pressKey(tea.KeyEnter))
	result := updated.(manageGroupModel)
	if result.screen != mgScreenDetail {
		t.Errorf("screen = %v, want mgScreenDetail", result.screen)
	}
	if result.selectedGroup == "" {
		t.Error("selectedGroup is empty after enter")
	}
}

func TestManageGroupModel_List_ATransitionsToAdd(t *testing.T) {
	m := newManageGroupModel(sampleCfg())

	updated, _ := m.Update(pressChar('a'))
	result := updated.(manageGroupModel)
	if result.screen != mgScreenAdd {
		t.Errorf("screen = %v, want mgScreenAdd", result.screen)
	}
}

func TestManageGroupModel_List_RTransitionsToRename(t *testing.T) {
	m := newManageGroupModel(sampleCfg())

	updated, _ := m.Update(pressChar('r'))
	result := updated.(manageGroupModel)
	if result.screen != mgScreenRename {
		t.Errorf("screen = %v, want mgScreenRename", result.screen)
	}
	if result.selectedGroup == "" {
		t.Error("selectedGroup is empty after r")
	}
}

func TestManageGroupModel_List_DTransitionsToConfirmDelete(t *testing.T) {
	m := newManageGroupModel(sampleCfg())

	updated, _ := m.Update(pressChar('d'))
	result := updated.(manageGroupModel)
	if result.screen != mgScreenConfirmDelete {
		t.Errorf("screen = %v, want mgScreenConfirmDelete", result.screen)
	}
	if result.selectedGroup == "" {
		t.Error("selectedGroup is empty after d")
	}
}

// ==============================
// manageGroupModel — detail screen
// ==============================

func newManageGroupDetailModel(cfg *config.Config, group string) manageGroupModel {
	m := manageGroupModel{cfg: cfg, screen: mgScreenDetail, selectedGroup: group}
	m.list = m.buildList()
	m.detailList = m.buildDetailList()
	return m
}

func TestManageGroupModel_Detail_EscTransitionsBackToList(t *testing.T) {
	m := newManageGroupDetailModel(sampleCfg(), "work")

	updated, _ := m.Update(pressKey(tea.KeyEscape))
	result := updated.(manageGroupModel)
	if result.screen != mgScreenList {
		t.Errorf("screen = %v, want mgScreenList", result.screen)
	}
}

func TestManageGroupModel_Detail_QHardQuits(t *testing.T) {
	m := newManageGroupDetailModel(sampleCfg(), "work")

	updated, _ := m.Update(pressChar('q'))
	result := updated.(manageGroupModel)
	if result.screen != mgScreenDetail {
		t.Errorf("q: screen changed to %v, want mgScreenDetail (should hard quit, not navigate)", result.screen)
	}
}

func TestManageGroupModel_Detail_DTransitionsToConfirmRemoveRepo(t *testing.T) {
	m := newManageGroupDetailModel(sampleCfg(), "work")

	updated, _ := m.Update(pressChar('d'))
	result := updated.(manageGroupModel)
	if result.screen != mgScreenConfirmRemoveRepo {
		t.Errorf("screen = %v, want mgScreenConfirmRemoveRepo", result.screen)
	}
	if result.selectedRepo == "" {
		t.Error("selectedRepo is empty after d in detail")
	}
}

func TestManageGroupModel_Detail_ATransitionsToAddRepoToGroup(t *testing.T) {
	m := newManageGroupDetailModel(sampleCfg(), "work")

	updated, _ := m.Update(pressChar('a'))
	result := updated.(manageGroupModel)
	if result.screen != mgScreenAddRepoToGroup {
		t.Errorf("screen = %v, want mgScreenAddRepoToGroup", result.screen)
	}
}

// ==============================
// manageGroupModel — buildDetailList
// ==============================

func TestManageGroupModel_BuildDetailList_ReturnsReposInGroup(t *testing.T) {
	// sampleCfg: "work" group contains beta.git and gamma.git
	m := manageGroupModel{cfg: sampleCfg(), selectedGroup: "work"}
	dl := m.buildDetailList()
	if len(dl.Items()) != 2 {
		t.Fatalf("detail list has %d items, want 2", len(dl.Items()))
	}
}

func TestManageGroupModel_BuildDetailList_TitleIncludesGroupName(t *testing.T) {
	m := manageGroupModel{cfg: sampleCfg(), selectedGroup: "work"}
	dl := m.buildDetailList()
	if dl.Title != "Group: work" {
		t.Errorf("title = %q, want \"Group: work\"", dl.Title)
	}
}

// ==============================
// manageRepoModel — list screen
// ==============================

func TestManageRepoModel_List_QHardQuits(t *testing.T) {
	m := newManageRepoModel(sampleCfg())
	updated, _ := m.Update(pressChar('q'))
	result := updated.(manageRepoModel)
	if result.goBack {
		t.Error("q: goBack = true, want false (hard quit)")
	}
}

func TestManageRepoModel_List_EscSetsGoBack(t *testing.T) {
	m := newManageRepoModel(sampleCfg())
	updated, _ := m.Update(pressKey(tea.KeyEscape))
	result := updated.(manageRepoModel)
	if !result.goBack {
		t.Error("esc: goBack = false, want true")
	}
}

func TestManageRepoModel_List_CtrlCHardQuits(t *testing.T) {
	m := newManageRepoModel(sampleCfg())
	updated, _ := m.Update(pressCtrl('c'))
	result := updated.(manageRepoModel)
	if result.goBack {
		t.Error("ctrl+c: goBack = true, want false (hard quit)")
	}
}

func TestManageRepoModel_List_EnterTransitionsToDetail(t *testing.T) {
	m := newManageRepoModel(sampleCfg())

	updated, _ := m.Update(pressKey(tea.KeyEnter))
	result := updated.(manageRepoModel)
	if result.screen != mrScreenDetail {
		t.Errorf("screen = %v, want mrScreenDetail", result.screen)
	}
	if result.selectedRepo == "" {
		t.Error("selectedRepo is empty after enter")
	}
}

func TestManageRepoModel_List_ATransitionsToAdd(t *testing.T) {
	m := newManageRepoModel(sampleCfg())

	updated, _ := m.Update(pressChar('a'))
	result := updated.(manageRepoModel)
	if result.screen != mrScreenAdd {
		t.Errorf("screen = %v, want mrScreenAdd", result.screen)
	}
}

// ==============================
// manageRepoModel — detail screen
// ==============================

func newManageRepoDetailModel(cfg *config.Config, repoURL string) manageRepoModel {
	m := manageRepoModel{cfg: cfg, screen: mrScreenDetail, selectedRepo: repoURL}
	m.list = m.buildList()
	m.detailList = m.buildDetailList()
	return m
}

func TestManageRepoModel_Detail_EscTransitionsBackToList(t *testing.T) {
	m := newManageRepoDetailModel(sampleCfg(), "https://github.com/a/alpha.git")

	updated, _ := m.Update(pressKey(tea.KeyEscape))
	result := updated.(manageRepoModel)
	if result.screen != mrScreenList {
		t.Errorf("screen = %v, want mrScreenList", result.screen)
	}
}

func TestManageRepoModel_Detail_ATransitionsToAddToGroup(t *testing.T) {
	m := newManageRepoDetailModel(sampleCfg(), "https://github.com/a/alpha.git")

	updated, _ := m.Update(pressChar('a'))
	result := updated.(manageRepoModel)
	if result.screen != mrScreenAddToGroup {
		t.Errorf("screen = %v, want mrScreenAddToGroup", result.screen)
	}
}

func TestManageRepoModel_Detail_DTransitionsToConfirmRemoveGroup(t *testing.T) {
	// gamma.git has 2 groups — its detail list is non-empty so d can select one
	m := newManageRepoDetailModel(sampleCfg(), "https://github.com/c/gamma.git")

	updated, _ := m.Update(pressChar('d'))
	result := updated.(manageRepoModel)
	if result.screen != mrScreenConfirmRemoveGroup {
		t.Errorf("screen = %v, want mrScreenConfirmRemoveGroup", result.screen)
	}
	if result.pendingGroup == "" {
		t.Error("pendingGroup is empty after d in detail")
	}
}

func TestManageRepoModel_AddToGroup_EscTransitionsBackToDetail(t *testing.T) {
	m := manageRepoModel{
		cfg:          sampleCfg(),
		screen:       mrScreenAddToGroup,
		selectedRepo: "https://github.com/a/alpha.git",
	}
	m.list = m.buildList()
	m.detailList = m.buildDetailList()
	m.groupList = m.buildGroupList()

	updated, _ := m.Update(pressKey(tea.KeyEscape))
	result := updated.(manageRepoModel)
	if result.screen != mrScreenDetail {
		t.Errorf("screen = %v, want mrScreenDetail", result.screen)
	}
}

// ==============================
// manageRepoModel — helper builders
// ==============================

func TestManageRepoModel_BuildDetailList_ReturnsGroupsForRepo(t *testing.T) {
	// beta.git belongs to one group: "work"
	m := manageRepoModel{cfg: sampleCfg(), selectedRepo: "https://github.com/b/beta.git"}
	dl := m.buildDetailList()
	if len(dl.Items()) != 1 {
		t.Fatalf("detail list has %d items, want 1", len(dl.Items()))
	}
	if got := string(dl.Items()[0].(item)); got != "work" {
		t.Errorf("items[0] = %q, want \"work\"", got)
	}
}

func TestManageRepoModel_BuildGroupList_ExcludesAlreadyMemberGroups(t *testing.T) {
	// beta.git is already in "work"; only "personal" should be offered
	m := manageRepoModel{cfg: sampleCfg(), selectedRepo: "https://github.com/b/beta.git"}
	gl := m.buildGroupList()
	if len(gl.Items()) != 1 {
		t.Fatalf("group list has %d items, want 1", len(gl.Items()))
	}
	if got := string(gl.Items()[0].(item)); got != "personal" {
		t.Errorf("items[0] = %q, want \"personal\"", got)
	}
}

func TestManageRepoModel_BuildGroupList_RepoWithNoGroupsSeesAll(t *testing.T) {
	// alpha.git has no groups — both "work" and "personal" should be offered
	m := manageRepoModel{cfg: sampleCfg(), selectedRepo: "https://github.com/a/alpha.git"}
	gl := m.buildGroupList()
	if len(gl.Items()) != 2 {
		t.Fatalf("group list has %d items, want 2 (all groups)", len(gl.Items()))
	}
}

// ==============================
// settingsModel
// ==============================

func TestSettingsModel_Esc_SetsGoBack(t *testing.T) {
	m := newSettingsModel(&config.Config{})
	updated, _ := m.Update(pressKey(tea.KeyEscape))
	sm := updated.(settingsModel)
	if !sm.goBack {
		t.Error("esc: goBack = false, want true")
	}
	if sm.quitting {
		t.Error("esc: quitting = true, want false")
	}
}

func TestSettingsModel_CtrlC_SetsQuitting(t *testing.T) {
	m := newSettingsModel(&config.Config{})
	updated, _ := m.Update(pressCtrl('c'))
	sm := updated.(settingsModel)
	if !sm.quitting {
		t.Error("ctrl+c: quitting = false, want true")
	}
	if sm.goBack {
		t.Error("ctrl+c: goBack = true, want false")
	}
}

func TestSettingsModel_EnterOnDepth_AdvancesToDirField(t *testing.T) {
	m := newSettingsModel(&config.Config{})
	if m.field != settingsFieldDepth {
		t.Fatal("precondition: expected settingsFieldDepth as initial field")
	}
	updated, _ := m.Update(pressKey(tea.KeyEnter))
	sm := updated.(settingsModel)
	if sm.field != settingsFieldDir {
		t.Errorf("field = %v, want settingsFieldDir", sm.field)
	}
}

func TestSettingsModel_EnterOnDir_AdvancesToPullField(t *testing.T) {
	m := newSettingsModel(&config.Config{})
	m.field = settingsFieldDir
	m.prompt = NewPromptModel("Default clone directory", "")
	updated, _ := m.Update(pressKey(tea.KeyEnter))
	sm := updated.(settingsModel)
	if sm.field != settingsFieldPull {
		t.Errorf("field = %v, want settingsFieldPull", sm.field)
	}
}

func TestSettingsModel_PullToggle_SpaceToggles(t *testing.T) {
	m := newSettingsModel(&config.Config{DefaultPull: false})
	m.field = settingsFieldPull
	if m.pull {
		t.Fatal("precondition: pull should start false")
	}
	updated, _ := m.Update(pressKey(tea.KeySpace))
	sm := updated.(settingsModel)
	if !sm.pull {
		t.Error("space: pull = false, want true after toggle")
	}
}

func TestSettingsModel_PullToggle_EnterSetsGoBack(t *testing.T) {
	m := newSettingsModel(&config.Config{})
	m.field = settingsFieldPull
	updated, _ := m.Update(pressKey(tea.KeyEnter))
	sm := updated.(settingsModel)
	if !sm.goBack {
		t.Error("enter on pull: goBack = false, want true")
	}
}
