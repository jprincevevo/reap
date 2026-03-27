package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jprincevevo/reap/config"
)

func TestHasGroups(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want bool
	}{
		{
			name: "empty config",
			cfg:  config.Config{},
			want: false,
		},
		{
			name: "repos without any groups",
			cfg: config.Config{
				Repos: []config.Repo{
					{URL: "https://github.com/a/b.git"},
					{URL: "https://github.com/c/d.git"},
				},
			},
			want: false,
		},
		{
			name: "single repo with a group",
			cfg: config.Config{
				Repos: []config.Repo{
					{URL: "https://github.com/a/b.git", Groups: []config.Group{{Name: "work"}}},
				},
			},
			want: true,
		},
		{
			name: "mixed: only second repo has groups",
			cfg: config.Config{
				Repos: []config.Repo{
					{URL: "https://github.com/a/b.git"},
					{URL: "https://github.com/c/d.git", Groups: []config.Group{{Name: "personal"}}},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.HasGroups(); got != tt.want {
				t.Errorf("HasGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetConfigPath verifies the path resolves under $HOME/.config/reap/ and
// that the directory is created as a side effect.
func TestGetConfigPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	got, err := config.GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() error = %v", err)
	}

	want := filepath.Join(tmp, ".config", "reap", "config.yaml")
	if got != want {
		t.Errorf("GetConfigPath() = %q, want %q", got, want)
	}

	if _, err := os.Stat(filepath.Dir(got)); os.IsNotExist(err) {
		t.Error("GetConfigPath() did not create the config directory")
	}
}

// TestLoad_CreatesDefaultConfig checks that the very first Load on a pristine
// home directory creates an empty config file and returns created=true.
func TestLoad_CreatesDefaultConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, created, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !created {
		t.Error("Load() created = false, want true on first run")
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if len(cfg.Repos) != 0 {
		t.Errorf("default config has %d repos, want 0", len(cfg.Repos))
	}
}

// TestLoad_FalseCreatedOnSubsequentCall ensures the created flag is false once
// the config file already exists.
func TestLoad_FalseCreatedOnSubsequentCall(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, _, err := config.Load(); err != nil {
		t.Fatalf("first Load() error = %v", err)
	}

	_, created, err := config.Load()
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if created {
		t.Error("second Load() created = true, want false")
	}
}

// TestSave_RoundTrip saves a richly populated Config and reloads it, asserting
// every field survives the YAML marshal/unmarshal cycle.
func TestSave_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	want := &config.Config{
		DefaultDepth: 3,
		Repos: []config.Repo{
			{
				URL:      "https://github.com/owner/repo.git",
				Selected: true,
				Groups: []config.Group{
					{Name: "work", Selected: true},
					{Name: "personal", Selected: false},
				},
			},
			{
				URL:      "https://github.com/other/project.git",
				Selected: false,
			},
		},
	}

	if err := config.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, created, err := config.Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}
	if created {
		t.Error("Load() after Save() reported created = true")
	}
	if got.DefaultDepth != want.DefaultDepth {
		t.Errorf("DefaultDepth = %d, want %d", got.DefaultDepth, want.DefaultDepth)
	}
	if len(got.Repos) != len(want.Repos) {
		t.Fatalf("len(Repos) = %d, want %d", len(got.Repos), len(want.Repos))
	}
	for i, w := range want.Repos {
		g := got.Repos[i]
		if g.URL != w.URL {
			t.Errorf("Repos[%d].URL = %q, want %q", i, g.URL, w.URL)
		}
		if g.Selected != w.Selected {
			t.Errorf("Repos[%d].Selected = %v, want %v", i, g.Selected, w.Selected)
		}
		if len(g.Groups) != len(w.Groups) {
			t.Fatalf("Repos[%d]: len(Groups) = %d, want %d", i, len(g.Groups), len(w.Groups))
		}
		for j, wg := range w.Groups {
			gg := g.Groups[j]
			if gg.Name != wg.Name {
				t.Errorf("Repos[%d].Groups[%d].Name = %q, want %q", i, j, gg.Name, wg.Name)
			}
			if gg.Selected != wg.Selected {
				t.Errorf("Repos[%d].Groups[%d].Selected = %v, want %v", i, j, gg.Selected, wg.Selected)
			}
		}
	}
}
