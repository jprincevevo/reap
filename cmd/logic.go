package cmd

import "github.com/jprincevevo/reap/config"

// addRepo appends a new repo with the given URL to cfg (Selected: true) and
// returns true. If the URL is already present it is a no-op and returns false.
func addRepo(cfg *config.Config, url string) bool {
	for _, r := range cfg.Repos {
		if r.URL == url {
			return false
		}
	}
	cfg.Repos = append(cfg.Repos, config.Repo{URL: url, Selected: true})
	return true
}

// removeRepo returns a new repo slice with every entry whose URL matches url
// removed. Repos whose URL does not match are preserved in order.
func removeRepo(cfg *config.Config, url string) []config.Repo {
	var kept []config.Repo
	for _, r := range cfg.Repos {
		if r.URL != url {
			kept = append(kept, r)
		}
	}
	return kept
}

// listRepos returns the URL of every repo in cfg, in order.
func listRepos(cfg *config.Config) []string {
	urls := make([]string, 0, len(cfg.Repos))
	for _, r := range cfg.Repos {
		urls = append(urls, r.URL)
	}
	return urls
}

// applyGroupToRepos adds groupName (Selected: true) to each repo whose URL
// appears in selectedURLs, skipping repos that already have the group.
// Returns the number of repos actually modified.
func applyGroupToRepos(cfg *config.Config, groupName string, selectedURLs []string) int {
	selected := make(map[string]bool, len(selectedURLs))
	for _, u := range selectedURLs {
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
			cfg.Repos[i].Groups = append(cfg.Repos[i].Groups, config.Group{
				Name:     groupName,
				Selected: true,
			})
			modified++
		}
	}
	return modified
}

// removeGroupFromAllRepos removes groupName from every repo that has it.
// Returns the total number of group entries removed (0 means the group was
// not found anywhere).
func removeGroupFromAllRepos(cfg *config.Config, groupName string) int {
	removed := 0
	for i, repo := range cfg.Repos {
		var kept []config.Group
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

// uniqueGroupNames returns the deduplicated list of group names seen across
// all repos, in the order they are first encountered. Returns an empty
// (non-nil) slice when no groups are defined.
func uniqueGroupNames(cfg *config.Config) []string {
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
