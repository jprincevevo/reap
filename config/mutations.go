package config

// AddRepo appends a new repo with the given URL (Selected: true) and returns
// true. If the URL is already present it is a no-op and returns false.
func (cfg *Config) AddRepo(url string) bool {
	for _, r := range cfg.Repos {
		if r.URL == url {
			return false
		}
	}
	cfg.Repos = append(cfg.Repos, Repo{URL: url, Selected: true})
	return true
}

// RemoveRepo removes every repo whose URL equals url from cfg.Repos in place.
func (cfg *Config) RemoveRepo(url string) {
	var kept []Repo
	for _, r := range cfg.Repos {
		if r.URL != url {
			kept = append(kept, r)
		}
	}
	cfg.Repos = kept
}

// ApplyGroupToRepos adds groupName (Selected: true) to each repo whose URL
// appears in urls, skipping repos that already have the group. Returns the
// number of repos actually modified.
func (cfg *Config) ApplyGroupToRepos(groupName string, urls []string) int {
	selected := make(map[string]bool, len(urls))
	for _, u := range urls {
		selected[u] = true
	}

	modified := 0
	for i, repo := range cfg.Repos {
		if !selected[repo.URL] {
			continue
		}
		alreadyHas := false
		for _, g := range repo.Groups {
			if g.Name == groupName {
				alreadyHas = true
				break
			}
		}
		if !alreadyHas {
			cfg.Repos[i].Groups = append(cfg.Repos[i].Groups, Group{
				Name:     groupName,
				Selected: true,
			})
			modified++
		}
	}
	return modified
}

// RemoveGroupFromAllRepos removes groupName from every repo that has it.
// Returns the total number of group entries removed (0 means the group was
// not found anywhere).
func (cfg *Config) RemoveGroupFromAllRepos(groupName string) int {
	removed := 0
	for i, repo := range cfg.Repos {
		var kept []Group
		for _, g := range repo.Groups {
			if g.Name != groupName {
				kept = append(kept, g)
			} else {
				removed++
			}
		}
		cfg.Repos[i].Groups = kept
	}
	return removed
}

// UniqueGroupNames returns the deduplicated list of group names seen across
// all repos, in the order they are first encountered. Always returns a
// non-nil slice (empty when no groups are defined).
func (cfg *Config) UniqueGroupNames() []string {
	seen := make(map[string]bool)
	names := []string{}
	for _, repo := range cfg.Repos {
		for _, g := range repo.Groups {
			if !seen[g.Name] {
				seen[g.Name] = true
				names = append(names, g.Name)
			}
		}
	}
	return names
}
