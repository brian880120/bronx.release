package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"bronx.release/common"
	"bronx.release/model"

	"github.com/xanzy/go-gitlab"
)

const REGEX_TICKET_ID_PATTERN = "[\\[]?[a-zA-Z]*[-][0-9]*[\\]]?"
const REGEX_TICKET_Id = "[a-zA-Z]*[-][0-9]*"

type ReleaseBuilder struct {
	git            *gitlab.Client
	packageVersion string
	releaseNote    string
	releaseVersion string
	pipelineId     int
	jobId          int
}

// Initialize releaseBuilder
func (rb *ReleaseBuilder) Initialize() {
	token := os.Getenv("GITLAB_TOKEN")

	git, err := gitlab.NewClient(token, gitlab.WithBaseURL("https://gitlab.lblw.ca/api/v4"))

	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	rb.git = git
}

func (rb *ReleaseBuilder) Run() {
	go func() {
		rb.getReleaseNote()
	}()
	rb.startBranchProtect()
	rb.prepareMaster()
	rb.getPackageVersion()
	rb.createTag()
	// have wait process
	rb.getPipeline()
	// have wait process
	rb.getJob()
	rb.getArtifact()
	rb.updateTag()
	rb.endBranchProtect()

	fmt.Println(rb.releaseVersion)
	fmt.Println(rb.packageVersion)
	fmt.Println(rb.pipelineId)
	// TODO: delete bronx
}

func (rb *ReleaseBuilder) prepareMaster() {
	fmt.Println("push to master")
	cmd := exec.Command("bash", "push-master.sh")
	cmd.Stdin = strings.NewReader("")

	var out bytes.Buffer

	cmd.Stdout = &out

	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Output \n", out.String())
}

func (rb *ReleaseBuilder) getPackageVersion() {
	fmt.Println("getting package json version")
	var packageInfo model.PackageInfo

	file, err := ioutil.ReadFile("bronx/client/package.json")

	if err != nil {
		log.Fatal(err)
	}

	json.Unmarshal(file, &packageInfo)

	rb.packageVersion = packageInfo.Version
}

func (rb *ReleaseBuilder) createTag() {
	time.Sleep(10 * time.Second)
	fmt.Println("create release tag")
	tagName := "test-" + rb.packageVersion
	opt := &gitlab.CreateTagOptions{
		TagName:            gitlab.String(tagName),
		Ref:                gitlab.String("test-master"),
		Message:            gitlab.String("test message"),
		ReleaseDescription: gitlab.String("test release"),
	}

	tag, _, _ := rb.git.Tags.CreateTag(290, opt)
	fmt.Println(tag.Name)
}

func (rb *ReleaseBuilder) updateTag() {
	fmt.Println("update tag release note")
	tagName := "test-" + rb.packageVersion
	opt := &gitlab.UpdateReleaseNoteOptions{
		Description: gitlab.String(rb.releaseNote),
	}

	rb.git.Tags.UpdateReleaseNote(290, tagName, opt)
}

func (rb *ReleaseBuilder) getReleaseNote() {
	fmt.Println("getting release note")
	var lastMR, currentMR *gitlab.MergeRequest
	var ticketIdPattern, title, ticketId string
	var labels []string
	regexTicketIdPattern := regexp.MustCompile(REGEX_TICKET_ID_PATTERN)
	regexTicketId := regexp.MustCompile(REGEX_TICKET_Id)

	sb := new(strings.Builder)

	opts := &gitlab.ListProjectMergeRequestsOptions{
		State:        gitlab.String("merged"),
		Scope:        gitlab.String("all"),
		TargetBranch: gitlab.String("develop"),
		Labels:       &gitlab.Labels{"Version Update"},
	}

	versionUpdateMRs, _, _ := rb.git.MergeRequests.ListProjectMergeRequests(290, opts)

	currentMR = versionUpdateMRs[0]
	lastMR = versionUpdateMRs[1]

	opts.Labels = nil
	opts.UpdatedAfter = lastMR.MergedAt
	opts.UpdatedBefore = currentMR.MergedAt

	targetMRs, _, _ := rb.git.MergeRequests.ListProjectMergeRequests(290, opts)

	for _, mr := range targetMRs {
		ticketId = regexTicketId.FindString(mr.Title)
		if len(ticketId) > 0 {
			ticketIdPattern = regexTicketIdPattern.FindString(mr.Title)
			title = common.GetSubstringAfter(mr.Title, ticketIdPattern)
			labels = mr.Labels
			sb.WriteString(fmt.Sprintf("%s - %s - %s\n", ticketId, title, common.ParseLabel(labels)))
		}
	}

	rb.releaseNote = sb.String()
}

