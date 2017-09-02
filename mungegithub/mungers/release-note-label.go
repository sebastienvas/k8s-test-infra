/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mungers

import (
	"fmt"

	"k8s.io/kubernetes/pkg/util/sets"
	"k8s.io/test-infra/mungegithub/features"
	"k8s.io/test-infra/mungegithub/github"
	"k8s.io/test-infra/mungegithub/options"

	"regexp"
	"strings"

	"github.com/golang/glog"
	githubapi "github.com/google/go-github/github"
)

const (
	releaseNoteLabeler = "release-note-label"

	// deprecatedReleaseNoteLabelNeeded is the previous version of the
	// releaseNotLabelNeeded label, which we continue to honor for the
	// time being
	deprecatedReleaseNoteLabelNeeded = "release-note-label-needed"

	releaseNoteLabelNeeded    = "do-not-merge/release-note-label-needed"
	releaseNote               = "release-note"
	releaseNoteNone           = "release-note-none"
	releaseNoteActionRequired = "release-note-action-required"

	releaseNoteFormat = `Adding %s because the release note process has not been followed.
%s`
	releaseNoteSuffixFormat = `One of the following labels is required %q, %q, or %q.
Please see: https://github.com/kubernetes/community/blob/master/contributors/devel/pull-requests.md#write-release-notes-if-needed.`
	parentReleaseNoteFormat = `The 'parent' PR of a cherry-pick PR must have one of the %q or %q labels, or this PR must follow the standard/parent release note labeling requirement.`

	noReleaseNoteComment = "none"
	actionRequiredNote   = "action required"
)

var (
	releaseNoteSuffix         = fmt.Sprintf(releaseNoteSuffixFormat, releaseNote, releaseNoteActionRequired, releaseNoteNone)
	releaseNoteBody           = fmt.Sprintf(releaseNoteFormat, releaseNoteLabelNeeded, releaseNoteSuffix)
	deprecatedReleaseNoteBody = fmt.Sprintf(releaseNoteFormat, deprecatedReleaseNoteLabelNeeded, releaseNoteSuffix)
	parentReleaseNoteBody     = fmt.Sprintf(parentReleaseNoteFormat, releaseNote, releaseNoteActionRequired)
	noteMatcherRE             = regexp.MustCompile(`(?s)(?:Release note\*\*:\s*(?:<!--[^<>]*-->\s*)?` + "```(?:release-note)?|```release-note)(.+?)```")
)

// ReleaseNoteLabel will add the releaseNoteMissingLabel to a PR which has not
// set one of the appropriate 'release-note-*' labels but has LGTM
type ReleaseNoteLabel struct {
	config *github.Config
}

func init() {
	r := &ReleaseNoteLabel{}
	RegisterMungerOrDie(r)
	RegisterStaleIssueComments(r)
}

// Name is the name usable in --pr-mungers
func (r *ReleaseNoteLabel) Name() string { return releaseNoteLabeler }

// RequiredFeatures is a slice of 'features' that must be provided
func (r *ReleaseNoteLabel) RequiredFeatures() []string { return []string{} }

// Initialize will initialize the munger
func (r *ReleaseNoteLabel) Initialize(config *github.Config, features *features.Features) error {
	r.config = config
	return nil
}

// EachLoop is called at the start of every munge loop
func (r *ReleaseNoteLabel) EachLoop() error { return nil }

// RegisterOptions registers options for this munger; returns any that require a restart when changed.
func (r *ReleaseNoteLabel) RegisterOptions(opts *options.Options) sets.String { return nil }

func (r *ReleaseNoteLabel) prMustFollowRelNoteProcess(obj *github.MungeObject) bool {
	boolean, ok := obj.IsForBranch("master")

	if !ok || boolean {
		return true
	}

	parents := getCherrypickParentPRs(obj, r.config)
	// if it has no parents it needs to follow the release note process
	if len(parents) == 0 {
		return true
	}

	for _, parent := range parents {
		// If the parent didn't set a release note, the CP must
		if !parent.HasLabel(releaseNote) &&
			!parent.HasLabel(releaseNoteActionRequired) {
			if !obj.HasLabel(releaseNoteLabelNeeded) {
				obj.WriteComment(parentReleaseNoteBody)
			}
			return true
		}
	}
	// All of the parents set the releaseNote or releaseNoteActionRequired label,
	// so this cherrypick PR needs to do nothing.
	return false
}

