// Code generated by spicegen. DO NOT EDIT
package authz

type ResourceType string

const (
	Document     ResourceType = "document"
	User         ResourceType = "user"
	Organization ResourceType = "organization"
)

type Resource interface {
	ResourceType() ResourceType
	ID() string
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