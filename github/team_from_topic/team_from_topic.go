/*
# `teams_from_topic` script

This script provides an way to assign permissions to a team on all the repos of a github organization
that has a given tag.

Usage:

				go get "github.com/aerostitch/joeyzTools/github/team_from_topic"


				./team_from_topic.go -organization myOrg -permission admin -team "web team" -topic "webserver"

Arguments:

  -organization string
        Name of the organization you want to work on.
  -permission string
        Permission (pull, push or admin) that you want to give to the team on the selected repositories of the org.
  -team string
        Name of the team you want to assign the permission to when the repo has the given topic.
  -topic string
        Topic that will be used to select the repositories of the organization to add the permissions to.

*/
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var client *github.Client
var ctx = context.Background()
var (
	providedOrg   = flag.String("organization", "", "Name of the organization you want to work on.")
	providedTeam  = flag.String("team", "", "Name of the team you want to assign the permission to when the repo has the given topic.")
	providedPerm  = flag.String("permission", "", "Permission (pull, push or admin) that you want to give to the team on the selected repositories of the org.")
	providedTopic = flag.String("topic", "", "Topic that will be used to select the repositories of the organization to add the permissions to.")
)

// listAllRepos returns all the repo with a given type (all, public, private, forks, sources, member) in a given org
func listAllRepos(org, reposType string) ([]*github.Repository, error) {
	opt := &github.RepositoryListByOrgOptions{
		Type:        reposType,
		ListOptions: github.ListOptions{},
	}
	// get all pages of results
	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			return allRepos, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allRepos, nil
}

// setRepoTeam sets the given permission level to the given team on the given
// repository
func setRepoTeam(teamID int64, org, repoName, perm string) error {
	permStruct := &github.OrganizationAddTeamRepoOptions{Permission: perm}
	_, err := client.Organizations.AddTeamRepo(ctx, teamID, org, repoName, permStruct)
	return err
}

// getTeamID get the ID of a given org's team from its name
func getTeamID(org, teamName string) (int64, error) {
	opts := &github.ListOptions{}
	for {
		teams, resp, err := client.Organizations.ListTeams(ctx, org, opts)
		if err != nil {
			return -1, err
		}
		for _, t := range teams {
			if t.GetName() == teamName {
				return t.GetID(), nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return -1, nil
}

// filterRepo returns true if the repo is ok to be processed (has the topic) and
// false if not. Done in a separate function so that we can expand later
func filterRepo(repo *github.Repository, topic string) bool {
	if repo == nil {
		return false
	}
	for _, t := range repo.Topics {
		if t == topic {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()
	if *providedOrg == "" || *providedTeam == "" || *providedTopic == "" || *providedPerm == "" {
		log.Fatalf("Please make sure you specified -organization, -team, -topic and -permission flags")
	}

	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("Unauthorized: No token present")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client = github.NewClient(tc)

	teamID, err := getTeamID(*providedOrg, *providedTeam)
	if err != nil {
		log.Fatalf("Error while pulling team ID for team %s: %s", *providedTeam, err)
	}

	repos, err := listAllRepos(*providedOrg, "private")
	if err != nil {
		log.Fatalf("Error while listing repos: %s", err)
	}

	for _, repo := range repos {
		if ok := filterRepo(repo, *providedTopic); !ok {
			continue
		}
		log.Printf("%s\n", *repo.Name)
		if err := setRepoTeam(teamID, *providedOrg, *repo.Name, *providedPerm); err != nil {
			log.Fatalf("Error while checking team for https://github.com/%s/%s: %s", *providedOrg, *repo.Name, err)
		}
	}
}
