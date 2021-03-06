package simple

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/machinebox/graphql"
	"quozlet.net/birbbot/app/commands"
)

const githubAPI string = "https://api.github.com/graphql"

// https://docs.github.com/en/graphql/reference/mutations#createissue

var (
	client         *graphql.Client
	repositoryID   string
	bugLabelID     string
	errNoTicketIDs = errors.New("failed to get necessary IDs")
)

// Issue is a command to open a GitHub issue.
type Issue struct{}

// Check asserts that the GraphQL client connection could be made.
func (i Issue) Check() error {
	client = graphql.NewClient(githubAPI)
	req := graphql.NewRequest(`
	query ($name: String!, $owner: String!) {
		repository(name: $name, owner: $owner) {
		  id
		  label(name: "bug") {
			id
		  }
		}
	}	  
	`)
	req.Var("name", "BirbBot")
	req.Var("owner", "Quozlet")
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", os.Getenv("GITHUB_TOKEN")))

	var repoInfo RepoData
	if err := client.Run(context.Background(), req, &repoInfo); err != nil {
		log.Println(err)

		return err
	}

	repositoryID = repoInfo.Repository.ID
	bugLabelID = repoInfo.Repository.Label.ID

	if repositoryID == "" || bugLabelID == "" {
		return errNoTicketIDs
	}

	return nil
}

// ProcessMessage will attempt to create an issue with the given text.
func (i Issue) ProcessMessage(
	response chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
) *commands.CommandError {
	content := strings.Join(strings.Fields(m.Content)[1:], " ")
	if len(content) == 0 {
		return commands.NewError("Cannot make an issue with no information provided")
	}

	req := graphql.NewRequest(`
	mutation CreateIssue($repository: ID!, $title: String!, $body: String!, $label: [ID!]) {
		createIssue(input: {repositoryId: $repository, title: $title, body: $body, labelIds: $label}) {
		  issue {
			url
			id
		  }
		  clientMutationId
		}
	}
	`)
	req.Var("repository", repositoryID)

	issue := strings.Split(content, ".")
	req.Var("title", issue[0])
	req.Var("body", strings.TrimSpace(strings.Join(issue[1:], "")))

	label := []string{}

	if strings.Fields(m.Content)[0] == "!bug" {
		label = append(label, bugLabelID)
	}

	req.Var("label", label)

	var issueData IssueData

	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", os.Getenv("GITHUB_TOKEN")))

	if commandError := commands.CreateCommandError(
		"Failed to make the issue",
		client.Run(context.Background(), req, &issueData),
	); commandError != nil {
		return commandError
	}

	log.Printf("%+v", issueData)
	response <- commands.MessageResponse{
		ChannelID: m.ChannelID,
		Message:   fmt.Sprintf("Successfully created!\n%s", issueData.CreateIssue.Issue.URL),
	}

	return nil
}

// CommandList returns the invocable aliases for the Issue Command.
func (i Issue) CommandList() []string {
	return []string{"issue", "bug"}
}

// Help gives help information for the Issue Command.
func (i Issue) Help() string {
	return "Opens a GitHub issue with the provided text.\n" +
		"- `issue` opens an issue with no tags\n" +
		"- `bug` is a shorthand for opening an issue and tagging as a bug"
}

// Label for the bug tag.
type Label struct {
	ID string `json:"id"`
}

// Repository information for this repo.
type Repository struct {
	ID    string `json:"id"`
	Label Label  `json:"label"`
}

// RepoData is the wrapper object for Repo information.
type RepoData struct {
	Repository Repository `json:"repository"`
}

// IssueInfo for the newly created issue.
type IssueInfo struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

// CreateIssue is the result of a successful creation.
type CreateIssue struct {
	Issue            IssueInfo   `json:"issue"`
	ClientMutationID interface{} `json:"clientMutationId"`
}

// IssueData is the wrapper object for issue creation information.
type IssueData struct {
	CreateIssue CreateIssue `json:"createIssue"`
}
