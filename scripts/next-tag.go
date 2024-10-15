package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	// Get current tag
	currentTag, err := getCurrentGitTag()
	if err != nil {
		log.Fatal(err)
	}

	// Get recent commits since the last tag
	commits, err := getGitCommitsSinceTag(currentTag)
	if err != nil {
		log.Fatal(err)
	}

	// Determine the next version
	nextTag, err := getNextTag(currentTag, commits)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(nextTag)
}

// getCurrentGitTag returns the latest Git tag
func getCurrentGitTag() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("could not get current tag: %v", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// getGitCommitsSinceTag returns a list of commit messages since the given tag
func getGitCommitsSinceTag(tag string) ([]string, error) {
	cmd := exec.Command("git", "log", fmt.Sprintf("%s..HEAD", tag), "--pretty=%s")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("could not get commits: %v", err)
	}
	commits := strings.Split(strings.TrimSpace(out.String()), "\n")
	return commits, nil
}

// getNextTag determines the next tag based on conventional commits
func getNextTag(currentTag string, commits []string) (string, error) {
	// Parse the current version
	versionParts := strings.Split(strings.TrimPrefix(currentTag, "v"), ".")
	if len(versionParts) != 3 {
		return "", fmt.Errorf("invalid tag format")
	}

	major := atoi(versionParts[0])
	minor := atoi(versionParts[1])
	patch := atoi(versionParts[2])

	// Conventional commit patterns with optional parentheses
	featPattern := regexp.MustCompile(`^feat(\([^)]*\))?:`)
	fixPattern := regexp.MustCompile(`^fix(\([^)]*\))?:`)
	breakingPattern := regexp.MustCompile(`(?i)BREAKING CHANGE`)

	incrementedMajor, incrementedMinor, incrementedPatch := false, false, false

	// Analyze commits
	for _, commit := range commits {
		switch {
		case breakingPattern.MatchString(commit):
			incrementedMajor = true
		case featPattern.MatchString(commit):
			incrementedMinor = true
		case fixPattern.MatchString(commit):
			incrementedPatch = true
		}
	}

	// Determine next version based on commit types
	if incrementedMajor {
		major++
		minor = 0
		patch = 0
	} else if incrementedMinor {
		minor++
		patch = 0
	} else if incrementedPatch {
		patch++
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch), nil
}

// atoi is a helper function to convert string to int
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
