package cmd

import (
	"reflect"
	"testing"

	"github.com/jprincevevo/reap/config"
)

// helpers ----------------------------------------------------------------

func repos(urls ...string) []config.Repo {
	r := make([]config.Repo, len(urls))
	for i, u := range urls {
		r[i] = config.Repo{URL: u, Selected: true}
	}
	return r
}

func cfg(urls ...string) *config.Config {
	return &config.Config{Repos: repos(urls...)}
}

// addRepo ----------------------------------------------------------------

func TestAddRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		existing    []string
		add         string
		wantAdded   bool
		wantLen     int
		wantLastURL string
	}{
		{
			name:        "new URL is appended",
			existing:    []string{"https://a.git"},
			add:         "https://b.git",
			wantAdded:   true,
			wantLen:     2,
			wantLastURL: "https://b.git",
		},
		{
			name:      "duplicate URL is skipped",
			existing:  []string{"https://a.git"},
			add:       "https://a.git",
			wantAdded: false,
			wantLen:   1,
		},
		{
			name:        "add to empty config",
			existing:    nil,
			add:         "https://a.git",
			wantAdded:   true,
			wantLen:     1,
			wantLastURL: "https://a.git",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := cfg(tc.existing...)

			got := addRepo(c, tc.add)

			if got != tc.wantAdded {
				t.Errorf("addRepo() returned %v, want %v", got, tc.wantAdded)
			}
			if len(c.Repos) != tc.wantLen {
				t.Errorf("len(cfg.Repos) = %d, want %d", len(c.Repos), tc.wantLen)
			}
			if tc.wantAdded {
				last := c.Repos[len(c.Repos)-1]
				if last.URL != tc.wantLastURL {
					t.Errorf("last URL = %q, want %q", last.URL, tc.wantLastURL)
				}
				if !last.Selected {
					t.Error("newly added repo should have Selected: true")
				}
			}
		})
	}
}

// removeRepo -------------------------------------------------------------

func TestRemoveRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []string
		remove   string
		wantURLs []string
	}{
		{
			name:     "target URL is removed",
			existing: []string{"https://a.git", "https://b.git", "https://c.git"},
			remove:   "https://b.git",
			wantURLs: []string{"https://a.git", "https://c.git"},
		},
		{
			name:     "unrelated repos are untouched",
			existing: []string{"https://a.git", "https://b.git"},
			remove:   "https://b.git",
			wantURLs: []string{"https://a.git"},
		},
		{
			name:     "removing a URL not in the list is a no-op",
			existing: []string{"https://a.git"},
			remove:   "https://z.git",
			wantURLs: []string{"https://a.git"},
		},
		{
			name:     "removing from empty config returns nil",
			existing: nil,
			remove:   "https://a.git",
			wantURLs: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := cfg(tc.existing...)

			got := removeRepo(c, tc.remove)

			var gotURLs []string
			for _, r := range got {
				gotURLs = append(gotURLs, r.URL)
			}
			if !reflect.DeepEqual(gotURLs, tc.wantURLs) {
				t.Errorf("removeRepo() URLs = %v, want %v", gotURLs, tc.wantURLs)
			}
		})
	}
}

// listRepos --------------------------------------------------------------

func TestListRepos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []string
		want     []string
	}{
		{
			name:     "returns URLs in order",
			existing: []string{"https://a.git", "https://b.git", "https://c.git"},
			want:     []string{"https://a.git", "https://b.git", "https://c.git"},
		},
		{
			name:     "empty config returns empty slice",
			existing: nil,
			want:     []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := cfg(tc.existing...)

			got := listRepos(c)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("listRepos() = %v, want %v", got, tc.want)
			}
		})
	}
}

// applyGroupToRepos ------------------------------------------------------

func TestApplyGroupToRepos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		repos        []config.Repo
		groupName    string
		selectedURLs []string
		wantModified int
		checkRepo    string
		wantGroups   []string
	}{
		{
			name: "group is added to selected repos",
			repos: []config.Repo{
				{URL: "https://a.git"},
				{URL: "https://b.git"},
			},
			groupName:    "team",
			selectedURLs: []string{"https://a.git"},
			wantModified: 1,
			checkRepo:    "https://a.git",
			wantGroups:   []string{"team"},
		},
		{
			name: "already-having the group is a no-op",
			repos: []config.Repo{
				{URL: "https://a.git", Groups: []config.Group{{Name: "team", Selected: true}}},
			},
			groupName:    "team",
			selectedURLs: []string{"https://a.git"},
			wantModified: 0,
			checkRepo:    "https://a.git",
			wantGroups:   []string{"team"},
		},
		{
			name: "unselected repos are untouched",
			repos: []config.Repo{
				{URL: "https://a.git"},
				{URL: "https://b.git"},
			},
			groupName:    "team",
			selectedURLs: []string{"https://a.git"},
			wantModified: 1,
			checkRepo:    "https://b.git",
			wantGroups:   nil,
		},
		{
			name: "returns correct count when multiple repos modified",
			repos: []config.Repo{
				{URL: "https://a.git"},
				{URL: "https://b.git"},
				{URL: "https://c.git"},
			},
			groupName:    "team",
			selectedURLs: []string{"https://a.git", "https://b.git"},
			wantModified: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &config.Config{Repos: tc.repos}

			got := applyGroupToRepos(c, tc.groupName, tc.selectedURLs)

			if got != tc.wantModified {
				t.Errorf("applyGroupToRepos() = %d, want %d", got, tc.wantModified)
			}
			if tc.checkRepo == "" {
				return
			}
			for _, r := range c.Repos {
				if r.URL != tc.checkRepo {
					continue
				}
				var names []string
				for _, g := range r.Groups {
					names = append(names, g.Name)
				}
				if !reflect.DeepEqual(names, tc.wantGroups) {
					t.Errorf("repo %q groups = %v, want %v", tc.checkRepo, names, tc.wantGroups)
				}
				return
			}
			t.Errorf("checkRepo %q not found in config", tc.checkRepo)
		})
	}
}

