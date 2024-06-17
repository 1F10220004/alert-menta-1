package main

import (
	"flag"
	"log"
	"os"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/github"
	"github.com/3-shake/alert-menta/internal/utils"
)

func main() {
	// Get command line arguments
	var (
		repo        = flag.String("repo", "", "Repository name")
		owner       = flag.String("owner", "", "Repository owner")
		issueNumber = flag.Int("issue", 0, "Issue number")
		command     = flag.String("command", "", "Command to be executed by AI")
		configFile  = flag.String("config", "./internal/config/config.yaml", "Configuration file")
		gh_token    = flag.String("github-token", "", "GitHub token")
		oai_key     = flag.String("api-key", "", "OpenAI api key")
	)
	flag.Parse()

	if *repo == "" || *owner == "" || *issueNumber == 0 || *gh_token == "" || *oai_key == "" || *command == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta main] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	// Pre-define variables for error handling
	var err error

	// Read the configuration file
	cfg, err := utils.NewConfig(*configFile)
	if err != nil {
		logger.Fatalf("Error creating comment: %s", err)
	}

	// Create a GitHub Issues instance. From now on, you can control GitHub from this instance.
	issue := github.NewIssue(*owner, *repo, *issueNumber, *gh_token)

	// Get Issue's information(e.g. Title, Body) and add them to the user prompt except for comments by Actions.
	title, _ := issue.GetTitle()
	body, _ := issue.GetBody()
	if cfg.System.Debug.Log_level == "debug" {
		logger.Println("Title:", *title)
		logger.Println("Body:", *body)
	}
	user_prompt := "Title:" + *title + "\n"
	user_prompt += "Body:" + *body + "\n"

	// Get comments under the Issue and add them to the user prompt except for comments by Actions.
	comments, _ := issue.GetComments()
	for _, v := range comments {
		if *v.User.Login == "github-actions[bot]" {
			continue
		}
		if cfg.System.Debug.Log_level == "debug" {
			logger.Printf("%s: %s", *v.User.Login, *v.Body)
		}
		user_prompt += *v.User.Login + ":" + *v.Body + "\n"
	}

	user_prompt += "----------\nBelow is the source code for the repository.  Please use the code below to help you answer the issue, including how to respond to the issue.\n"
	files, _ := utils.GetAllFiles("./")
	for _, file := range files {
		user_prompt += file.Path + ":" + file.Data + "\n"
	}

	// Set system prompt
	system_prompt := cfg.Ai.Commands[*command].System_prompt

	// Get response from OpenAI
	logger.Println("\x1b[34mPrompt: |\n", system_prompt, user_prompt, "\x1b[0m")
	ai := ai.NewOpenAIClient(*oai_key, cfg.Ai.Model)
	comment, _ := ai.GetResponse(system_prompt + user_prompt)
	logger.Println("\x1b[32mResponse: |\n", comment, "\x1b[0m")

	// Post a comment on the Issue
	err = issue.PostComment(comment)
	if err != nil {
		logger.Fatalf("Error creating comment: %s", err)
	}
}
