package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-github/v48/github"
)

const perPage = 100

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
	HideMembers          bool
}

func (p *Processor) GetOrganizationID(orgName string) (int64, error) {
	org, _, err := p.OrganizationsService.Get(p.Context, orgName)
	if err != nil {
		return 0, err
	}

	return *org.ID, nil
}

func (p *Processor) Members(orgName string) ([]string, error) {
	var currentPage = 0
	var result []string

	for {
		members, response, err := p.OrganizationsService.ListMembers(
			p.Context,
			orgName,
			&github.ListMembersOptions{
				ListOptions: github.ListOptions{
					Page:    currentPage,
					PerPage: perPage,
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list org members: %w", err)
		}

		for _, member := range members {
			result = append(result, *member.Login)
		}

		if currentPage == response.LastPage {
			break
		}

		currentPage = response.NextPage
	}

	return result, nil
}

func (p *Processor) Teams(orgName string, orgID int64) (teamMembers map[string][]string, teamParents map[string]string, err error) {
	teams, err := p.getTeamsPaginated(orgName)
	if err != nil {
		return nil, nil, err
	}

	teamMembers = make(map[string][]string)
	teamParents = make(map[string]string)
	for _, team := range teams {
		if !p.HideMembers {
			members, err := p.getTeamMembersPaginated(orgID, team)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to list team members for %q: %w", *team.Name, err)
			}

			var returnMembers []string
			for _, member := range members {
				returnMembers = append(returnMembers, *member.Login)
			}

			sort.Slice(returnMembers, func(i, j int) bool { return strings.ToLower(returnMembers[i]) < strings.ToLower(returnMembers[j]) })

			teamMembers[*team.Name] = returnMembers
		}

		if team.Parent != nil {
			teamParents[*team.Name] = *team.Parent.Name
		}
	}

	return teamMembers, teamParents, nil
}

func (p *Processor) getTeamsPaginated(orgName string) ([]*github.Team, error) {
	currentPage := 0

	var allTeams []*github.Team
	for {
		teams, response, err := p.TeamsService.ListTeams(
			p.Context,
			orgName,
			&github.ListOptions{
				Page:    currentPage,
				PerPage: perPage,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list teams: %w", err)
		}

		allTeams = append(allTeams, teams...)

		if currentPage == response.LastPage {
			break
		}

		currentPage = response.NextPage
	}

	return allTeams, nil
}

func (p *Processor) getTeamMembersPaginated(orgID int64, team *github.Team) ([]*github.User, error) {
	currentPage := 0

	var allMembers []*github.User
	for {
		members, response, err := p.TeamsService.ListTeamMembersByID(
			p.Context,
			orgID,
			*team.ID,
			&github.TeamListTeamMembersOptions{
				ListOptions: github.ListOptions{
					Page:    currentPage,
					PerPage: perPage,
				},
			},
		)
		if err != nil {
			return nil, err
		}

		allMembers = append(allMembers, members...)

		if currentPage == response.LastPage {
			break
		}

		currentPage = response.NextPage
	}

	return allMembers, nil
}
