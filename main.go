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
	Token    string `env:"GITHUB_TOKEN" long:"token" description:"GitHub access token" required:"true"`
	OrgName  string `env:"GITHUB_ORG" long:"org" description:"GitHub organization name" required:"true"`
	Template string `env:"TEMPLATE" long:"template" description:"Go template" default:""`
	Output   string `env:"OUTPUT" long:"output" description:"Output file" default:"output/graph.dot"`
}

func main() {
	var cfg config
	_, err := flags.Parse(&cfg)
	if err != nil {
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
	members, err := processor.Members(cfg.OrgName)
	if err != nil {
		log.Fatalf("Error getting members: %v", err)
	}

	log.Println("Getting organization teams...")
	teams, parents, err := processor.Teams(cfg.OrgName, orgID)
	if err != nil {
		log.Fatalf("Error getting teams: %v", err)
	}

	membersWitoutTeam := FindMembersWithoutTeam(teams, members)
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

func FindSubsets(teams map[string][]string) map[string]string {
	subsets := make(map[string]string)

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

			if subsets[team] == otherTeam || subsets[otherTeam] == team {
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
				subsets[team] = otherTeam
			}

			if len(commonMembers) == len(otherMembers) {
				subsets[otherTeam] = team
			}
		}
	}

	return subsets
}

func FindMembersWithoutTeam(teams map[string][]string, members []string) []string {
	var existingMembers = make(map[string]struct{})
	for _, teamMembers := range teams {
		for _, member := range teamMembers {
			existingMembers[member] = struct{}{}
		}
	}

	var membersWithoutTeam []string
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
	Subsets            map[string]string
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
