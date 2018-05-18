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