func (r *ReleaseNoteLabel) ensureNoRelNoteNeededLabel(obj *github.MungeObject) {
	if obj.HasLabel(releaseNoteLabelNeeded) {
		obj.RemoveLabel(releaseNoteLabelNeeded)
	}
	if obj.HasLabel(deprecatedReleaseNoteLabelNeeded) {
		obj.RemoveLabel(deprecatedReleaseNoteLabelNeeded)
	}
}

// Munge is the workhorse the will actually make updates to the PR
func (r *ReleaseNoteLabel) Munge(obj *github.MungeObject) {
	if !obj.IsPR() {
		return
	}

	if releaseNoteAlreadyAdded(obj) {
		r.ensureNoRelNoteNeededLabel(obj)
		return
	}

	if !r.prMustFollowRelNoteProcess(obj) {
		r.ensureNoRelNoteNeededLabel(obj)
		return
	}

	labelToAdd := determineReleaseNoteLabel(obj)
	if labelToAdd == releaseNoteLabelNeeded {
		if !obj.HasLabel(releaseNoteLabelNeeded) {
			obj.WriteComment(releaseNoteBody)
		}
	} else {
		//going to apply some other release-note-label
		r.ensureNoRelNoteNeededLabel(obj)
	}
	if !obj.HasLabel(labelToAdd) {
		obj.AddLabel(labelToAdd)
	}
	return
}

// determineReleaseNoteLabel returns the label to be added if
// correctly implemented in the PR template.  returns nil otherwise
func determineReleaseNoteLabel(obj *github.MungeObject) string {
	if obj.Issue.Body == nil {
		return releaseNoteLabelNeeded
	}
	potentialMatch := getReleaseNote(*obj.Issue.Body)
	return chooseLabel(potentialMatch)
}

// getReleaseNote returns the release note from a PR body
// assumes that the PR body followed the PR template
func getReleaseNote(body string) string {
	potentialMatch := noteMatcherRE.FindStringSubmatch(body)
	if potentialMatch == nil {
		return ""
	}
	return strings.TrimSpace(potentialMatch[1])
}

func chooseLabel(composedReleaseNote string) string {
	composedReleaseNote = strings.ToLower(strings.TrimSpace(composedReleaseNote))

	if composedReleaseNote == "" {
		return releaseNoteLabelNeeded
	}
	if composedReleaseNote == noReleaseNoteComment {
		return releaseNoteNone
	}
	if strings.Contains(composedReleaseNote, actionRequiredNote) {
		return releaseNoteActionRequired
	}
	return releaseNote
}

func releaseNoteAlreadyAdded(obj *github.MungeObject) bool {
	return obj.HasLabel(releaseNote) ||
		obj.HasLabel(releaseNoteActionRequired) ||
		obj.HasLabel(releaseNoteNone)
}

func (r *ReleaseNoteLabel) isStaleIssueComment(obj *github.MungeObject, comment *githubapi.IssueComment) bool {
	if !obj.IsRobot(comment.User) {
		return false
	}
	if *comment.Body != releaseNoteBody && *comment.Body != parentReleaseNoteBody && *comment.Body != deprecatedReleaseNoteBody {
		return false
	}
	if !r.prMustFollowRelNoteProcess(obj) {
		glog.V(6).Infof("Found stale ReleaseNoteLabel comment")
		return true
	}
	stale := releaseNoteAlreadyAdded(obj)
	if stale {
		glog.V(6).Infof("Found stale ReleaseNoteLabel comment")
	}
	return stale
}

// StaleIssueComments returns a slice of stale issue comments.
func (r *ReleaseNoteLabel) StaleIssueComments(obj *github.MungeObject, comments []*githubapi.IssueComment) []*githubapi.IssueComment {
	return forEachCommentTest(obj, comments, r.isStaleIssueComment)
}
