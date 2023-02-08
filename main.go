package main

import (
	"context"
	_ "embed"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/google/go-github/v48/github"
	flags "github.com/jessevdk/go-flags"
	"golang.org/x/oauth2"
)

type config struct {
	HideTeamMembers bool   `env:"HIDE_TEAM_MEMBERS" long:"show_team_members" description:"Show Team Members on the diagram" required:"true"`
	Token           string `env:"GITHUB_TOKEN" long:"token" description:"GitHub access token" required:"true"`
	OrgName         string `env:"GITHUB_ORG" long:"org" description:"GitHub organization name" required:"true"`
	Template        string `env:"TEMPLATE" long:"template" description:"Go template (optional)" default:""`
	Output          string `env:"OUTPUT" long:"output" description:"Output file" default:"output/graph.dot"`
}

func main() {
	var cfg config
	_, err := flags.Parse(&cfg)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
			return
		}
		log.Fatalf("Error parsing flags: %v", err)
	}

	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	processor := Processor{
		Context:              ctx,
		OrganizationsService: client.Organizations,
		TeamsService:         client.Teams,
	}

	log.Println("Getting organization ID...")
	orgID, err := processor.GetOrganizationID(cfg.OrgName)
	if err != nil {
		log.Fatalf("Error checking organization access: %v", err)
	}

	log.Println("Getting organization members...")
	members, err := processor.Members(cfg.OrgName, cfg.HideTeamMembers)
	if err != nil {
		log.Fatalf("Error getting members: %v", err)
	}

	log.Println("Getting organization teams...")
	teams, parents, err := processor.Teams(cfg.OrgName, cfg.HideTeamMembers, orgID)
	if err != nil {
		log.Fatalf("Error getting teams: %v", err)
	}

	membersWitoutTeam := FindMembersWithoutTeam(cfg.HideTeamMembers, teams, members)
	if len(membersWitoutTeam) > 0 {
		teams["NO_TEAM"] = membersWitoutTeam
	}

	log.Println("Rendering template...")
	if err := renderTemplate(cfg.Template, cfg.Output, data{
		Teams:   teams,
		Parents: parents,
		Members: members,
		Subsets: FindSubsets(teams),
	}); err != nil {
		log.Fatalf("Error rendering template: %v", err)
	}

	log.Println("Done!")
}

type subsets map[string]map[string]struct{}

func (s subsets) IsSubset(team string, otherTeam string) bool {
	if teamSubsets, ok := s[team]; ok {
		if _, ok := teamSubsets[otherTeam]; ok {
			return true
		}
	}

	return false
}

func (s subsets) AddSubset(team string, otherTeam string) {
	if _, ok := s[team]; !ok {
		s[team] = map[string]struct{}{}
	}

	s[team][otherTeam] = struct{}{}
}

func (s subsets) GetSubsets(team string) []string {
	var subsets []string
	if teamSubsets, ok := s[team]; ok {
		for subset := range teamSubsets {
			subsets = append(subsets, subset)
		}
	}

	return subsets
}

func FindSubsets(teams map[string][]string) subsets {
	var s subsets = map[string]map[string]struct{}{}

	for team, members := range teams {
		if len(members) == 0 {
			continue
		}

		for otherTeam, otherMembers := range teams {
			if team == otherTeam {
				continue
			}

			if len(otherMembers) == 0 {
				continue
			}

			// check if we already found a subset
			if s.IsSubset(team, otherTeam) || s.IsSubset(otherTeam, team) {
				// already found a subset
				continue
			}

			// find common members
			var commonMembers []string
			for _, member := range members {
				for _, otherMember := range otherMembers {
					if member == otherMember {
						commonMembers = append(commonMembers, member)
					}
				}
			}

			if len(commonMembers) == 0 {
				continue
			}

			if len(commonMembers) == len(members) {
				s.AddSubset(team, otherTeam)
			}

			if len(commonMembers) == len(otherMembers) {
				s.AddSubset(otherTeam, team)
			}
		}
	}

	return s
}

func FindMembersWithoutTeam(hideTeamMembers bool, teams map[string][]string, members []string) []string {
	var membersWithoutTeam []string
	if hideTeamMembers {
		return membersWithoutTeam
	}

	var existingMembers = make(map[string]struct{})
	for _, teamMembers := range teams {
		for _, member := range teamMembers {
			existingMembers[member] = struct{}{}
		}
	}

	for _, member := range members {
		if _, ok := existingMembers[member]; !ok {
			membersWithoutTeam = append(membersWithoutTeam, member)
		}
	}

	return membersWithoutTeam
}

type data struct {
	Teams              map[string][]string
	Parents            map[string]string
	Members            []string
	Subsets            subsets
	MembersWithoutTeam []string
}

var funcMap = template.FuncMap{
	"join": func(a []string, sep string) string {
		return strings.Join(a, sep)
	},
}

//go:embed dot.tmpl
var dotTemplate string

func renderTemplate(tmpl, output string, data data) error {
	var err error

	t := template.New(tmpl).Funcs(funcMap)

	if tmpl == "" {
		t, err = t.Parse(dotTemplate)
	} else {
		t, err = t.ParseFiles(tmpl)
	}

	if err != nil {
		return err
	}

	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.ExecuteTemplate(f, tmpl, data)
}