// removeGroupFromAllRepos ------------------------------------------------

func TestRemoveGroupFromAllRepos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		repos       []config.Repo
		groupName   string
		wantRemoved int
		checkRepo   string
		wantGroups  []string
	}{
		{
			name: "group is removed from all repos that have it",
			repos: []config.Repo{
				{URL: "https://a.git", Groups: []config.Group{{Name: "team"}, {Name: "other"}}},
				{URL: "https://b.git", Groups: []config.Group{{Name: "team"}}},
			},
			groupName:   "team",
			wantRemoved: 2,
			checkRepo:   "https://a.git",
			wantGroups:  []string{"other"},
		},
		{
			name: "returns 0 when group not found",
			repos: []config.Repo{
				{URL: "https://a.git", Groups: []config.Group{{Name: "other"}}},
			},
			groupName:   "missing",
			wantRemoved: 0,
			checkRepo:   "https://a.git",
			wantGroups:  []string{"other"},
		},
		{
			name: "unrelated groups on the same repo survive",
			repos: []config.Repo{
				{URL: "https://a.git", Groups: []config.Group{
					{Name: "keep-me"},
					{Name: "remove-me"},
					{Name: "keep-me-too"},
				}},
			},
			groupName:   "remove-me",
			wantRemoved: 1,
			checkRepo:   "https://a.git",
			wantGroups:  []string{"keep-me", "keep-me-too"},
		},
		{
			name:        "no-op on empty config",
			repos:       nil,
			groupName:   "team",
			wantRemoved: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &config.Config{Repos: tc.repos}

			got := removeGroupFromAllRepos(c, tc.groupName)

			if got != tc.wantRemoved {
				t.Errorf("removeGroupFromAllRepos() = %d, want %d", got, tc.wantRemoved)
			}
			if tc.checkRepo == "" {
				return
			}
			for _, r := range c.Repos {
				if r.URL != tc.checkRepo {
					continue
				}
				var names []string
				for _, g := range r.Groups {
					names = append(names, g.Name)
				}
				if !reflect.DeepEqual(names, tc.wantGroups) {
					t.Errorf("repo %q groups = %v, want %v", tc.checkRepo, names, tc.wantGroups)
				}
				return
			}
			t.Errorf("checkRepo %q not found in config", tc.checkRepo)
		})
	}
}

// uniqueGroupNames -------------------------------------------------------

func TestUniqueGroupNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		repos []config.Repo
		want  []string
	}{
		{
			name: "deduplicates and preserves first-seen order",
			repos: []config.Repo{
				{URL: "https://a.git", Groups: []config.Group{{Name: "beta"}, {Name: "alpha"}}},
				{URL: "https://b.git", Groups: []config.Group{{Name: "alpha"}, {Name: "gamma"}}},
			},
			want: []string{"beta", "alpha", "gamma"},
		},
		{
			name: "returns empty (non-nil) slice for config with no groups",
			repos: []config.Repo{
				{URL: "https://a.git"},
			},
			want: []string{},
		},
		{
			name:  "returns empty (non-nil) slice for empty config",
			repos: nil,
			want:  []string{},
		},
		{
			name: "single group across multiple repos deduplicated to one entry",
			repos: []config.Repo{
				{URL: "https://a.git", Groups: []config.Group{{Name: "team"}}},
				{URL: "https://b.git", Groups: []config.Group{{Name: "team"}}},
				{URL: "https://c.git", Groups: []config.Group{{Name: "team"}}},
			},
			want: []string{"team"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &config.Config{Repos: tc.repos}

			got := uniqueGroupNames(c)

			if got == nil {
				t.Fatal("uniqueGroupNames() returned nil, want non-nil slice")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("uniqueGroupNames() = %v, want %v", got, tc.want)
			}
		})
	}
}
