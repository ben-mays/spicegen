// Code generated by spicegen. DO NOT EDIT
package authz

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/ben-mays/spicegen/examples/permissions/document"
	"github.com/ben-mays/spicegen/examples/permissions/organization"
)

// SpiceDBClient is the interface that the spicegen generated client wraps.
type SpiceDBClient interface {
	pb.PermissionsServiceClient
	pb.SchemaServiceClient
}

// Client is a SpiceDB client that can be used to check permissions on resources. It is safe for concurrent use. This client implements SpiceGenClient.
type Client struct {
	sync.RWMutex

	spicedbClient SpiceDBClient
	// Lock protects lastZedToken. Updated whenever a write occurs to provide read-my-write semantics.
	lastZedToken string
}

func NewClient(spicedbClient SpiceDBClient) SpiceGenClient {
	return &Client{
		spicedbClient: spicedbClient,
	}
}

func (c *Client) getConsistency() *pb.Consistency {
	return &pb.Consistency{Requirement: &pb.Consistency_AtLeastAsFresh{AtLeastAsFresh: &pb.ZedToken{Token: c.lastZedToken}}}
}

func (c *Client) CheckPermission(ctx context.Context, subject Resource, permission string, resource Resource, opts *CheckPermissionOptions) (bool, error) {
	c.RLock()
	defer c.RUnlock()
	var consistency *pb.Consistency
	var context *structpb.Struct
	if opts != nil {
		consistency = opts.Consistency
		context = opts.Context
	}
	if consistency == nil {
		if c.lastZedToken != "" {
			consistency = c.getConsistency()
		}
	}
	resp, err := c.spicedbClient.CheckPermission(ctx, &pb.CheckPermissionRequest{
		Consistency: consistency,
		Context:     context,
		Subject: &pb.SubjectReference{
			Object: &pb.ObjectReference{ObjectType: string(subject.ResourceType()), ObjectId: subject.ID()},
		},
		Permission: permission,
		Resource:   &pb.ObjectReference{ObjectType: string(resource.ResourceType()), ObjectId: resource.ID()},
	})
	if err != nil {
		return false, err
	}
	return resp.Permissionship == pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION, nil
}

func (c *Client) CheckOrganizationPermission(ctx context.Context, subject UserResource, permission organization.OrganizationPermission, resource OrganizationResource, opts *CheckPermissionOptions) (bool, error) {
	if organization.ALLOWED_PERMISSION_SUBJECT_TYPES[permission][string(subject.ResourceType())] || organization.ALLOWED_PERMISSION_SUBJECT_TYPES[permission]["*"] {
		return c.CheckPermission(ctx, subject, string(permission), resource, opts)
	} else {
		return false, errors.New(fmt.Sprintf("subject type not allowed for permission %s", string(permission)))
	}
}

func (c *Client) CheckDocumentPermission(ctx context.Context, subject Resource, permission document.DocumentPermission, resource DocumentResource, opts *CheckPermissionOptions) (bool, error) {
	if document.ALLOWED_PERMISSION_SUBJECT_TYPES[permission][string(subject.ResourceType())] || document.ALLOWED_PERMISSION_SUBJECT_TYPES[permission]["*"] {
		return c.CheckPermission(ctx, subject, string(permission), resource, opts)
	} else {
		return false, errors.New(fmt.Sprintf("subject type not allowed for permission %s", string(permission)))
	}
}

func (c *Client) AddRelationship(ctx context.Context, resource Resource, relation string, subject Resource, opts *AddRelationshipOptions) error {
	c.Lock()
	defer c.Unlock()
	var caveat *pb.ContextualizedCaveat
	if opts != nil {
		caveat = opts.Caveat
	}
	subjectRef := &pb.SubjectReference{
		Object: &pb.ObjectReference{
			ObjectType: string(subject.ResourceType()),
			ObjectId:   subject.ID(),
		},
	}
	if opts != nil && opts.OptionalSubjectRelation != "" {
		subjectRef.OptionalRelation = opts.OptionalSubjectRelation
	}
	resp, err := c.spicedbClient.WriteRelationships(ctx, &pb.WriteRelationshipsRequest{
		Updates: []*pb.RelationshipUpdate{{
			Operation: pb.RelationshipUpdate_OPERATION_TOUCH,
			Relationship: &pb.Relationship{
				Subject:  subjectRef,
				Relation: relation,
				Resource: &pb.ObjectReference{
					ObjectType: string(resource.ResourceType()),
					ObjectId:   resource.ID(),
				},
				OptionalCaveat: caveat,
			},
		}},
	})
	if err != nil {
		return err
	}
	c.lastZedToken = resp.WrittenAt.Token
	return nil
}

