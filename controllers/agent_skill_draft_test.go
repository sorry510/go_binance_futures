package controllers

import "testing"

func TestDraftUploadDefaultPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "helper.sh", want: "helper.sh"},
		{name: "windows fakepath", in: `C:\\fakepath\\helper.sh`, want: "helper.sh"},
		{name: "unix absolute", in: "/tmp/helper.sh", want: "helper.sh"},
		{name: "relative traversal filename", in: "../../helper.sh", want: "helper.sh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := draftUploadDefaultPath(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
	for _, in := range []string{"", ".", "..", "   "} {
		if _, err := draftUploadDefaultPath(in); err == nil {
			t.Fatalf("expected invalid upload filename %q to fail", in)
		}
	}
}
