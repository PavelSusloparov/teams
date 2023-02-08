package main

import (
	"context"
	"log"
	"sort"
	"strings"

	"github.com/google/go-github/v48/github"
)

type organizationsService interface {
	Get(ctx context.Context, org string) (*github.Organization, *github.Response, error)
	ListMembers(ctx context.Context, org string, opt *github.ListMembersOptions) ([]*github.User, *github.Response, error)
}

type teamsService interface {
	ListTeams(ctx context.Context, org string, opt *github.ListOptions) ([]*github.Team, *github.Response, error)
	ListTeamMembersByID(ctx context.Context, orgID, teamID int64, opts *github.TeamListTeamMembersOptions) ([]*github.User, *github.Response, error)
}

type Processor struct {
	Context              context.Context
	OrganizationsService organizationsService
	TeamsService         teamsService
}

func (p *Processor) GetOrganizationID(orgName string) (int64, error) {
	org, _, err := p.OrganizationsService.Get(p.Context, orgName)
	if err != nil {
		return 0, err
	}

	return *org.ID, nil
}

func (p *Processor) Members(orgName string) ([]string, error) {
	members, _, err := p.OrganizationsService.ListMembers(p.Context, orgName, nil)
	if err != nil {
		return nil, err
	}

	var returnMembers []string
	for _, member := range members {
		returnMembers = append(returnMembers, *member.Login)
	}

	return returnMembers, nil
}

func check(err error) {
	if err != nil {
		log.Fatalf("Something went wrong. Error message - %q", err)
	}
}

func (p *Processor) getTeamsPaginated(orgName string) []*github.Team {
	nextPage := 0
	var allTeams []*github.Team
	for {
		opt := github.ListOptions{Page: nextPage, PerPage: 30}
		teams, response, err := p.TeamsService.ListTeams(p.Context, orgName, &opt)
		check(err)
		for _, currentTeam := range teams {
			allTeams = append(allTeams, currentTeam)
		}
		if response.NextPage == response.LastPage {
			break
		} else {
			nextPage = response.NextPage
		}
	}
	return allTeams
}

func (p *Processor) Teams(orgName string, orgID int64) (teamMembers map[string][]string, teamParents map[string]string, err error) {
	teams := p.getTeamsPaginated(orgName)

	if err != nil {
		return nil, nil, err
	}

	teamMembers = make(map[string][]string)
	teamParents = make(map[string]string)
	for _, team := range teams {
		members, _, err := p.TeamsService.ListTeamMembersByID(p.Context, orgID, *team.ID, nil)
		if err != nil {
			return nil, nil, err
		}

		var returnMembers []string
		for _, member := range members {
			returnMembers = append(returnMembers, *member.Login)
		}

		sort.Slice(returnMembers, func(i, j int) bool { return strings.ToLower(returnMembers[i]) < strings.ToLower(returnMembers[j]) })

		teamMembers[*team.Name] = returnMembers
		if team.Parent != nil {
			teamParents[*team.Name] = *team.Parent.Name
		}
	}

	return teamMembers, teamParents, nil
}
