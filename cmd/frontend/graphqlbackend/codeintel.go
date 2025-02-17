package graphqlbackend

import (
	"context"
	"errors"

	"github.com/graph-gophers/graphql-go"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type CodeIntelResolver interface {
	LSIFUploadByID(ctx context.Context, id graphql.ID) (LSIFUploadResolver, error)
	LSIFUploads(ctx context.Context, args *LSIFUploadsQueryArgs) (LSIFUploadConnectionResolver, error)
	LSIFUploadsByRepo(ctx context.Context, args *LSIFRepositoryUploadsQueryArgs) (LSIFUploadConnectionResolver, error)
	DeleteLSIFUpload(ctx context.Context, id graphql.ID) (*EmptyResponse, error)
	LSIFIndexByID(ctx context.Context, id graphql.ID) (LSIFIndexResolver, error)
	LSIFIndexes(ctx context.Context, args *LSIFIndexesQueryArgs) (LSIFIndexConnectionResolver, error)
	LSIFIndexesByRepo(ctx context.Context, args *LSIFRepositoryIndexesQueryArgs) (LSIFIndexConnectionResolver, error)
	DeleteLSIFIndex(ctx context.Context, id graphql.ID) (*EmptyResponse, error)
	IndexConfiguration(ctx context.Context, id graphql.ID) (IndexConfigurationResolver, error) // TODO - rename ...ForRepo
	UpdateRepositoryIndexConfiguration(ctx context.Context, args *UpdateRepositoryIndexConfigurationArgs) (*EmptyResponse, error)
	QueueAutoIndexJobForRepo(ctx context.Context, id graphql.ID) (*EmptyResponse, error)
	GitBlobLSIFData(ctx context.Context, args *GitBlobLSIFDataArgs) (GitBlobLSIFDataResolver, error)
}

var codeIntelOnlyInEnterprise = errors.New("precise code intelligence only available in enterprise")

type defaultCodeIntelResolver struct{}

var DefaultCodeIntelResolver CodeIntelResolver = defaultCodeIntelResolver{}

func (defaultCodeIntelResolver) LSIFUploadByID(ctx context.Context, id graphql.ID) (LSIFUploadResolver, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) LSIFUploads(ctx context.Context, args *LSIFUploadsQueryArgs) (LSIFUploadConnectionResolver, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) LSIFUploadsByRepo(ctx context.Context, args *LSIFRepositoryUploadsQueryArgs) (LSIFUploadConnectionResolver, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) DeleteLSIFUpload(ctx context.Context, id graphql.ID) (*EmptyResponse, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) LSIFIndexByID(ctx context.Context, id graphql.ID) (LSIFIndexResolver, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) LSIFIndexes(ctx context.Context, args *LSIFIndexesQueryArgs) (LSIFIndexConnectionResolver, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) LSIFIndexesByRepo(ctx context.Context, args *LSIFRepositoryIndexesQueryArgs) (LSIFIndexConnectionResolver, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) DeleteLSIFIndex(ctx context.Context, id graphql.ID) (*EmptyResponse, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) IndexConfiguration(ctx context.Context, id graphql.ID) (IndexConfigurationResolver, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) UpdateRepositoryIndexConfiguration(ctx context.Context, args *UpdateRepositoryIndexConfigurationArgs) (*EmptyResponse, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) QueueAutoIndexJobForRepo(ctx context.Context, id graphql.ID) (*EmptyResponse, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (defaultCodeIntelResolver) GitBlobLSIFData(ctx context.Context, args *GitBlobLSIFDataArgs) (GitBlobLSIFDataResolver, error) {
	return nil, codeIntelOnlyInEnterprise
}

func (r *schemaResolver) LSIFUploads(ctx context.Context, args *LSIFUploadsQueryArgs) (LSIFUploadConnectionResolver, error) {
	return r.CodeIntelResolver.LSIFUploads(ctx, args)
}

func (r *schemaResolver) LSIFIndexes(ctx context.Context, args *LSIFIndexesQueryArgs) (LSIFIndexConnectionResolver, error) {
	return r.CodeIntelResolver.LSIFIndexes(ctx, args)
}

func (r *schemaResolver) DeleteLSIFUpload(ctx context.Context, args *struct{ ID graphql.ID }) (*EmptyResponse, error) {
	return r.CodeIntelResolver.DeleteLSIFUpload(ctx, args.ID)
}

func (r *schemaResolver) DeleteLSIFIndex(ctx context.Context, args *struct{ ID graphql.ID }) (*EmptyResponse, error) {
	return r.CodeIntelResolver.DeleteLSIFIndex(ctx, args.ID)
}

func (r *schemaResolver) UpdateRepositoryIndexConfigurationArgs(ctx context.Context, args *UpdateRepositoryIndexConfigurationArgs) (*EmptyResponse, error) {
	return r.CodeIntelResolver.UpdateRepositoryIndexConfiguration(ctx, args)
}

func (r *schemaResolver) QueueAutoIndexJobForRepo(ctx context.Context, args *struct{ Repository graphql.ID }) (*EmptyResponse, error) {
	return r.CodeIntelResolver.QueueAutoIndexJobForRepo(ctx, args.Repository)
}

type LSIFUploadsQueryArgs struct {
	graphqlutil.ConnectionArgs
	Query           *string
	State           *string
	IsLatestForRepo *bool
	After           *string
}

type LSIFRepositoryUploadsQueryArgs struct {
	*LSIFUploadsQueryArgs
	RepositoryID graphql.ID
}

type LSIFUploadResolver interface {
	ID() graphql.ID
	InputCommit() string
	InputRoot() string
	IsLatestForRepo() bool
	UploadedAt() DateTime
	State() string
	Failure() *string
	StartedAt() *DateTime
	FinishedAt() *DateTime
	InputIndexer() string
	PlaceInQueue() *int32
	ProjectRoot(ctx context.Context) (*GitTreeEntryResolver, error)
}

type LSIFUploadConnectionResolver interface {
	Nodes(ctx context.Context) ([]LSIFUploadResolver, error)
	TotalCount(ctx context.Context) (*int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type LSIFIndexesQueryArgs struct {
	graphqlutil.ConnectionArgs
	Query *string
	State *string
	After *string
}

type LSIFRepositoryIndexesQueryArgs struct {
	*LSIFIndexesQueryArgs
	RepositoryID graphql.ID
}

type LSIFIndexResolver interface {
	ID() graphql.ID
	InputCommit() string
	InputRoot() string
	InputIndexer() string
	QueuedAt() DateTime
	State() string
	Failure() *string
	StartedAt() *DateTime
	FinishedAt() *DateTime
	Steps() IndexStepsResolver
	PlaceInQueue() *int32
	ProjectRoot(ctx context.Context) (*GitTreeEntryResolver, error)
}

type IndexStepsResolver interface {
	Setup() []ExecutionLogEntryResolver
	PreIndex() []PreIndexStepResolver
	Index() IndexStepResolver
	Upload() ExecutionLogEntryResolver
	Teardown() []ExecutionLogEntryResolver
}

type PreIndexStepResolver interface {
	Root() string
	Image() string
	Commands() []string
	LogEntry() ExecutionLogEntryResolver
}

type IndexStepResolver interface {
	IndexerArgs() []string
	Outfile() *string
	LogEntry() ExecutionLogEntryResolver
}

type ExecutionLogEntryResolver interface {
	Key() string
	Command() []string
	StartTime() DateTime
	ExitCode() int32
	Out(ctx context.Context) (string, error)
	DurationMilliseconds() int32
}

type LSIFIndexConnectionResolver interface {
	Nodes(ctx context.Context) ([]LSIFIndexResolver, error)
	TotalCount(ctx context.Context) (*int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type IndexConfigurationResolver interface {
	Configuration() *string
}

type UpdateRepositoryIndexConfigurationArgs struct {
	Repository    graphql.ID
	Configuration string
}

type QueueAutoIndexJobArgs struct {
	Repository graphql.ID
}

type GitTreeLSIFDataResolver interface {
	Diagnostics(ctx context.Context, args *LSIFDiagnosticsArgs) (DiagnosticConnectionResolver, error)
}

type GitBlobLSIFDataResolver interface {
	GitTreeLSIFDataResolver
	ToGitTreeLSIFData() (GitTreeLSIFDataResolver, bool)
	ToGitBlobLSIFData() (GitBlobLSIFDataResolver, bool)

	Ranges(ctx context.Context, args *LSIFRangesArgs) (CodeIntelligenceRangeConnectionResolver, error)
	Definitions(ctx context.Context, args *LSIFQueryPositionArgs) (LocationConnectionResolver, error)
	References(ctx context.Context, args *LSIFPagedQueryPositionArgs) (LocationConnectionResolver, error)
	Hover(ctx context.Context, args *LSIFQueryPositionArgs) (HoverResolver, error)
}

type GitBlobLSIFDataArgs struct {
	Repo      *types.Repo
	Commit    api.CommitID
	Path      string
	ExactPath bool
	ToolName  string
}

type LSIFRangesArgs struct {
	StartLine int32
	EndLine   int32
}

type LSIFQueryPositionArgs struct {
	Line      int32
	Character int32
}

type LSIFPagedQueryPositionArgs struct {
	LSIFQueryPositionArgs
	graphqlutil.ConnectionArgs
	After *string
}

type LSIFDiagnosticsArgs struct {
	graphqlutil.ConnectionArgs
}

type CodeIntelligenceRangeConnectionResolver interface {
	Nodes(ctx context.Context) ([]CodeIntelligenceRangeResolver, error)
}

type CodeIntelligenceRangeResolver interface {
	Range(ctx context.Context) (RangeResolver, error)
	Definitions(ctx context.Context) (LocationConnectionResolver, error)
	References(ctx context.Context) (LocationConnectionResolver, error)
	Hover(ctx context.Context) (HoverResolver, error)
}

type LocationConnectionResolver interface {
	Nodes(ctx context.Context) ([]LocationResolver, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type HoverResolver interface {
	Markdown() Markdown
	Range() RangeResolver
}

type DiagnosticConnectionResolver interface {
	Nodes(ctx context.Context) ([]DiagnosticResolver, error)
	TotalCount(ctx context.Context) (int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type DiagnosticResolver interface {
	Severity() (*string, error)
	Code() (*string, error)
	Source() (*string, error)
	Message() (*string, error)
	Location(ctx context.Context) (LocationResolver, error)
}
