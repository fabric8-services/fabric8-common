package codebase

import (
	"errors"
	"fmt"
	"regexp"
)

type Scheme string

const (
	HTTPS Scheme = "https"
)

// There is lot with Git URL - supported scheme, format and validation (regex). Currently, GitURL support very specific conversion.
// reference:
// https://mirrors.edge.kernel.org/pub/software/scm/git/docs/git-clone.html#_git_urls_a_id_urls_a
// https://github.com/git/git/blob/77bd3ea9f54f1584147b594abc04c26ca516d987/urlmatch.c
// https://stackoverflow.com/questions/2514859/regular-expression-for-git-repository
// https://git-scm.com/book/en/v2/Git-on-the-Server-The-Protocols
type GitURL struct {
	url    string
	scheme Scheme
	host   string
	user   string
	repo   string
}

func NewGitURL(url string) (*GitURL, error) {
	if url == "" {
		return nil, errors.New("invalid URL, URL is blank")
	}

	pattern := regexp.MustCompile(`^(https|http|git)(:\/\/|@)([^\/:]+)[\/:]([^\/:]+)\/(.+).git$`)
	if !pattern.MatchString(url) {
		return nil, fmt.Errorf("invalid URL, %v", url)
	}
	components := pattern.FindStringSubmatch(url)
	l := len(components)

	gitURL := &GitURL{url: url}
	gitURL.scheme = Scheme(components[1])
	gitURL.host = components[l-3]
	gitURL.user = components[l-2]
	gitURL.repo = components[l-1]
	return gitURL, nil
}

func (url *GitURL) Convert(target Scheme) (string, error) {
	if target == "" {
		return "", nil
	}
	if target == url.scheme {
		return url.url, nil
	}

	switch target {
	case HTTPS:
		return toHTTPS(url), nil
	default:
		return "", errors.New("conversion not supported.")
	}
}

func toHTTPS(gitURL *GitURL) string {
	return fmt.Sprintf("%s://%s/%s/%s.git", HTTPS, gitURL.host, gitURL.user, gitURL.repo)
}
