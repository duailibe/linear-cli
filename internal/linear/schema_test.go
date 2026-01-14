package linear

import "testing"

func TestSchemaArgBaseType(t *testing.T) {
	cache := &schemaCache{
		QueryType: schemaTypeInfo{Fields: []schemaField{
			{
				Name: "issue",
				Args: []schemaArg{
					{
						Name: "id",
						Type: schemaGQLType{Kind: "NON_NULL", OfType: &schemaGQLType{Kind: "SCALAR", Name: "String"}},
					},
				},
			},
		}},
	}

	name, ok := cache.argBaseType("issue", "id")
	if !ok {
		t.Fatalf("expected arg type")
	}
	if name != "String" {
		t.Fatalf("expected String, got %s", name)
	}
}