func (c *Client) AddOrganizationRelationship(ctx context.Context, resource OrganizationResource, relation organization.OrganizationRelation, subject UserResource, opts *AddRelationshipOptions) error {
	if organization.ALLOWED_RELATION_SUBJECT_TYPES[relation][string(subject.ResourceType())] || organization.ALLOWED_RELATION_SUBJECT_TYPES[relation]["*"] {
		return c.AddRelationship(ctx, resource, string(relation), subject, opts)
	} else {
		return errors.New(fmt.Sprintf("subject type not allowed for relation %s", string(relation)))
	}
}

func (c *Client) AddDocumentRelationship(ctx context.Context, resource DocumentResource, relation document.DocumentRelation, subject Resource, opts *AddRelationshipOptions) error {
	if document.ALLOWED_RELATION_SUBJECT_TYPES[relation][string(subject.ResourceType())] || document.ALLOWED_RELATION_SUBJECT_TYPES[relation]["*"] {
		return c.AddRelationship(ctx, resource, string(relation), subject, opts)
	} else {
		return errors.New(fmt.Sprintf("subject type not allowed for relation %s", string(relation)))
	}
}

func (c *Client) DeleteRelationship(ctx context.Context, resource Resource, relation string, subject Resource) error {
	c.Lock()
	defer c.Unlock()
	resp, err := c.spicedbClient.DeleteRelationships(ctx, &pb.DeleteRelationshipsRequest{
		RelationshipFilter: &pb.RelationshipFilter{ResourceType: string(resource.ResourceType()), OptionalResourceId: resource.ID(), OptionalRelation: relation, OptionalSubjectFilter: &pb.SubjectFilter{SubjectType: string(subject.ResourceType()), OptionalSubjectId: subject.ID()}},
	})
	if err != nil {
		return err
	}
	c.lastZedToken = resp.DeletedAt.Token
	return nil
}

func (c *Client) DeleteOrganizationRelationship(ctx context.Context, resource OrganizationResource, relation organization.OrganizationRelation, subject UserResource) error {
	return c.DeleteRelationship(ctx, resource, string(relation), subject)
}

func (c *Client) DeleteDocumentRelationship(ctx context.Context, resource DocumentResource, relation document.DocumentRelation, subject Resource) error {
	return c.DeleteRelationship(ctx, resource, string(relation), subject)
}

func (c *Client) LookupResources(ctx context.Context, resourceType ResourceType, subject Resource, permission string, opts *LookupResourcesOptions) ([]Resource, error) {
	c.RLock()
	defer c.RUnlock()
	subjectRef := &pb.SubjectReference{
		Object: &pb.ObjectReference{
			ObjectType: string(subject.ResourceType()),
			ObjectId:   subject.ID(),
		},
	}
	if opts != nil && opts.OptionalSubjectRelation != "" {
		subjectRef.OptionalRelation = opts.OptionalSubjectRelation
	}
	client, err := c.spicedbClient.LookupResources(ctx, &pb.LookupResourcesRequest{
		Consistency:        c.getConsistency(),
		ResourceObjectType: string(resourceType),
		Permission:         permission,
		Subject:            subjectRef,
	})
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0)
	for {
		resp, err := client.Recv()
		if resp != nil && resp.ResourceObjectId != "" {
			// suppress error because we _know_ this is a correct resource type given the generation
			resource, _ := NewResource(resourceType, resp.ResourceObjectId)
			resources = append(resources, resource)
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
	}
	return resources, nil
}

func (c *Client) LookupOrganizationResources(ctx context.Context, subject UserResource, permission organization.OrganizationPermission, opts *LookupResourcesOptions) ([]Resource, error) {
	return c.LookupResources(ctx, Organization, subject, string(permission), opts)
}

func (c *Client) LookupDocumentResources(ctx context.Context, subject Resource, permission document.DocumentPermission, opts *LookupResourcesOptions) ([]Resource, error) {
	return c.LookupResources(ctx, Document, subject, string(permission), opts)
}
