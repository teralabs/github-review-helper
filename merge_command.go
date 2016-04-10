package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/salemove/github-review-helper/git"
)

const (
	MergingLabel    = "merging"
	MergeRetryLimit = 3
)

func isMergeCommand(comment string) bool {
	return strings.TrimSpace(comment) == "!merge"
}

func handleMergeCommand(issueComment IssueComment, issues Issues, pullRequests PullRequests,
	repositories Repositories, gitRepos git.Repos) Response {
	errResp := addLabel(issueComment.Repository, issueComment.IssueNumber, MergingLabel, issues)
	if errResp != nil {
		return errResp
	}
	return mergeWithRetry(MergeRetryLimit, issueComment, issues, pullRequests, repositories, gitRepos)
}

func mergeWithRetry(nrOfRetries int, issueComment IssueComment, issues Issues, pullRequests PullRequests,
	repositories Repositories, gitRepos git.Repos) Response {
	pr, errResp := getPR(issueComment, pullRequests)
	if errResp != nil {
		return errResp
	} else if *pr.Merged {
		log.Printf("PR #%d already merged. Removing the '%s' label.\n", issueComment.IssueNumber, MergingLabel)
		errResp = removeLabel(issueComment.Repository, issueComment.IssueNumber, MergingLabel, issues)
		if errResp != nil {
			return errResp
		}
		return SuccessResponse{}
	} else if !*pr.Mergeable {
		return SuccessResponse{}
	}
	state, statuses, errResp := getStatuses(pr, repositories)
	if errResp != nil {
		return errResp
	} else if state == "pending" && containsPendingSquashStatus(statuses) {
		return squashAndReportFailure(pr, gitRepos, repositories)
	} else if state != "success" {
		log.Printf("PR #%d has pending and/or failed statuses. Not merging.\n", issueComment.IssueNumber)
		return SuccessResponse{}
	}
	err := merge(issueComment.Repository, issueComment.IssueNumber, pullRequests)
	if err == OutdatedMergeRefError && nrOfRetries > 0 {
		return mergeWithRetry(nrOfRetries-1, issueComment, issues, pullRequests, repositories, gitRepos)
	} else if err != nil {
		message := fmt.Sprintf("Failed to merge PR #%d", issueComment.IssueNumber)
		return ErrorResponse{err, http.StatusBadGateway, message}
	}
	log.Printf("PR #%d successfully merged. Removing the '%s' label.\n", issueComment.IssueNumber, MergingLabel)
	errResp = removeLabel(issueComment.Repository, issueComment.IssueNumber, MergingLabel, issues)
	if errResp != nil {
		return errResp
	}
	return SuccessResponse{}
}

func containsPendingSquashStatus(statuses []github.RepoStatus) bool {
	for _, status := range statuses {
		if *status.Context == githubStatusSquashContext && *status.State == "pending" {
			return true
		}
	}
	return false
}
