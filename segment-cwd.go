package main

import (
	"os"
	"strings"
	"path/filepath"
)

const ellipsis = "\u2026"

type pathSegment struct {
	path     string
	home     bool
	root     bool
	ellipsis bool
}

func exists(elem ...string) bool {
	_, err := os.Stat(filepath.Join(elem...))
	return err == nil
}

func searchGitRoot(cwd string) (string, bool) {
	for !(cwd == "/" || cwd == "") {
		if exists(cwd, ".git") {
			return cwd, true
		}
		cwd = filepath.Dir(cwd)
	}
	return "", false
}

func cwdToPathSegments(cwd string) []pathSegment {
	pathSegments := make([]pathSegment, 0)
	gitRoot, _ := searchGitRoot(cwd)

	skip := 0
	joined := false
	gopath, _ := os.LookupEnv("GOPATH")
	home, _ := os.LookupEnv("HOME")
	if gopath != "" && strings.HasPrefix(cwd, gopath+"/src") {
		pathSegments = append(pathSegments, pathSegment{
			path: "🐭", // go
			home: true,
		})
		cwd = cwd[len(gopath+"/src"):]
		skip = 2
		joined = true
	} else if gitRoot != "" {
		icon := "Git" // github
		if exists(gitRoot, "package.json") {
			icon = "JS" // javascript
		} else if exists(gitRoot, "composer.json") {
			icon = "🐘" // php
		} else if exists(gitRoot, "Gemfile") {
			icon = "💎" // ruby
		} else if exists(gitRoot, "cpanfile") {
			icon = "🐪" // perl
		} else if exists(gitRoot, "__pycache__") {
			icon = "Py" // python
		}
		pathSegments = append(pathSegments, pathSegment{
			path: icon,
			home: true,
		})
		cwd = cwd[len(filepath.Dir(gitRoot)):]
		joined = true
	} else if strings.HasPrefix(cwd, home) {
		pathSegments = append(pathSegments, pathSegment{
			path: "~",
			home: true,
		})
		cwd = cwd[len(home):]
	} else if cwd == "/" {
		pathSegments = append(pathSegments, pathSegment{
			path: "/",
			root: true,
		})
	}

	cwd = strings.Trim(cwd, "/")
	names := strings.Split(cwd, "/")
	if names[0] == "" {
		names = names[1:]
	}

	l := len(names)
	if skip < l {
		names = names[skip:]
	} else if 2 <= l {
		names = names[l-1:l]
	}

	if 0 < len(names) {
		if joined {
			pathSegments = append(pathSegments, pathSegment{
				path: strings.Join(names, "/"),
			})
		} else {
			for _, name := range names {
				pathSegments = append(pathSegments, pathSegment{
					path: name,
				})
			}
		}
	}

	return pathSegments
}

func maybeShortenName(p *powerline, pathSegment string) string {
	if *p.args.CwdMaxDirSize > 0 && len(pathSegment) > *p.args.CwdMaxDirSize {
		return pathSegment[:*p.args.CwdMaxDirSize]
	} else {
		return pathSegment
	}
}

func escapeVariables(p *powerline, pathSegment string) string {
	pathSegment = strings.Replace(pathSegment, `\`, p.shellInfo.escapedBackslash, -1)
	pathSegment = strings.Replace(pathSegment, "`", p.shellInfo.escapedBacktick, -1)
	pathSegment = strings.Replace(pathSegment, `$`, p.shellInfo.escapedDollar, -1)
	return pathSegment
}

func getColor(p *powerline, pathSegment pathSegment, isLastDir bool) (uint8, uint8) {
	if pathSegment.home && p.theme.HomeSpecialDisplay {
		return p.theme.HomeFg, p.theme.HomeBg
	} else if isLastDir {
		return p.theme.CwdFg, p.theme.PathBg
	} else {
		return p.theme.PathFg, p.theme.PathBg
	}
}

func segmentCwd(p *powerline) {
	cwd := p.cwd
	if cwd == "" {
		cwd, _ = os.LookupEnv("PWD")
	}

	if *p.args.CwdMode == "plain" {
		home, _ := os.LookupEnv("HOME")
		if strings.HasPrefix(cwd, home) {
			cwd = "~" + cwd[len(home):]
		}

		p.appendSegment("cwd", segment{
			content:    cwd,
			foreground: p.theme.CwdFg,
			background: p.theme.PathBg,
		})
	} else {
		pathSegments := cwdToPathSegments(cwd)

		if *p.args.CwdMode == "dironly" {
			pathSegments = pathSegments[len(pathSegments)-1:]
		} else {
			maxDepth := *p.args.CwdMaxDepth
			if maxDepth <= 0 {
				warn("Ignoring -cwd-max-depth argument since it's smaller than or equal to 0")
			} else if len(pathSegments) > maxDepth {
				var nBefore int
				if maxDepth > 2 {
					nBefore = 2
				} else {
					nBefore = maxDepth - 1
				}
				firstPart := pathSegments[:nBefore]
				secondPart := pathSegments[len(pathSegments)+nBefore-maxDepth:]

				pathSegments = make([]pathSegment, 0)
				for _, segment := range firstPart {
					pathSegments = append(pathSegments, segment)
				}
				pathSegments = append(pathSegments, pathSegment{
					path:     ellipsis,
					ellipsis: true,
				})
				pathSegments = append(pathSegments, secondPart...)
			}

			for idx, pathSegment := range pathSegments {
				isLastDir := idx == len(pathSegments)-1
				foreground, background := getColor(p, pathSegment, isLastDir)

				segment := segment{
					content:    escapeVariables(p, maybeShortenName(p, pathSegment.path)),
					foreground: foreground,
					background: background,
				}

				if !(pathSegment.home && p.theme.HomeSpecialDisplay) && !isLastDir {
					segment.separator = p.symbolTemplates.SeparatorThin
					segment.separatorForeground = p.theme.SeparatorFg
				}

				origin := "cwd-path"
				if isLastDir {
					origin = "cwd"
				}

				p.appendSegment(origin, segment)
			}
		}
	}
}
