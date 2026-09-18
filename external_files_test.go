package sdk

import "testing"

func TestExternalFileCapability(t *testing.T) {
	client := NewClient(func(method string, params, result any) error {
		q := params.(ExternalFileRequest)
		if method != "external.files.read" || q.Source != "approved" || q.Path != "a.go" || q.Start != 2 || q.End != 4 {
			t.Fatalf("unexpected request %s %+v", method, q)
		}
		*result.(*ExternalFile) = ExternalFile{Content: "a\nb\nc", Start: 2}
		return nil
	})
	got, err := client.ExternalFiles().Read(ExternalFileRequest{Source: "approved", Path: "a.go", Start: 2, End: 4})
	if err != nil || got.Start != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	if p, ok := PermissionFor("external.files.read"); !ok || p != "external:read" || !ValidPermission(p) {
		t.Fatal("missing external permission")
	}
}
