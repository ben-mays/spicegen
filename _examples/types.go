// Code generated by spicegen. DO NOT EDIT
package authz

import (
	"context"
	"errors"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/ben-mays/spicegen/_examples/permissions/document"
	"github.com/ben-mays/spicegen/_examples/permissions/organization"
	"github.com/ben-mays/spicegen/_examples/permissions/team"
)

type ResourceType string

const (
	Team         ResourceType = "team"
	Organization ResourceType = "organization"
	Document     ResourceType = "document"
	User         ResourceType = "user"
)

type Resource interface {
	ResourceType() ResourceType
	ID() string
}

func NewResource(resourceType ResourceType, ID string) (Resource, error) {
	switch resourceType {

	case Team:
		return TeamResource{rid: ID}, nil

	case Organization:
		return OrganizationResource{rid: ID}, nil

	case Document:
		return DocumentResource{rid: ID}, nil

	case User:
		return UserResource{rid: ID}, nil

	}
	return nil, errors.New("resourceType given is not valid")
}

type TeamResource struct {
	rid string
}

func (r TeamResource) ID() string {
	return r.rid
}

func (r TeamResource) ResourceType() ResourceType {
	return Team
}

func NewTeamResource(ID string) TeamResource {
	return TeamResource{rid: ID}
}

type OrganizationResource struct {
	rid string
}

func (r OrganizationResource) ID() string {
	return r.rid
}

func (r OrganizationResource) ResourceType() ResourceType {
	return Organization
}

func NewOrganizationResource(ID string) OrganizationResource {
	return OrganizationResource{rid: ID}
}

type DocumentResource struct {
	rid string
}

func (r DocumentResource) ID() string {
	return r.rid
}

func (r DocumentResource) ResourceType() ResourceType {
	return Document
}

func NewDocumentResource(ID string) DocumentResource {
	return DocumentResource{rid: ID}
}

type UserResource struct {
	rid string
}

func (r UserResource) ID() string {
	return r.rid
}

func (r UserResource) ResourceType() ResourceType {
	return User
}

func NewUserResource(ID string) UserResource {
	return UserResource{rid: ID}
}

type SpiceGenClient interface {
	CheckOrganizationPermission(ctx context.Context, subject UserResource, permission organization.OrganizationPermission, resource OrganizationResource, opts *CheckPermissionOptions) (bool, error)
	CheckDocumentPermission(ctx context.Context, subject UserResource, permission document.DocumentPermission, resource DocumentResource, opts *CheckPermissionOptions) (bool, error)

	AddTeamRelationship(ctx context.Context, resource TeamResource, relation team.TeamRelation, subject Resource, opts *AddRelationshipOptions) error
	AddOrganizationRelationship(ctx context.Context, resource OrganizationResource, relation organization.OrganizationRelation, subject Resource, opts *AddRelationshipOptions) error
	AddDocumentRelationship(ctx context.Context, resource DocumentResource, relation document.DocumentRelation, subject Resource, opts *AddRelationshipOptions) error

	DeleteTeamRelationship(ctx context.Context, resource TeamResource, relation team.TeamRelation, subject Resource, opts *DeleteRelationshipOptions) error
	DeleteOrganizationRelationship(ctx context.Context, resource OrganizationResource, relation organization.OrganizationRelation, subject Resource, opts *DeleteRelationshipOptions) error
	DeleteDocumentRelationship(ctx context.Context, resource DocumentResource, relation document.DocumentRelation, subject Resource, opts *DeleteRelationshipOptions) error

	LookupOrganizationResources(ctx context.Context, subject UserResource, permission organization.OrganizationPermission, opts *LookupResourcesOptions) ([]Resource, string, error)
	LookupOrganizationSubjects(ctx context.Context, resourceID string, subjectType ResourceType, permission organization.OrganizationPermission, opts *LookupSubjectsOptions) ([]Resource, string, error)
	LookupDocumentResources(ctx context.Context, subject UserResource, permission document.DocumentPermission, opts *LookupResourcesOptions) ([]Resource, string, error)
	LookupDocumentSubjects(ctx context.Context, resourceID string, subjectType ResourceType, permission document.DocumentPermission, opts *LookupSubjectsOptions) ([]Resource, string, error)
}

type CheckPermissionOptions struct {
	Context *structpb.Struct
}

type AddRelationshipOptions struct {
	Caveat                  *pb.ContextualizedCaveat
	OptionalSubjectRelation string
}

type DeleteRelationshipOptions struct {
	Pagination              Pagination
	OptionalSubjectRelation string
}

type LookupResourcesOptions struct {
	Pagination              Pagination
	OptionalSubjectRelation string
}

type LookupSubjectsOptions struct {
	Pagination              Pagination
	OptionalSubjectRelation string
}

type Pagination struct {
	Limit int
	Token string
}
