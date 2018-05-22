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

