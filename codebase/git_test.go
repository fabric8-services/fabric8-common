package codebase_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/codebase"
	"github.com/stretchr/testify/assert"
)

func TestGitURL(t *testing.T) {
	tables := []struct {
		url     string
		target  codebase.Scheme
		want    string
		wantErr bool
	}{
		{"git@github.com:testuser/testrepo.git", codebase.HTTPS, "https://github.com/testuser/testrepo.git", false},
		{"https://github.com/testuser/testrepo.git", codebase.HTTPS, "https://github.com/testuser/testrepo.git", false},
		{"git@github.com:testuser/testrepo", codebase.HTTPS, "", true},
		{"git@anything/testrepo", codebase.HTTPS, "", true},
		{"anything/testrepo", codebase.HTTPS, "", true},
		{"http://github.com/testuser/testrepo.git", codebase.HTTPS, "https://github.com/testuser/testrepo.git", false},
		{"http://anything/testrepo.git", codebase.HTTPS, "", true},
	}

	for _, table := range tables {
		gitURL, err := codebase.NewGitURL(table.url)
		if err != nil {
			assert.Equal(t, true, table.wantErr)
			continue
		}
		got, err := gitURL.Convert(codebase.HTTPS)
		assert.Equal(t, table.wantErr, (err != nil))
		assert.Equal(t, table.want, got)
	}
}