func (rb *ReleaseBuilder) startBranchProtect() {
	fmt.Println("protect branch")
	rb.git.ProtectedBranches.UnprotectRepositoryBranches(290, "test-develop")
	rb.git.ProtectedBranches.UnprotectRepositoryBranches(290, "test-master")

	opt := &gitlab.ProtectRepositoryBranchesOptions{
		Name:                      gitlab.String("test-develop"),
		PushAccessLevel:           gitlab.AccessLevel(gitlab.NoPermissions),
		MergeAccessLevel:          gitlab.AccessLevel(gitlab.NoPermissions),
		CodeOwnerApprovalRequired: gitlab.Bool(false),
	}

	rb.git.ProtectedBranches.ProtectRepositoryBranches(290, opt)
}

func (rb *ReleaseBuilder) endBranchProtect() {
	fmt.Println("reset protect branch")
	rb.git.ProtectedBranches.UnprotectRepositoryBranches(290, "test-develop")

	developOpt := &gitlab.ProtectRepositoryBranchesOptions{
		Name:                      gitlab.String("test-develop"),
		PushAccessLevel:           gitlab.AccessLevel(gitlab.NoPermissions),
		MergeAccessLevel:          gitlab.AccessLevel(gitlab.MaintainerPermissions),
		CodeOwnerApprovalRequired: gitlab.Bool(false),
	}

	masterOpt := &gitlab.ProtectRepositoryBranchesOptions{
		Name:                      gitlab.String("test-master"),
		PushAccessLevel:           gitlab.AccessLevel(gitlab.NoPermissions),
		MergeAccessLevel:          gitlab.AccessLevel(gitlab.MaintainerPermissions),
		CodeOwnerApprovalRequired: gitlab.Bool(true),
	}

	rb.git.ProtectedBranches.ProtectRepositoryBranches(290, masterOpt)
	rb.git.ProtectedBranches.ProtectRepositoryBranches(290, developOpt)
}

func (rb *ReleaseBuilder) getPipeline() {
	fmt.Println("getting pipeline Id")

	refName := "test-" + rb.packageVersion
	opt := &gitlab.ListProjectPipelinesOptions{
		Ref:  gitlab.String(refName),
		Name: gitlab.String("bo.dai"),
	}

	for {
		pipelines, _, err := rb.git.Pipelines.ListProjectPipelines(290, opt)
		if err != nil {
			fmt.Println("pipeline not found")
		}

		if len(pipelines) > 0 {
			rb.pipelineId = pipelines[0].ID
			break
		}

		time.Sleep(3 * time.Second)
	}
}

func (rb *ReleaseBuilder) getJob() {
	fmt.Println("getting jobId -- ", rb.pipelineId)

	opt := &gitlab.ListJobsOptions{
		Scope: []gitlab.BuildStateValue{gitlab.Running, gitlab.Pending, gitlab.Created},
	}

	jobs, _, _ := rb.git.Jobs.ListPipelineJobs(290, rb.pipelineId, opt)

	for _, job := range jobs {
		if strings.ToLower(job.Stage) == "prepare" {
			rb.jobId = job.ID
			break
		}
	}
}

func (rb *ReleaseBuilder) getArtifact() {
	fmt.Println("getting artifact")

	for {
		job, _, err := rb.git.Jobs.GetJob(290, rb.jobId)

		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(job.Status)

		if job.Status == "success" {
			reader, _, _ := rb.git.Jobs.DownloadSingleArtifactsFile(290, rb.jobId, "dist/build-info.json")
			result := &model.Artifact{}

			buf := new(bytes.Buffer)
			io.Copy(buf, reader)
			json.Unmarshal(buf.Bytes(), result)

			rb.releaseVersion = result.Version
			break
		}

		time.Sleep(10 * time.Second)
	}
}
