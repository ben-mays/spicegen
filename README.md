# spicegen
Generate strongly typed clients from spicedb schemas. See `_examples/` for a sample generated client.

```
Usage of spicegen:
  -client-name string
        Optional. The name of the client impl created by spicegen. (default "Client")
  -ignore-prefix string
        Optional. A prefix string to match against permission/relation names to ignore. Used to avoid exposing implicit permissions.
  -import-path string
        Required. The fully qualified module path for importing the generated client. e.x. github.com/ben-mays/spicegen/example
  -interface-name string
        Optional. The name of the client interface created by spicegen. (default "SpiceGenClient")
  -output-package string
        Optional. The package name of the generated client. This will default to the output directory name if not given.
  -output-path string
        Optional. The file or directory to which the generated client will be written. If a directory is given, the output filename will be client.go. If no output is given, current directory is used.
  -schema-file string
        Optional. Path to schema file for generation. If none given, the tool will look for schema.text in the current directory. (default "schema.text")
  -skip-client
        Optional. If present, will skip client generation and only generate types and permissions.
```

`spicegen` will generate a top-level `Resource` enum type that captures all object definitions in the schema.

```go
type ResourceType string

const (
	Organization ResourceType = "organization"
	Document     ResourceType = "document"
	User         ResourceType = "user"
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
```

For each resource type, we generate a set of enums for both permissions and relations (i.e. `DocPermission` and `DocRelation`) at `permissions/$resource_type/$resource_type.go` (i.e. `permissions/doc/doc.go`):

```go
type DocumentPermission string

const (
	ViewPermission DocumentPermission = "view"
)

type DocumentRelation string

const (
	DocorgRelation DocumentRelation = "docorg"
	ReaderRelation DocumentRelation = "reader"
	WriterRelation DocumentRelation = "writer"
)
```

These resource-specific types are then used by the top-level generated client to force inputs that match your schema:

```go
AddDocRelationship(ctx context.Context, 
   resource DocResource, 
   relation document.DocRelation, 
   subject Resource, 
   opts *AddRelationshipOptions) (bool, error)
```

And can be used with by wrapping inputs in the right resource types (in practice you would do this mapping in your datastore layer):

```go
// Add doc:readme to organization:foo
svc.AddDocRelationship(ctx, authz.NewDocResource("readme"), doc.DocorgRelation, authz.NewOrganizationResource("foo"), nil)

// Can org admin Ben view doc:readme? Expectation: yes
svc.CheckDocPermission(ctx, authz.NewDocResource("readme"), doc.ViewPermission, authz.NewUserResource("ben"), nil)
```

## Renaming generated relations

`spicegen` allows renaming a permission or relation using the `//spicegen:rename=$new_name` tag in a comment. This will only change the generated enum value, not the underlying schema string.

## Subject Types
The parser today is unable to determine subject types for indirect relations nor generate a union type to fit multiple subjects (i.e. user | team). Spicegen will enforce allowed types at runtime though. It will enforce optional subject relations as well.

You can override the spicegen inferred types by specifying `//spicegen:subject_type=$resource` comment(s) on the relation:

```json
definition document {
  ...
  /** view indicates whether the user can view the document */
  /** //spicegen:subject_type=user */
  permission view = reader + writer + docorg->view_all_documents
}
```

The above tag will result in the generator using `["user"]` as the allowed subject resource. If only one allowed subject type is present for an entire resource, spicegen will use that concrete subject resource type in the resource API.

## Example

```
go run cmd/spicegen/main.go -import-path github.com/ben-mays/spicegen/_examples -s _examples/schema.text -o _examples -op authz 
```

## TODO
* Support caveat types
* Auto mapping allowed types [Done?]
* Auto-add optional relations [Done]
* Functional opts
