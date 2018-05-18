/*
# `sanitize_repos_perms` script

This script provides an way to make sure that a given team has rights on all
the github repositories of a given organization and adds the team with push permissions if not.

Optionally you can use the `-admin-team` flag to ensure that all the teams other than this one
should have at most push permissions on the repos. Any permissions higher than that for the other
teams will be downgraded to push permissions.

Usage:

				go get "github.com/aerostitch/joeyzTools/github/sanitize_repos_perms"

Arguments:

  -admin-team string
        If specified, every teams other than this team having admin permissions on a repo will see its permissions downgraded to push.
  -organization string
        Name of the organization you want to work on.
  -team string
        Name of the team you want to make sure as at least some rights on every repositories of your org.

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
	expectedOrg       = flag.String("organization", "", "Name of the organization you want to work on.")
	expectedTeam      = flag.String("team", "", "Name of the team you want to make sure as at least some rights on every repositories of your org.")
	expectedAdminTeam = flag.String("admin-team", "", "If specified, every teams other than this team having admin permissions on a repo will see its permissions downgraded to push.")
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

// checkTeam makes sure a the given team ID has rights on the repo. If it has,
// do nothing, if not, add push rights.
// If adminTeam is not empty, makes sure than no other team has rights on the repo
func checkTeam(org, repoName, adminTeam string, expectedTeamID int64) error {
	opt := &github.ListOptions{}
	isTeamIn := false
	for {
		teams, resp, err := client.Repositories.ListTeams(ctx, org, repoName, opt)
		if err != nil {
			return err
		}
		for _, team := range teams {
			if expectedTeamID == team.GetID() {
				isTeamIn = true
			}
			if adminTeam != "" && team.GetName() != adminTeam && team.GetPermission() == "admin" {
				log.Printf("Downgrading %s permissions for %s - https://github.com/%s/%s\n", team.GetPermission(), team.GetName(), org, repoName)
				setRepoTeam(team.GetID(), org, repoName, "push")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	if !isTeamIn {
		return setRepoTeam(expectedTeamID, org, repoName, "push")
	}
	return nil
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

func main() {
	flag.Parse()
	if *expectedOrg == "" || *expectedTeam == "" {
		log.Fatalf("Please make sure you specified both an -organization and a -team flag")
	}

	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("Unauthorized: No token present")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client = github.NewClient(tc)

	teamID, err := getTeamID(*expectedOrg, *expectedTeam)
	if err != nil {
		log.Fatalf("Error while pulling team ID for team %s: %s", *expectedTeam, err)
	}

	repos, err := listAllRepos(*expectedOrg, "private")
	if err != nil {
		log.Fatalf("Error while listing repos: %s", err)
	}

	for _, repo := range repos {
		log.Printf("%s\n", *repo.Name)
		if err := checkTeam(*expectedOrg, *repo.Name, *expectedAdminTeam, teamID); err != nil {
			log.Fatalf("Error while checking team for https://github.com/%s/%s: %s", *expectedOrg, *repo.Name, err)
		}
	}
}
